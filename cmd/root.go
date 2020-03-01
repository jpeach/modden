package cmd

import (
	"fmt"

	"github.com/jpeach/modden/pkg/version"

	"github.com/spf13/cobra"
)

const (
	// EX_FAIL is an exit code indicating an unspecified error.
	EX_FAIL = 1 //nolint(golint)
	// EX_USAGE is an exit code indicating invalid invocation syntax.
	EX_USAGE = 65 //nolint(golint)
	// EX_NOINPUT is an exit code indicating missing input data.
	EX_NOINPUT = 66 //nolint(golint)
)

// CommandWithDefaults overwrites default values in the given command.
func CommandWithDefaults(c *cobra.Command) *cobra.Command {
	c.SilenceUsage = true
	c.SilenceErrors = true

	return c
}

// NewRootCommand represents the base command when called without any subcommands
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   version.Progname,
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Version: fmt.Sprintf("v%d.%s/%s", version.Major, version.Minor, version.Sha),
	}

	root.AddCommand(NewRunCommand())
	root.AddCommand(NewGetCommand())

	return CommandWithDefaults(root)
}
