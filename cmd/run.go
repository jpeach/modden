package cmd

import (
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			testDoc, err := doc.ReadFile(a)
			if err != nil {
				return err
			}

			log.Printf("read document with %d parts from %s",
				len(testDoc.Parts), a)

			for i, p := range testDoc.Parts {
				switch p.Decode() {
				case doc.FragmentTypeObject:
					log.Printf("applying YAML fragment %d", i)
				case doc.FragmentTypeRego:
					log.Printf("executing Rego fragment %d", i)
				default:
					log.Printf("ignoring unknown fragment %d", i)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
