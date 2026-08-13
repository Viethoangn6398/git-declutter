package git

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

type Remote struct {
	Name  string
	URL   string
	Fetch bool
}

type RemoteProvider struct {
	Provider string
	Host     string
	Owner    string
	Repo     string
	URL      string
}

func (r *Repo) Remotes(ctx context.Context) ([]Remote, error) {
	out, err := r.runner.Git(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := map[string]Remote{}
	for _, line := range splitLines(out) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, raw := fields[0], fields[1]
		fetch := len(fields) >= 3 && fields[2] == "(fetch)"
		existing, ok := seen[name]
		if !ok {
			existing = Remote{Name: name, URL: raw}
		}
		if fetch {
			existing.Fetch = true
			existing.URL = raw
		}
		if existing.URL == "" {
			existing.URL = raw
		}
		seen[name] = existing
	}
	remotes := make([]Remote, 0, len(seen))
	for _, rem := range seen {
		remotes = append(remotes, rem)
	}
	return remotes, nil
}

func (r *Repo) PrimaryRemote(ctx context.Context) (Remote, error) {
	remotes, err := r.Remotes(ctx)
	if err != nil {
		return Remote{}, err
	}
	for _, rem := range remotes {
		if rem.Name == "origin" {
			return rem, nil
		}
	}
	if len(remotes) > 0 {
		return remotes[0], nil
	}
	return Remote{}, nil
}

var scpLike = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:]+):(?P<path>.+)$`)

func ParseRemoteURL(raw string) (RemoteProvider, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return RemoteProvider{}, false
	}

	if strings.HasPrefix(raw, "git@") || (!strings.Contains(raw, "://") && strings.Contains(raw, ":")) {
		m := scpLike.FindStringSubmatch(raw)
		if len(m) == 4 {
			return classify(m[2], m[3], raw)
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return RemoteProvider{}, false
	}
	host := u.Host
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h
	}
	path := strings.TrimPrefix(u.Path, "/")
	return classify(host, path, raw)
}

func classify(host, path, raw string) (RemoteProvider, bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" {
		return RemoteProvider{}, false
	}
	owner, repo, ok := splitOwnerRepo(path)
	if !ok {
		return RemoteProvider{}, false
	}
	provider := ""
	switch {
	case host == "github.com" || strings.Contains(host, "github"):
		provider = "github"
	case host == "gitlab.com" || strings.Contains(host, "gitlab"):
		provider = "gitlab"
	}
	if provider == "" {
		return RemoteProvider{}, false
	}
	return RemoteProvider{
		Provider: provider,
		Host:     host,
		Owner:    owner,
		Repo:     repo,
		URL:      raw,
	}, true
}

func splitOwnerRepo(path string) (string, string, bool) {
	i := strings.LastIndexByte(path, '/')
	if i <= 0 || i == len(path)-1 {
		return "", "", false
	}
	return path[:i], path[i+1:], true
}
