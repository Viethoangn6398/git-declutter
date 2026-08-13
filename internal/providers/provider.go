package providers

import (
	"context"
	"time"

	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/safety"
)

type Repository struct {
	Owner         string
	Name          string
	DefaultBranch string
	HTMLURL       string
}

type AuthStatus int

const (
	AuthNone AuthStatus = iota
	AuthAvailable
	AuthRequired
)

type Provider interface {
	Name() string
	Repository(ctx context.Context) (*Repository, error)
	BranchExists(ctx context.Context, branch string) (safety.RemoteState, error)
	PullRequestsForBranch(ctx context.Context, branch string) ([]safety.PullRequest, error)
	PullRequests(ctx context.Context) ([]safety.PullRequest, error)
}

type AuthMessage struct {
	Connected bool
	Provider  string
	Source    string
	Warning   string
}

type CacheEntry struct {
	FetchedAt    time.Time            `json:"fetchedAt"`
	Repository   *Repository          `json:"repository,omitempty"`
	PullRequests []safety.PullRequest `json:"pullRequests,omitempty"`
}

func Detect(remote git.RemoteProvider) string {
	return remote.Provider
}

func MostRecentMerged(prs []safety.PullRequest) *safety.PullRequest {
	var best *safety.PullRequest
	for i := range prs {
		pr := &prs[i]
		if pr.State != safety.PRMerged && !pr.Merged {
			continue
		}
		if best == nil {
			best = pr
			continue
		}
		if pr.MergedAt != nil && (best.MergedAt == nil || pr.MergedAt.After(*best.MergedAt)) {
			best = pr
		} else if pr.MergedAt == nil && pr.Number > best.Number {
			best = pr
		}
	}
	if best == nil {
		return nil
	}
	cp := *best
	return &cp
}
