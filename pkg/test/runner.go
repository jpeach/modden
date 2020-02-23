package test

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
)

// Runner executes a test document with the help of a collection of drivers.
type Runner struct {
	Kube *driver.KubeClient
	Env  driver.Environment
	Obj  driver.ObjectDriver
	Rego driver.CheckDriver
}

// Run executes a test document.
func (r *Runner) Run(testDoc *doc.Document) error {
	for i, p := range testDoc.Parts {
		// TODO(jpeach): this is a step, record actions, errors, results.

		// TODO(jpeach): update Runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:
			log.Printf("applying Kubernetes object fragment %d", i)

			if err := executeObjectFragment(r, &p); err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				return err
			}

		case doc.FragmentTypeRego:
			log.Printf("executing Rego fragment %d", i)

			if err := executeRegoFragment(r, &p); err != nil {
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

func executeObjectFragment(r *Runner, f *doc.Fragment) error {
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

func executeRegoFragment(r *Runner, f *doc.Fragment) error {
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
