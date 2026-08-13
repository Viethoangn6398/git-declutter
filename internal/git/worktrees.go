package git

import (
	"context"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
	SHA    string
	Bare   bool
}

func (r *Repo) Worktrees(ctx context.Context) ([]Worktree, error) {
	out, err := r.runner.Git(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var trees []Worktree
	var cur Worktree
	flush := func() {
		if cur.Path != "" {
			trees = append(trees, cur)
		}
		cur = Worktree{}
	}
	for _, line := range splitLines(out) {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			cur.SHA = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "detached":
			cur.Branch = ""
		}
	}
	flush()
	return trees, nil
}

func WorktreeBranches(trees []Worktree) map[string]bool {
	out := make(map[string]bool)
	for _, t := range trees {
		if t.Branch != "" {
			out[t.Branch] = true
		}
	}
	return out
}
