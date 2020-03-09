package main

import (
	"errors"
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
		if msg := err.Error(); msg != "" {
			fmt.Fprintf(os.Stderr, "%s: %s\n", version.Progname, msg)
		}

		var exit *cmd.ExitError
		if errors.As(err, &exit) {
			os.Exit(int(exit.Code))
		}

		os.Exit(int(cmd.EX_FAIL))
	}
}
