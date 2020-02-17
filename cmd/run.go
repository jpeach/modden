package cmd

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/spf13/cobra"
)

func executeDocument(kube *driver.KubeClient, testDoc *doc.Document) error {
	// TODO(jpeach): move document execution to a new package. Break it down.

	env := driver.NewEnvironment()
	objDriver := driver.NewObjectDriver(kube)

	for i, p := range testDoc.Parts {
		// TODO(jpeach): this is a step, record actions, errors, results.

		switch p.Decode() {
		case doc.FragmentTypeObject:
			log.Printf("applying YAML fragment %d", i)
			obj, err := env.HydrateObject(p.Bytes)
			if err != nil {
				// TODO(jpeach): attach error to
				// step. This is a fatal error, so we
				// can't continue test execution.
				return fmt.Errorf("failed to hydrate object: %s", err)
			}

			// Implicitly create the object namespace to reduce test document boilerplate.
			if nsName := obj.GetNamespace(); nsName != "" {
				exists, err := kube.NamespaceExists(nsName)
				if err != nil {
					return fmt.Errorf("failed check for namespace '%s': %s",
						nsName, err)
				}

				if !exists {
					nsObject := driver.NewNamespace(nsName)

					// TODO(jpeach): hydrate this object as if it was from YAML.

					// Apply the implicit namespace,
					// failing the test step if it errors.
					// Since we are creating the namespace
					// implicitly, we know to expect that
					// the creating should succeed.
					if err := objDriver.Apply(nsObject); err != nil {
						return fmt.Errorf("failed to create implicit namespace %q: %s",
							nsName, err)
					}
				}
			}

			// TODO(jpeach): We don't know whether this
			// object should fail or not. There are test
			// approaches where we expect that it should
			// fail (e.g API server validation).
			if err := objDriver.Apply(obj); err != nil {
				return fmt.Errorf("failed to apply object: %s", err)
			}

		case doc.FragmentTypeRego:
			log.Printf("executing Rego fragment %d", i)
		default:
			log.Printf("ignoring unknown fragment %d", i)
		}
	}

	// TODO(jpeach): return a structured test result object.
	return nil
}

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

				log.Printf("read document with %d parts from %s",
					len(testDoc.Parts), a)

				err = executeDocument(kube, testDoc)
				if err != nil {
					return fmt.Errorf("document execution failed: %s", err)
				}
			}

			return nil
		},
	}
}
