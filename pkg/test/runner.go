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
	// Initialize the Rego store.
	if err := r.Rego.StoreItem("/", skel()); err != nil {
		return err
	}

	for i, p := range testDoc.Parts {
		// TODO(jpeach): this is a step, record actions, errors, results.

		// TODO(jpeach): update Runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:
			log.Printf("applying Kubernetes object fragment %d", i)

			result, err := executeObjectFragment(r, &p)
			if err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				log.Printf("unable to apply object update: %s", err)
				return err
			}

			err = r.Rego.StoreItem("/resources/applied/last", result)
			if err != nil {
				return err
			}

			// TODO(jpeach): If the object has a check directly attached on the $check
			// pseudo-element, run it how. Otherwise, run the default one from assets.

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

func executeObjectFragment(r *Runner, f *doc.Fragment) (*driver.OperationResult, error) {
	obj, err := r.Env.HydrateObject(f.Bytes)
	if err != nil {
		// TODO(jpeach): attach error to
		// step. This is a fatal error, so we
		// can't continue test execution.
		return nil, fmt.Errorf("failed to hydrate object: %s", err)
	}

	// Implicitly create the object namespace to reduce test document boilerplate.
	if nsName := obj.GetNamespace(); nsName != "" {
		exists, err := r.Kube.NamespaceExists(nsName)
		if err != nil {
			return nil, fmt.Errorf(
				"failed check for namespace '%s': %s", nsName, err)
		}

		if !exists {
			nsObject := driver.NewNamespace(nsName)

			// TODO(jpeach): hydrate this object as if it was from YAML.

			// Eval the implicit namespace,
			// failing the test step if it errors.
			// Since we are creating the namespace
			// implicitly, we know to expect that
			// the creating should succeed.
			result, err := r.Obj.Apply(nsObject)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to create implicit namespace %q: %w", nsName, err)
			}

			if !result.Succeeded() {
				return result, nil
			}
		}
	}

	return r.Obj.Apply(obj)
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

// skel returns a skeleton data structure used to initialize the
// Rego store. We need to sketch out some initial nodes to have
//places to store subsequent data items.
func skel() interface{} {
	return map[string]interface{}{
		"resources": map[string]interface{}{
			"applied": map[string]interface{}{},
		},
	}
}
