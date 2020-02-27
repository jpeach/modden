package cmd

import (
	"github.com/spf13/cobra"
)

const (
	// EX_FAIL is an exit code indicating an unspecified error.
	EX_FAIL = 1
	// EX_USAGE is an exit code indicating invalid invocation syntax.
	EX_USAGE = 65
	// EX_NOINPUT is an exit code indicating missing input data.
	EX_NOINPUT = 66
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
		Use:   "modden",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}

	root.AddCommand(NewRunCommand())
	root.AddCommand(NewGetCommand())

	return CommandWithDefaults(root)
}
