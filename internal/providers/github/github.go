package github

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
		host = "github.com"
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
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t, "GITHUB_TOKEN"
	}
	if t := strings.TrimSpace(os.Getenv("GH_TOKEN")); t != "" {
		return t, "GH_TOKEN"
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, "gh"
		}
	}
	return "", ""
}

func (c *Client) Name() string { return "github" }

func (c *Client) AuthMessage() providers.AuthMessage {
	if c.Token != "" {
		src := "GitHub connected"
		if c.TokenSrc == "gh" {
			src = "GitHub connected via gh."
		}
		return providers.AuthMessage{Connected: true, Provider: "github", Source: src}
	}
	return providers.AuthMessage{
		Connected: false,
		Provider:  "github",
		Warning:   "GitHub authentication unavailable.\nRunning with local Git analysis only.\n\nSome branches may be marked REVIEW.",
	}
}

func (c *Client) apiBase() string {
	if c.Host == "github.com" || c.Host == "" {
		return "https://api.github.com"
	}
	return "https://" + c.Host + "/api/v3"
}

func (c *Client) Repository(ctx context.Context) (*providers.Repository, error) {
	var raw struct {
		DefaultBranch string `json:"default_branch"`
		HTMLURL       string `json:"html_url"`
		Name          string `json:"name"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s", c.Owner, c.Repo), &raw); err != nil {
		return nil, err
	}
	return &providers.Repository{
		Owner:         raw.Owner.Login,
		Name:          raw.Name,
		DefaultBranch: raw.DefaultBranch,
		HTMLURL:       raw.HTMLURL,
	}, nil
}

func (c *Client) BranchExists(ctx context.Context, branch string) (safety.RemoteState, error) {
	path := fmt.Sprintf("/repos/%s/%s/branches/%s", c.Owner, c.Repo, url.PathEscape(branch))
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
		return safety.RemoteUnknown, fmt.Errorf("github branch lookup: HTTP %d", resp.StatusCode)
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
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=all&per_page=100&sort=updated&direction=desc", c.Owner, c.Repo)
	for page := 0; page < 10; page++ {
		var raw []ghPR
		next, err := c.getList(ctx, path, &raw)
		if err != nil {
			return nil, err
		}
		for _, p := range raw {
			all = append(all, p.toPR())
		}
		if next == "" {
			break
		}
		path = next
	}
	return all, nil
}

type ghPR struct {
	Number   int        `json:"number"`
	State    string     `json:"state"`
	Title    string     `json:"title"`
	Draft    bool       `json:"draft"`
	HTMLURL  string     `json:"html_url"`
	Merged   bool       `json:"merged"`
	MergedAt *time.Time `json:"merged_at"`
	Head     struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (p ghPR) toPR() safety.PullRequest {
	state := safety.PRClosed
	merged := p.Merged || p.MergedAt != nil
	switch {
	case merged:
		state = safety.PRMerged
	case p.Draft && p.State == "open":
		state = safety.PRDraft
	case p.State == "open":
		state = safety.PROpen
	}
	return safety.PullRequest{
		Number:     p.Number,
		State:      state,
		Title:      p.Title,
		HeadSHA:    p.Head.SHA,
		HeadBranch: p.Head.Ref,
		BaseBranch: p.Base.Ref,
		MergedAt:   p.MergedAt,
		URL:        p.HTMLURL,
		Merged:     merged,
		Draft:      p.Draft,
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
		return fmt.Errorf("github authentication required (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github API %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) getList(ctx context.Context, path string, dest any) (next string, err error) {
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
		return "", fmt.Errorf("github authentication required (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("github API: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "git-declutter")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
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
