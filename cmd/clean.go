package cmd

import (
	"fmt"
	"os"

	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/kunmi02/git-declutter/internal/engine"
	"github.com/kunmi02/git-declutter/internal/exitcode"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/safety"
	"github.com/kunmi02/git-declutter/internal/tui"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	var dryRun, safeOnly, yes, permanent, hard bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete branches classified as safe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if hard {
				permanent = true
			}
			ctx := repoContext()
			repo, err := git.OpenFromCwd(ctx)
			if err != nil {
				return fail(exitcode.NotRepository, "Not a Git repository.\n\nRun GitDeclutter from inside a repository.")
			}
			cfg, err := config.LoadWithRepo(repo.Path())
			if err != nil {
				return err
			}
			result, err := engine.Scan(ctx, repo, cfg, engine.ScanOptions{})
			if err != nil {
				return err
			}

			var selected []safety.BranchAnalysis
			for _, a := range result.Branches {
				if a.Status == safety.StatusSafe {
					selected = append(selected, a)
				}
			}

			if !yes && !dryRun {
				if !tui.IsInteractive() {
					return fail(exitcode.InvalidArgs, "refusing to clean without --yes in a non-interactive terminal")
				}
				retention := cfg.Recovery.Retention
				if permanent || !cfg.Recovery.Enabled {
					retention = "permanent (no GitDeclutter undo)"
				}
				picked, err := tui.SelectSafeBranches(os.Stdin, stdout(), result.Branches, retention)
				if err != nil {
					return err
				}
				if picked == nil {
					fmt.Fprintln(stdout(), "Aborted.")
					return nil
				}
				selected = picked
			}

			if len(selected) == 0 {
				fmt.Fprintln(stdout(), "No SAFE branches to clean.")
				return nil
			}

			out, err := engine.Clean(ctx, repo, cfg, selected, engine.CleanOptions{
				DryRun:    dryRun,
				SafeOnly:  true,
				Yes:       yes,
				Permanent: permanent,
			})
			if err != nil {
				return err
			}

			for _, s := range out.Skipped {
				fmt.Fprintf(stdout(), "Skipping %s:\n%s.\n\n", s.Name, s.Reason)
			}
			if dryRun {
				fmt.Fprintf(stdout(), "Dry run: %d branch(es) would be removed.\n", len(out.Deleted))
				for _, d := range out.Deleted {
					fmt.Fprintf(stdout(), "  %s\n", d.Name)
				}
				return nil
			}
			for _, d := range out.Deleted {
				if d.Permanent {
					fmt.Fprintf(stdout(), "Removed %s (permanent).\n", d.Name)
				} else {
					fmt.Fprintf(stdout(), "Removed %s (recoverable).\n", d.Name)
				}
			}
			if len(out.Failed) > 0 {
				for _, f := range out.Failed {
					fmt.Fprintf(stderr(), "Failed to remove %s: %s\n", f.Name, f.Error)
				}
				return fail(exitcode.PartialCleanup, "cleanup partially completed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Analyze and select, but make no changes")
	cmd.Flags().BoolVar(&safeOnly, "safe-only", false, "Only consider branches classified SAFE")
	cmd.Flags().BoolVar(&yes, "yes", false, "Do not prompt; delete SAFE branches")
	cmd.Flags().BoolVar(&permanent, "permanent", false, "Do not create GitDeclutter recovery refs")
	cmd.Flags().BoolVar(&hard, "hard", false, "Alias for --permanent")
	_ = safeOnly
	return cmd
}
