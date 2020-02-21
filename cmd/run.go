package cmd

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/spf13/cobra"
)

type runner struct {
	Kube *driver.KubeClient
	Env  driver.Environment
	Obj  driver.ObjectDriver
	Rego driver.CheckDriver
}

func executeObjectFragment(r *runner, f *doc.Fragment) error {
	obj, err := r.Env.HydrateObject(f.Bytes)
	if err != nil {
		// TODO(jpeach): attach error to
		// step. This is a fatal error, so we
		// can't continue test execution.
		return fmt.Errorf("failed to hydrate object: %s", err)
	}

	// Implicitly create the object namespace to reduce test document boilerplate.
	if nsName := obj.GetNamespace(); nsName != "" {
		exists, err := r.Kube.NamespaceExists(nsName)
		if err != nil {
			return fmt.Errorf("failed check for namespace '%s': %s",
				nsName, err)
		}

		if !exists {
			nsObject := driver.NewNamespace(nsName)

			// TODO(jpeach): hydrate this object as if it was from YAML.

			// Eval the implicit namespace,
			// failing the test step if it errors.
			// Since we are creating the namespace
			// implicitly, we know to expect that
			// the creating should succeed.
			if err := r.Obj.Apply(nsObject); err != nil {
				return fmt.Errorf("failed to create implicit namespace %q: %s",
					nsName, err)
			}
		}
	}

	// TODO(jpeach): We don't know whether this
	// object should fail or not. There are test
	// approaches where we expect that it should
	// fail (e.g API server validation).
	if err := r.Obj.Apply(obj); err != nil {
		// TODO(jpeach): store the apply result in the object driver
		log.Printf("failed to apply object: %s", err)
	}

	// TODO(jpeach): If the object has a check directly attached on the $check
	// pseudo-element, run it how. Otherwise, run the default one from assets.

	return nil
}

func executeRegoFragment(r *runner, f *doc.Fragment) error {
	resultSet, err := r.Rego.Eval(f.Rego())
	if err != nil {
		return err
	}

	for _, r := range resultSet {
		// TODO(jpeach): convert to test result and propagate.
		log.Printf("%s: %s", r.Severity, r.Message)
	}

	return err
}

func executeDocument(kube *driver.KubeClient, testDoc *doc.Document) error {
	r := runner{
		Kube: kube,
		Env:  driver.NewEnvironment(),
		Obj:  driver.NewObjectDriver(kube),
		Rego: driver.NewRegoDriver(),
	}

	// TODO(jpeach): move document execution to a new package. Break it down.

	for i, p := range testDoc.Parts {
		// TODO(jpeach): this is a step, record actions, errors, results.

		// TODO(jpeach): update runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:
			log.Printf("applying Kubernetes object fragment %d", i)

			if err := executeObjectFragment(&r, &p); err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				return err
			}

		case doc.FragmentTypeRego:
			log.Printf("executing Rego fragment %d", i)

			if err := executeRegoFragment(&r, &p); err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				return err
			}

		case doc.FragmentTypeUnknown:
			log.Printf("ignoring unknown fragment %d", i)

		case doc.FragmentTypeInvalid:
			// XXX(jpeach): We can't get here because
			// fragments never store an invalid type. Any
			// invalid fragments should already have been
			// fatally handled.
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

				err = executeDocument(kube, testDoc)
				if err != nil {
					return fmt.Errorf("document execution failed: %s", err)
				}
			}

			return nil
		},
	}
}
