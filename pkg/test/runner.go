package test

import (
	"fmt"
	"log"
	"os"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/topdown"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

		// TODO(jpeach): if there are any pending fatal
		// actions, stop the test. Depending on config
		// we may have to clean up.

		// TODO(jpeach): update Runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:
			log.Printf("applying Kubernetes object fragment %d", i)

			obj, err := r.Env.HydrateObject(p.Bytes)
			if err != nil {
				// TODO(jpeach): attach error to
				// step. This is a fatal error, so we
				// can't continue test execution.
				return fmt.Errorf("failed to hydrate object: %s", err)
			}

			var result *driver.OperationResult

			if obj.Delete {

			} else {
				result, err = applyObject(r, obj.Object)
				if err != nil {
					// TODO(jpeach): this should be treated as a fatal test error.
					log.Printf("unable to apply object update: %s", err)
					return err
				}
			}

			// First, push the result into the store.
			err = r.Rego.StoreItem("/resources/applied/last", result)
			if err != nil {
				return err
			}

			// Now, if this object has a specific check, run it. Otherwise, we can
			if obj.Check != nil {
				err = runCheckWithInput(r, obj.Check, result)
			} else {
				err = runCheckWithInput(r, DefaultObjectUpdateCheck(), result)
			}

			if err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				log.Printf("%s", err)
			}

		case doc.FragmentTypeRego:
			log.Printf("executing Rego fragment %d", i)

			if err := runCheck(r, &p); err != nil {
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

func applyObject(r *Runner, u *unstructured.Unstructured) (*driver.OperationResult, error) {
	// Implicitly create the object namespace to reduce test document boilerplate.
	if nsName := u.GetNamespace(); nsName != "" {
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

	return r.Obj.Apply(u)
}

func runCheckWithInput(r *Runner, f *doc.Fragment, in interface{}) error {
	traceBuf := topdown.NewBufferTracer()

	resultSet, err := r.Rego.Eval(f.Rego(), rego.Input(in), rego.Tracer(traceBuf))
	if err != nil {
		return err
	}

	topdown.PrettyTrace(os.Stderr, *traceBuf)

	for _, r := range resultSet {
		// TODO(jpeach): convert to test result and propagate.
		log.Printf("%s: %s", r.Severity, r.Message)
	}

	return err
}

func runCheck(r *Runner, f *doc.Fragment) error {
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
