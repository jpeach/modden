package cmd

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/test"
	"github.com/spf13/cobra"
)

// NewRunCommand returns a command ro run a test case.
func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			kube, err := driver.NewKubeClient()
			if err != nil {
				return fmt.Errorf("failed to initialize Kubernetes context: %s", err)
			}

			// TODO(jpeach): set user agent from program version.
			kube.SetUserAgent("modden/TODO")

			for _, a := range args {
				testDoc, err := doc.ReadFile(a)
				if err != nil {
					return err
				}

				log.Printf("reading document with %d parts from %s",
					len(testDoc.Parts), a)

				// Before executing anything, verify that we can decode all the
				// fragments and raise any syntax errors.
				for i := range testDoc.Parts {
					p := &testDoc.Parts[i]

					fragType, err := p.Decode()
					if err == nil {
						log.Printf("decoded fragment %d as %s", i, fragType)
						continue
					}

					log.Printf("error on %s fragment %d: %s", fragType, i, err)

					// If we have a compile error, puke it.
					if err := doc.AsRegoCompilationErr(err); err != nil {
						// TODO(jpeach): rewrite the location
						// of the Rego error. The line number
						// will be relative to the start of the
						// fragment, and we should make it
						// relative to the start of the document.
						return err
					}

					return err
				}

				r := test.Runner{
					Kube: kube,
					Env:  driver.NewEnvironment(),
					Obj:  driver.NewObjectDriver(kube),
					Rego: driver.NewRegoDriver(),
				}

				if err := r.Run(testDoc); err != nil {
					return fmt.Errorf("test run failed: %s", err)
				}
			}

			return nil
		},
	}
}
