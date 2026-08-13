package cmd

import (
	"fmt"
	"os"

	"github.com/kunmi02/git-declutter/internal/exitcode"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/kunmi02/git-declutter/internal/output"
	"github.com/kunmi02/git-declutter/internal/recovery"
	"github.com/spf13/cobra"
)

func newRestoreCmd() *cobra.Command {
	var last bool
	cmd := &cobra.Command{
		Use:   "restore [branch]",
		Short: "Restore a branch removed by GitDeclutter",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := repoContext()
			repo, err := git.OpenFromCwd(ctx)
			if err != nil {
				return fail(exitcode.NotRepository, "Not a Git repository.\n\nRun GitDeclutter from inside a repository.")
			}
			store := recovery.Store{Repo: repo}
			_, _ = store.CleanupExpired(ctx)

			if last {
				ev, err := store.RestoreLast(ctx)
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout(), "Restored %s at %s.\n", ev.Branch, ev.SHA[:min(7, len(ev.SHA))])
				return nil
			}
			if len(args) == 1 {
				ev, err := store.Restore(ctx, args[0])
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout(), "Restored %s at %s.\n", ev.Branch, ev.SHA[:min(7, len(ev.SHA))])
				return nil
			}

			events, err := store.Active(ctx)
			if err != nil {
				return err
			}
			if len(events) == 0 {
				fmt.Fprintln(stdout(), "No recoverable branches.")
				return nil
			}
			output.HistoryText(stdout(), events)
			fmt.Fprintln(stdout())
			fmt.Fprintln(stdout(), "Run: git declutter restore <branch>")
			return nil
		},
	}
	cmd.Flags().BoolVar(&last, "last", false, "Restore the most recently deleted branch")
	return cmd
}

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List recoverable deleted branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := repoContext()
			repo, err := git.OpenFromCwd(ctx)
			if err != nil {
				return fail(exitcode.NotRepository, "Not a Git repository.\n\nRun GitDeclutter from inside a repository.")
			}
			store := recovery.Store{Repo: repo}
			_, _ = store.CleanupExpired(ctx)
			events, err := store.Active(ctx)
			if err != nil {
				return err
			}
			output.HistoryText(stdout(), events)
			return nil
		},
	}
}

func newGCCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Expire recovery refs that have passed their retention window",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := repoContext()
			repo, err := git.OpenFromCwd(ctx)
			if err != nil {
				return fail(exitcode.NotRepository, "Not a Git repository.\n\nRun GitDeclutter from inside a repository.")
			}
			store := recovery.Store{Repo: repo}
			if all {
				ok, err := confirmAll()
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(stdout(), "Aborted.")
					return nil
				}
				n, err := store.ClearAll(ctx)
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout(), "Cleared %d recovery record(s).\n", n)
				return nil
			}
			n, err := store.CleanupExpired(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout(), "Removed %d expired recovery record(s).\n", n)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Clear all recovery refs immediately (requires confirmation)")
	return cmd
}

func confirmAll() (bool, error) {
	fmt.Fprint(stdout(), "This will permanently drop all GitDeclutter recovery refs. Continue? [y/N] ")
	var s string
	_, _ = fmt.Fscanln(osStdinReader(), &s)
	return s == "y" || s == "Y" || s == "yes", nil
}

func osStdinReader() *os.File { return os.Stdin }
