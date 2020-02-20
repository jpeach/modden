package main

import (
	"log"
	"os"

	"github.com/jpeach/modden/cmd"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := cmd.NewRootCommand().Execute(); err != nil {
		// TODO(jpeach): fish the exit code out of the error type.
		os.Exit(cmd.EX_FAIL)
	}
}
