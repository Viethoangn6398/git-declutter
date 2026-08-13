package engine

import (
	"context"
	"fmt"

	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/recovery"
	"github.com/kunmi02/git-declutter/internal/safety"
)

type CleanOptions struct {
	DryRun    bool
	SafeOnly  bool
	Yes       bool
	Permanent bool
	Branches  []string
}

type CleanResult struct {
	Deleted []DeletedBranch
	Skipped []SkippedBranch
	Failed  []FailedBranch
}

type DeletedBranch struct {
	Name      string
	SHA       string
	Recovery  *recovery.Event
	Permanent bool
}

type SkippedBranch struct {
	Name   string
	Reason string
}

type FailedBranch struct {
	Name  string
	Error string
}

var ErrUnsafeDelete = fmt.Errorf("refusing to delete a branch that is not classified SAFE")

func Clean(ctx context.Context, repo *git.Repo, cfg config.Config, analyses []safety.BranchAnalysis, opts CleanOptions) (*CleanResult, error) {
	store := recovery.Store{Repo: repo}
	_, _ = store.CleanupExpired(ctx)

	selected := analyses
	if len(opts.Branches) > 0 {
		allow := map[string]bool{}
		for _, n := range opts.Branches {
			allow[n] = true
		}
		selected = selected[:0]
		for _, a := range analyses {
			if allow[a.Branch] {
				selected = append(selected, a)
			}
		}
	}

	out := &CleanResult{}
	for _, a := range selected {
		if opts.SafeOnly && a.Status != safety.StatusSafe {
			out.Skipped = append(out.Skipped, SkippedBranch{Name: a.Branch, Reason: "not classified SAFE"})
			continue
		}
		safe, ok := a.AsSafe()
		if !ok {
			out.Skipped = append(out.Skipped, SkippedBranch{
				Name:   a.Branch,
				Reason: fmt.Sprintf("status is %s", a.Status.Label()),
			})
			continue
		}
		if opts.DryRun {
			out.Deleted = append(out.Deleted, DeletedBranch{Name: a.Branch, SHA: a.SHA, Permanent: opts.Permanent})
			continue
		}
		deleted, skipped, err := deleteSafe(ctx, repo, store, cfg, safe, opts.Permanent)
		if err != nil {
			out.Failed = append(out.Failed, FailedBranch{Name: a.Branch, Error: err.Error()})
			continue
		}
		if skipped != nil {
			out.Skipped = append(out.Skipped, *skipped)
			continue
		}
		out.Deleted = append(out.Deleted, *deleted)
	}
	return out, nil
}

func deleteSafe(ctx context.Context, repo *git.Repo, store recovery.Store, cfg config.Config, sb safety.SafeBranch, permanent bool) (*DeletedBranch, *SkippedBranch, error) {
	a := sb.Analysis
	exists, err := repo.BranchExists(ctx, a.Branch)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, &SkippedBranch{Name: a.Branch, Reason: "branch no longer exists"}, nil
	}
	current, _ := repo.CurrentBranch(ctx)
	if current == a.Branch {
		return nil, &SkippedBranch{Name: a.Branch, Reason: "branch is currently checked out"}, nil
	}
	trees, err := repo.Worktrees(ctx)
	if err != nil {
		return nil, nil, err
	}
	if git.WorktreeBranches(trees)[a.Branch] {
		return nil, &SkippedBranch{Name: a.Branch, Reason: "branch is checked out in a worktree"}, nil
	}
	sha, err := repo.RevParse(ctx, "refs/heads/"+a.Branch)
	if err != nil {
		return nil, nil, err
	}
	if sha != a.SHA {
		return nil, &SkippedBranch{Name: a.Branch, Reason: "branch changed since it was analyzed"}, nil
	}

	reason := "safe"
	prNumber := 0
	if a.HasReason(safety.ReasonPullRequestMerged) {
		reason = "merged_pr"
	} else if a.HasReason(safety.ReasonMergedIntoTrunk) {
		reason = "merged_into_trunk"
	}
	if a.Evidence.PullRequest != nil {
		prNumber = a.Evidence.PullRequest.Number
	}

	var ev recovery.Event
	if permanent || !cfg.Recovery.Enabled {
		ev, err = store.CreatePermanent(ctx, a.Branch, sha, reason, prNumber)
		if err != nil {
			return nil, nil, err
		}
		ev.Permanent = true
	} else {
		ev, err = store.Create(ctx, a.Branch, sha, reason, prNumber, cfg)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := repo.DeleteBranch(ctx, a.Branch); err != nil {
		if !permanent && cfg.Recovery.Enabled {
			_ = repo.DeleteRef(ctx, recovery.RecoveryRef(ev.ID, a.Branch))
		}
		return nil, nil, err
	}

	return &DeletedBranch{
		Name:      a.Branch,
		SHA:       sha,
		Recovery:  &ev,
		Permanent: ev.Permanent,
	}, nil, nil
}

func Revalidate(ctx context.Context, repo *git.Repo, cfg config.Config, branch string, expectedSHA string) (safety.BranchAnalysis, error) {
	result, err := Scan(ctx, repo, cfg, ScanOptions{Branch: branch, Offline: false})
	if err != nil {
		return safety.BranchAnalysis{}, err
	}
	a, ok := FindAnalysis(result, branch)
	if !ok {
		return safety.BranchAnalysis{}, fmt.Errorf("branch %q not found", branch)
	}
	if expectedSHA != "" && a.SHA != expectedSHA {
		a.Status = safety.StatusReview
	}
	return a, nil
}
