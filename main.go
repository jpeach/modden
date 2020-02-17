package main

import (
	"os"

	"github.com/jpeach/modden/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		// TODO(jpeach): fish the exit code out of the error type.
		os.Exit(cmd.EX_FAIL)
	}
}
