package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Runner interface {
	Git(ctx context.Context, args ...string) (string, error)
	GitAllowFail(ctx context.Context, args ...string) (string, int, error)
}

type ExecRunner struct {
	Dir string
}

func (r ExecRunner) Git(ctx context.Context, args ...string) (string, error) {
	out, code, err := r.GitAllowFail(ctx, args...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, fmt.Errorf("git %s: exit %d: %s", strings.Join(args, " "), code, strings.TrimSpace(out))
	}
	return out, nil
}

func (r ExecRunner) GitAllowFail(ctx context.Context, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := stdout.String()
	if stderr.Len() > 0 {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += stderr.String()
	}
	if err == nil {
		return stdout.String(), 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return combined, exitErr.ExitCode(), nil
	}
	return combined, -1, err
}

type Repo struct {
	WorkDir string
	runner  Runner
}

func Open(ctx context.Context, dir string) (*Repo, error) {
	r := &Repo{WorkDir: dir, runner: ExecRunner{Dir: dir}}
	out, err := r.runner.Git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}
	r.WorkDir = strings.TrimSpace(out)
	r.runner = ExecRunner{Dir: r.WorkDir}
	return r, nil
}

func OpenFromCwd(ctx context.Context) (*Repo, error) {
	return Open(ctx, "")
}

func (r *Repo) Git(ctx context.Context, args ...string) (string, error) {
	return r.runner.Git(ctx, args...)
}

func (r *Repo) GitAllowFail(ctx context.Context, args ...string) (string, int, error) {
	return r.runner.GitAllowFail(ctx, args...)
}

func (r *Repo) Path() string {
	return r.WorkDir
}

func (r *Repo) CommonDir(ctx context.Context) (string, error) {
	out, err := r.runner.Git(ctx, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", errors.New("empty git common dir")
	}
	return dir, nil
}

func (r *Repo) RevParse(ctx context.Context, rev string) (string, error) {
	out, err := r.runner.Git(ctx, "rev-parse", "--verify", "--quiet", rev)
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", fmt.Errorf("unknown revision %q", rev)
	}
	return sha, nil
}

func (r *Repo) FetchPrune(ctx context.Context) error {
	_, err := r.runner.Git(ctx, "fetch", "--prune")
	return err
}
