package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kunmi02/git-declutter/internal/engine"
	"github.com/kunmi02/git-declutter/internal/recovery"
	"github.com/kunmi02/git-declutter/internal/safety"
)

func ScanText(w io.Writer, result *engine.ScanResult) {
	fmt.Fprintln(w, "GitDeclutter")
	name := result.Repository.Name
	if name == "" {
		name = result.Repository.Path
	}
	fmt.Fprintf(w, "Repository: %s\n\n", name)
	if result.Refreshed {
		fmt.Fprintln(w, "Refreshing remote references...")
		fmt.Fprintln(w)
	}
	if result.AuthMessage.Connected && result.AuthMessage.Source != "" {
		fmt.Fprintln(w, result.AuthMessage.Source)
		fmt.Fprintln(w)
	} else if result.ProviderNote != "" {
		fmt.Fprintln(w, result.ProviderNote)
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Scanning %d local branches...\n\n", len(result.Branches))

	byStatus := map[safety.BranchStatus][]safety.BranchAnalysis{}
	for _, a := range result.Branches {
		byStatus[a.Status] = append(byStatus[a.Status], a)
	}

	printGroup(w, "SAFE TO REMOVE", result.Summary.Safe, byStatus[safety.StatusSafe], "✓")
	printGroup(w, "NEEDS REVIEW", result.Summary.Review, byStatus[safety.StatusReview], "⚠")
	printGroup(w, "KEEP", result.Summary.Keep, byStatus[safety.StatusKeep], "✕")
	printProtected(w, result.Summary.Protected, byStatus[safety.StatusProtected])

	fmt.Fprintf(w, "%d safe · %d review · %d keep · %d protected\n",
		result.Summary.Safe, result.Summary.Review, result.Summary.Keep, result.Summary.Protected)
}

func printGroup(w io.Writer, title string, count int, items []safety.BranchAnalysis, mark string) {
	fmt.Fprintf(w, "%s%s%d\n\n", title, pad(title, 50), count)
	if len(items) == 0 {
		fmt.Fprintln(w)
		return
	}
	for _, a := range items {
		fmt.Fprintf(w, "%s %s\n", mark, a.Branch)
		if a.Summary != "" {
			fmt.Fprintf(w, "  %s\n", a.Summary)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func printProtected(w io.Writer, count int, items []safety.BranchAnalysis) {
	fmt.Fprintf(w, "%s%s%d\n\n", "PROTECTED", pad("PROTECTED", 50), count)
	for _, a := range items {
		label := a.Branch
		if a.HasReason(safety.ReasonBranchMatchesProtected) && strings.Contains(a.Summary, "*") {
			label = a.Summary
		}
		fmt.Fprintf(w, "🔒 %s\n", label)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func pad(title string, width int) string {
	n := width - len(title)
	if n < 2 {
		n = 2
	}
	return strings.Repeat(" ", n)
}

func ScanJSON(w io.Writer, result *engine.ScanResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func WhyText(w io.Writer, a safety.BranchAnalysis) {
	fmt.Fprintf(w, "%s\n\n", a.Branch)
	fmt.Fprintf(w, "Status: %s\n\n", a.Status.Label())

	if pr := a.Evidence.PullRequest; pr != nil {
		fmt.Fprintln(w, "Pull Request")
		fmt.Fprintf(w, "  PR #%d\n", pr.Number)
		fmt.Fprintf(w, "  State: %s\n", pr.State)
		if pr.BaseBranch != "" {
			fmt.Fprintf(w, "  Base: %s\n", pr.BaseBranch)
		}
		if pr.HeadSHA != "" {
			fmt.Fprintf(w, "  PR head: %s\n", short(pr.HeadSHA))
		}
		if pr.MergedAt != nil {
			fmt.Fprintf(w, "  Merged: %s\n", ago(*pr.MergedAt))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Local State")
	fmt.Fprintf(w, "  Local HEAD: %s\n", short(a.SHA))
	fmt.Fprintf(w, "  Remote branch: %s\n", remoteLabel(a.Evidence.RemoteState))
	fmt.Fprintf(w, "  Worktree: %s\n", yn(a.Evidence.Worktree))
	fmt.Fprintf(w, "  Protected: %s\n\n", yn(a.Evidence.Protected))

	fmt.Fprintln(w, "Safety Analysis")
	fmt.Fprintln(w)
	for _, d := range a.ReasonDetails {
		mark := "⚠"
		if d.Code.Positive() || a.Status == safety.StatusProtected && d.Code != safety.ReasonInsufficientEvidence {
			if a.Status == safety.StatusProtected {
				mark = "🔒"
			} else if d.Code.Positive() {
				mark = "✓"
			}
		}
		if a.Status == safety.StatusKeep && !d.Code.Positive() {
			mark = "⚠"
		}
		msg := d.Message
		if msg == "" {
			msg = string(d.Code)
		}
		fmt.Fprintf(w, "%s %s\n", mark, msg)
	}
	if len(a.ReasonDetails) == 0 {
		for _, r := range a.Reasons {
			fmt.Fprintf(w, "• %s\n", r)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Recommendation")
	fmt.Fprintln(w)
	fmt.Fprintln(w, a.Status.Label())
	fmt.Fprintln(w)
	switch a.Status {
	case safety.StatusSafe:
		fmt.Fprintln(w, "Safe to remove.")
	case safety.StatusKeep:
		fmt.Fprintln(w, "Deleting this branch could remove local work.")
	case safety.StatusReview:
		if a.HasReason(safety.ReasonRemoteBranchExists) {
			fmt.Fprintln(w, "The remote branch still exists. Not selected for automatic cleanup.")
		} else {
			fmt.Fprintln(w, "Evidence is incomplete. Not selected for automatic cleanup.")
		}
	case safety.StatusProtected:
		fmt.Fprintln(w, "This branch is excluded from cleanup.")
	}
}

func WhyJSON(w io.Writer, a safety.BranchAnalysis) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(a)
}

func HistoryText(w io.Writer, events []recovery.Event) {
	if len(events) == 0 {
		fmt.Fprintln(w, "No recoverable branches.")
		return
	}
	fmt.Fprintln(w, "Recoverable branches")
	fmt.Fprintln(w)
	for _, ev := range events {
		extra := ago(ev.DeletedAt)
		if ev.ExpiresAt != nil {
			extra += " · expires " + ago(*ev.ExpiresAt)
		}
		fmt.Fprintf(w, "%-28s deleted %s\n", ev.Branch, extra)
	}
}

func short(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}

func yn(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func remoteLabel(s safety.RemoteState) string {
	switch s {
	case safety.RemoteDeleted:
		return "deleted"
	case safety.RemoteExists:
		return "exists"
	default:
		return "unknown"
	}
}

func ago(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
		switch {
		case d < 48*time.Hour:
			return "in " + fmt.Sprintf("%d days", int(d.Hours()/24)+1)
		default:
			return "in " + fmt.Sprintf("%d days", int(d.Hours()/24))
		}
	}
	days := int(d.Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days == 1:
		return "1 day ago"
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}
