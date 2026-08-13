package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/providers"
	"github.com/kunmi02/git-declutter/internal/safety"
)

type Client struct {
	Host     string
	Owner    string
	Repo     string
	Token    string
	TokenSrc string
	HTTP     *http.Client
}

func New(remote git.RemoteProvider) *Client {
	token, src := Token()
	host := remote.Host
	if host == "" {
		host = "gitlab.com"
	}
	return &Client{
		Host:     host,
		Owner:    remote.Owner,
		Repo:     remote.Repo,
		Token:    token,
		TokenSrc: src,
		HTTP:     &http.Client{Timeout: 20 * time.Second},
	}
}

func Token() (string, string) {
	if t := strings.TrimSpace(os.Getenv("GITLAB_TOKEN")); t != "" {
		return t, "GITLAB_TOKEN"
	}
	out, err := exec.Command("glab", "auth", "token").Output()
	if err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, "glab"
		}
	}
	return "", ""
}

func (c *Client) Name() string { return "gitlab" }

func (c *Client) AuthMessage() providers.AuthMessage {
	if c.Token != "" {
		src := "GitLab connected"
		if c.TokenSrc == "glab" {
			src = "GitLab connected via glab."
		}
		return providers.AuthMessage{Connected: true, Provider: "gitlab", Source: src}
	}
	return providers.AuthMessage{
		Connected: false,
		Provider:  "gitlab",
		Warning:   "GitLab authentication unavailable.\nRunning with local Git analysis only.\n\nSome branches may be marked REVIEW.",
	}
}

func (c *Client) projectID() string {
	return url.PathEscape(c.Owner + "/" + c.Repo)
}

func (c *Client) apiBase() string {
	host := c.Host
	if host == "" {
		host = "gitlab.com"
	}
	return "https://" + host + "/api/v4"
}

func (c *Client) Repository(ctx context.Context) (*providers.Repository, error) {
	var raw struct {
		DefaultBranch string `json:"default_branch"`
		WebURL        string `json:"web_url"`
		Path          string `json:"path"`
		PathWithNS    string `json:"path_with_namespace"`
	}
	if err := c.get(ctx, "/projects/"+c.projectID(), &raw); err != nil {
		return nil, err
	}
	return &providers.Repository{
		Owner:         c.Owner,
		Name:          raw.Path,
		DefaultBranch: raw.DefaultBranch,
		HTMLURL:       raw.WebURL,
	}, nil
}

func (c *Client) BranchExists(ctx context.Context, branch string) (safety.RemoteState, error) {
	path := fmt.Sprintf("/projects/%s/repository/branches/%s", c.projectID(), url.PathEscape(branch))
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return safety.RemoteUnknown, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return safety.RemoteUnknown, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return safety.RemoteExists, nil
	case http.StatusNotFound:
		return safety.RemoteDeleted, nil
	default:
		return safety.RemoteUnknown, fmt.Errorf("gitlab branch lookup: HTTP %d", resp.StatusCode)
	}
}

func (c *Client) PullRequestsForBranch(ctx context.Context, branch string) ([]safety.PullRequest, error) {
	all, err := c.PullRequests(ctx)
	if err != nil {
		return nil, err
	}
	var matched []safety.PullRequest
	for _, pr := range all {
		if pr.HeadBranch == branch {
			matched = append(matched, pr)
		}
	}
	return matched, nil
}

func (c *Client) PullRequests(ctx context.Context) ([]safety.PullRequest, error) {
	var all []safety.PullRequest
	path := fmt.Sprintf("/projects/%s/merge_requests?state=all&per_page=100&order_by=updated_at&sort=desc", c.projectID())
	for page := 0; page < 10; page++ {
		var raw []glMR
		next, err := c.getList(ctx, path, &raw)
		if err != nil {
			return nil, err
		}
		for _, m := range raw {
			all = append(all, m.toPR())
		}
		if next == "" {
			break
		}
		path = next
	}
	return all, nil
}

type glMR struct {
	IID          int        `json:"iid"`
	Title        string     `json:"title"`
	State        string     `json:"state"`
	Draft        bool       `json:"draft"`
	WebURL       string     `json:"web_url"`
	SHA          string     `json:"sha"`
	MergedAt     *time.Time `json:"merged_at"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	WIP          bool       `json:"work_in_progress"`
}

func (m glMR) toPR() safety.PullRequest {
	merged := m.State == "merged" || m.MergedAt != nil
	state := safety.PRClosed
	draft := m.Draft || m.WIP
	switch {
	case merged:
		state = safety.PRMerged
	case m.State == "opened" && draft:
		state = safety.PRDraft
	case m.State == "opened":
		state = safety.PROpen
	}
	return safety.PullRequest{
		Number:     m.IID,
		State:      state,
		Title:      m.Title,
		HeadSHA:    m.SHA,
		HeadBranch: m.SourceBranch,
		BaseBranch: m.TargetBranch,
		MergedAt:   m.MergedAt,
		URL:        m.WebURL,
		Merged:     merged,
		Draft:      draft,
	}
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("gitlab authentication required (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gitlab API %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) getList(ctx context.Context, path string, dest any) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("gitlab authentication required (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gitlab API: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return "", err
	}
	return parseNextLink(resp.Header.Get("Link")), nil
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	full := path
	if strings.HasPrefix(path, "/") {
		full = c.apiBase() + path
	}
	req, err := http.NewRequestWithContext(ctx, method, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "git-declutter")
	if c.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.Token)
	}
	return req, nil
}

func parseNextLink(h string) string {
	for _, part := range strings.Split(h, ",") {
		part = strings.TrimSpace(part)
		if strings.HasSuffix(part, `rel="next"`) {
			start := strings.IndexByte(part, '<')
			end := strings.IndexByte(part, '>')
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}
