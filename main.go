package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jpeach/modden/cmd"
	"github.com/jpeach/modden/pkg/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	if err := cmd.NewRootCommand().Execute(); err != nil {
		// TODO(jpeach): fish the exit code out of the error type.
		fmt.Fprintf(os.Stderr, "%s: %s\n", version.Progname, err)
		os.Exit(cmd.EX_FAIL)
	}
}
