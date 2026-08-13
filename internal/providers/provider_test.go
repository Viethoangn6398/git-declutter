package providers

import (
	"testing"
	"time"

	"github.com/kunmi02/git-declutter/internal/safety"
)

func TestMostRecentMerged(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	got := MostRecentMerged([]safety.PullRequest{
		{Number: 1, State: safety.PRMerged, Merged: true, HeadSHA: "old", MergedAt: &old},
		{Number: 2, State: safety.PROpen, HeadSHA: "open"},
		{Number: 3, State: safety.PRMerged, Merged: true, HeadSHA: "new", MergedAt: &newer},
	})
	if got == nil || got.Number != 3 || got.HeadSHA != "new" {
		t.Fatalf("got %+v", got)
	}
}
