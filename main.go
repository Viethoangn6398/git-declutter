package main

import (
	"fmt"
	"os"

	"github.com/kunmi02/git-declutter/cmd"
	"github.com/kunmi02/git-declutter/internal/exitcode"
)

var version = "0.1.0-dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitcode.Code(err))
	}
}
