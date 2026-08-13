package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kunmi02/git-declutter/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	appVersion = "0.1.0-dev"
	jsonOut    bool
)

func Execute(version string) error {
	if version != "" {
		appVersion = version
	}
	return newRoot().Execute()
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "git-declutter",
		Short:         "Know what's safe to delete before you delete it.",
		Long:          "GitDeclutter analyzes local Git branches, remote state, and pull requests to determine which branches are truly safe to remove.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	root.AddCommand(
		newScanCmd(),
		newWhyCmd(),
		newCleanCmd(),
		newRestoreCmd(),
		newHistoryCmd(),
		newGCCmd(),
		newConfigCmd(),
		newVersionCmd(),
	)
	return root
}

func fail(code int, format string, args ...any) error {
	return exitcode.New(code, fmt.Errorf(format, args...))
}

func repoContext() context.Context {
	return context.Background()
}

func stdout() *os.File { return os.Stdout }
func stderr() *os.File { return os.Stderr }
