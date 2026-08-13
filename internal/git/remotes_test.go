package git

import "testing"

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		raw      string
		provider string
		host     string
		owner    string
		repo     string
		ok       bool
	}{
		{"git@github.com:org/repo.git", "github", "github.com", "org", "repo", true},
		{"https://github.com/org/repo.git", "github", "github.com", "org", "repo", true},
		{"https://github.com/org/repo", "github", "github.com", "org", "repo", true},
		{"git@gitlab.com:org/repo.git", "gitlab", "gitlab.com", "org", "repo", true},
		{"https://gitlab.com/org/repo.git", "gitlab", "gitlab.com", "org", "repo", true},
		{"https://gitlab.com/group/sub/repo.git", "gitlab", "gitlab.com", "group/sub", "repo", true},
		{"ssh://git@github.com/org/repo.git", "github", "github.com", "org", "repo", true},
		{"https://github.example.com/org/repo.git", "github", "github.example.com", "org", "repo", true},
		{"not-a-url", "", "", "", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseRemoteURL(tc.raw)
		if ok != tc.ok {
			t.Fatalf("%s: ok=%v want %v", tc.raw, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if got.Provider != tc.provider || got.Host != tc.host || got.Owner != tc.owner || got.Repo != tc.repo {
			t.Fatalf("%s: %+v", tc.raw, got)
		}
	}
}
