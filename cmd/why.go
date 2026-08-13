package cmd

import (
	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/kunmi02/git-declutter/internal/engine"
	"github.com/kunmi02/git-declutter/internal/exitcode"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/output"
	"github.com/spf13/cobra"
)

func newWhyCmd() *cobra.Command {
	var offline bool
	cmd := &cobra.Command{
		Use:   "why <branch>",
		Short: "Explain one branch's classification",
		Args:  cobra.ExactArgs(1),
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
			result, err := engine.Scan(ctx, repo, cfg, engine.ScanOptions{
				Branch:  args[0],
				Offline: offline,
			})
			if err != nil {
				return err
			}
			a, ok := engine.FindAnalysis(result, args[0])
			if !ok {
				return fail(exitcode.InvalidArgs, "branch %q not found", args[0])
			}
			if jsonOut {
				return output.WhyJSON(stdout(), a)
			}
			output.WhyText(stdout(), a)
			return nil
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "Do not contact providers or remotes")
	return cmd
}
