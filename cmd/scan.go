package cmd

import (
	"fmt"

	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/kunmi02/git-declutter/internal/engine"
	"github.com/kunmi02/git-declutter/internal/exitcode"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/output"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var offline, refresh bool
	var branch string
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Analyze local branches without deleting anything",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := repoContext()
			repo, err := git.OpenFromCwd(ctx)
			if err != nil {
				return fail(exitcode.NotRepository, "Not a Git repository.\n\nRun GitDeclutter from inside a repository.")
			}
			cfg, err := config.LoadWithRepo(repo.Path())
			if err != nil {
				return err
			}
			if refresh {
				fmt.Fprintln(stderr(), "Refreshing remote references...")
			}
			result, err := engine.Scan(ctx, repo, cfg, engine.ScanOptions{
				Offline: offline,
				Refresh: refresh,
				Branch:  branch,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return output.ScanJSON(stdout(), result)
			}
			output.ScanText(stdout(), result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "Do not contact providers or remotes")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Fetch and prune remote refs before analysis")
	cmd.Flags().StringVar(&branch, "branch", "", "Analyze a single branch")
	return cmd
}
