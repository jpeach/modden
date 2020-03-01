package test

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type TraceFlag int

const (
	TraceNone TraceFlag = 0
	TraceRego TraceFlag = 1 << iota
)

// Runner executes a test document with the help of a collection of drivers.
type Runner struct {
	Kube *driver.KubeClient
	Env  driver.Environment
	Obj  driver.ObjectDriver
	Rego driver.CheckDriver

	Trace TraceFlag
}

// Run executes a test document.
func (r *Runner) Run(testDoc *doc.Document) error {
	if (r.Trace & TraceRego) != 0 {
		r.Rego.Trace(driver.NewCheckTracer(os.Stdout))
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
			if err := storeItem(r, "/resources/applied/last",
				result.Latest.UnstructuredContent()); err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				return err
			}

			// TODO(jpeach): create an array at `/resources/applied/log` and append this.

			// Also push the result into the resources hierarchy.
			if err := storeResource(r, result.Latest); err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
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

func printResults(resultSet []driver.CheckResult) {
	colors := map[driver.Severity]func(string, ...interface{}){
		driver.SeverityNone:  nil,
		driver.SeverityWarn:  color.Yellow,
		driver.SeverityError: color.Red,
		driver.SeverityFatal: color.HiMagenta,
	}

	for _, r := range resultSet {
		// TODO(jpeach): convert to test result and propagate.
		colors[r.Severity]("%s: %s", r.Severity, r.Message)
	}
}

func runCheckWithInput(r *Runner, f *doc.Fragment, in interface{}) error {
	resultSet, err := r.Rego.Eval(f.Rego(), rego.Input(in))
	if err != nil {
		return err
	}

	printResults(resultSet)
	return err
}

func runCheck(r *Runner, f *doc.Fragment) error {
	resultSet, err := r.Rego.Eval(f.Rego())
	if err != nil {
		return err
	}

	printResults(resultSet)
	return err
}

// storeItem stores an arbitrary item at the given path in the Rego
// data document. If we get a NotFound error when we store the resource,
// that means that an intermediate path element doesn't exist. In that
// case, we create the path and retry.
func storeItem(r *Runner, where string, what interface{}) error {
	err := r.Rego.StoreItem(where, what)
	if storage.IsNotFound(err) {
		err = r.Rego.StorePath(where)
		if err != nil {
			return err
		}

		err = r.Rego.StoreItem(where, what)
	}

	return err
}

// storeResource stores a Kubernetes object in the resources hierarchy
// of the Rego data document. The layout of this hierarchy is:
//
// Resources in the default namespace are stored as:
//	/resources/$resource/$name
//
// Namespaced resources are stored as:
//	/resources/$namespace/$resource/$name
func storeResource(r *Runner, u *unstructured.Unstructured) error {
	gvr, err := r.Kube.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	var resourcePath string

	if u.GetNamespace() == "default" {
		resourcePath = path.Join("/", "resources", gvr.Resource, u.GetName())
	} else {
		resourcePath = path.Join("/", u.GetNamespace(), "resources", gvr.Resource, u.GetName())
	}

	// NOTE(jpeach): we have to marshall the inner object into
	// the store because we don't want the resource enclosed in
	// a dictionary with the key "Object".
	return storeItem(r, resourcePath, u.UnstructuredContent())
}
