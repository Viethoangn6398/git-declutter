package git

import (
	"context"
	"strings"

	"github.com/kunmi02/git-declutter/internal/safety"
)

type Branch struct {
	Name       string
	SHA        string
	Upstream   string
	RemoteName string
	IsCurrent  bool
}

func (r *Repo) Branches(ctx context.Context) ([]Branch, error) {
	format := "%(refname:short)%00%(objectname)%00%(upstream:short)%00%(HEAD)"
	out, err := r.runner.Git(ctx, "for-each-ref", "--format="+format, "refs/heads")
	if err != nil {
		return nil, err
	}
	lines := splitLines(out)
	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\x00")
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		sha := parts[1]
		upstream := parts[2]
		head := parts[3]
		b := Branch{
			Name:      name,
			SHA:       sha,
			Upstream:  upstream,
			IsCurrent: head == "*",
		}
		if upstream != "" {
			b.RemoteName = remoteFromUpstream(upstream)
		}
		branches = append(branches, b)
	}
	return branches, nil
}

func (r *Repo) CurrentBranch(ctx context.Context) (string, error) {
	out, code, err := r.runner.GitAllowFail(ctx, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) DefaultBranchFromRemoteHEAD(ctx context.Context, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	out, code, err := r.runner.GitAllowFail(ctx, "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	ref := strings.TrimSpace(out)
	ref = strings.TrimPrefix(ref, remote+"/")
	return ref, nil
}

func (r *Repo) BranchExists(ctx context.Context, name string) (bool, error) {
	_, code, err := r.runner.GitAllowFail(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

func (r *Repo) DeleteBranch(ctx context.Context, name string) error {
	_, err := r.runner.Git(ctx, "branch", "-D", "--", name)
	return err
}

func (r *Repo) CreateBranchAt(ctx context.Context, name, sha string) error {
	_, err := r.runner.Git(ctx, "branch", "--", name, sha)
	return err
}

func (r *Repo) CreateRef(ctx context.Context, ref, sha string) error {
	_, err := r.runner.Git(ctx, "update-ref", ref, sha)
	return err
}

func (r *Repo) DeleteRef(ctx context.Context, ref string) error {
	_, err := r.runner.Git(ctx, "update-ref", "-d", ref)
	return err
}

func (r *Repo) ListRefs(ctx context.Context, prefix string) ([]Ref, error) {
	out, err := r.runner.Git(ctx, "for-each-ref", "--format=%(refname)%00%(objectname)", prefix)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, line := range splitLines(out) {
		parts := strings.Split(line, "\x00")
		if len(parts) < 2 {
			continue
		}
		refs = append(refs, Ref{Name: parts[0], SHA: parts[1]})
	}
	return refs, nil
}

type Ref struct {
	Name string
	SHA  string
}

func remoteFromUpstream(upstream string) string {
	i := strings.IndexByte(upstream, '/')
	if i <= 0 {
		return ""
	}
	return upstream[:i]
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func (r *Repo) RemoteTrackingSHA(ctx context.Context, remote, branch string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	out, code, err := r.runner.GitAllowFail(ctx, "rev-parse", "--verify", "--quiet", "refs/remotes/"+remote+"/"+branch)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) RemoteBranchState(ctx context.Context, remote, branch string) (safety.RemoteState, error) {
	sha, err := r.RemoteTrackingSHA(ctx, remote, branch)
	if err != nil {
		return safety.RemoteUnknown, err
	}
	if sha != "" {
		return safety.RemoteExists, nil
	}
	return safety.RemoteDeleted, nil
}
