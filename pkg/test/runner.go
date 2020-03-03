package test

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/must"

	"github.com/fatih/color"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

// TraceFlag specifies what tracing output to enable
type TraceFlag int

const (
	// TraceNone is the default trace type.
	TraceNone TraceFlag = 0
	// TraceRego enables tracing Rego execution.
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
//
// nolint(gocognit)
func (r *Runner) Run(testDoc *doc.Document) error {
	if (r.Trace & TraceRego) != 0 {
		r.Rego.Trace(driver.NewCheckTracer(os.Stdout))
	}

	// Start receiving Kubernetes objects and adding them to the
	// store. We currently don't need any locking around this since
	// the Rego store is transactional and this path doesn't touch
	// any other shared data.
	cancelWatch := r.Obj.Watch(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			if u, ok := o.(*unstructured.Unstructured); ok {
				must.Must(storeResource(r, u))
			}
		}, UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if u, ok := newObj.(*unstructured.Unstructured); ok {
				must.Must(storeResource(r, u))
			}
		}, DeleteFunc: func(o interface{}) {
			if u, ok := o.(*unstructured.Unstructured); ok {
				must.Must(removeResource(r, u))
			}
		},
	})

	defer cancelWatch()

	for i, p := range testDoc.Parts {
		// TODO(jpeach): this is a step, record actions, errors, results.

		// TODO(jpeach): if there are any pending fatal
		// actions, stop the test. Depending on config
		// we may have to clean up.

		// TODO(jpeach): update Runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:

			obj, err := r.Env.HydrateObject(p.Bytes)
			if err != nil {
				// TODO(jpeach): attach error to
				// step. This is a fatal error, so we
				// can't continue test execution.
				return fmt.Errorf("failed to hydrate object: %s", err)
			}

			var result *driver.OperationResult

			switch obj.Operation {
			case driver.ObjectOperationUpdate:
				log.Printf("applying Kubernetes object fragment %d", i)
				result, err = applyObject(r, obj.Object)
			case driver.ObjectOperationDelete:
				log.Printf("deleting Kubernetes object fragment %d", i)
				result, err = r.Obj.Delete(obj.Object)
			}

			if err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				log.Printf("unable to %s object: %s", obj.Operation, err)
				return err
			}

			if result.Latest != nil {
				// First, push the result into the store.
				if err := storeItem(r, "/resources/applied/last",
					result.Latest.UnstructuredContent()); err != nil {
					// TODO(jpeach): this should be treated as a fatal test error.
					return err
				}

				// TODO(jpeach): create an array at `/resources/applied/log` and append this.
			}

			// Now, if this object has a specific check, run it. Otherwise, we can
			if obj.Check != nil {
				err = runCheck(r, obj.Check, result)
			} else {
				err = runCheck(r, DefaultObjectCheckForOperation(obj.Operation), result)
			}

			if err != nil {
				// TODO(jpeach): this should be treated as a fatal test error.
				log.Printf("%s", err)
			}

		case doc.FragmentTypeRego:
			log.Printf("executing Rego fragment %d", i)

			if err := runCheck(r, &p, nil); err != nil {
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

	r.Obj.Done()

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

func runCheck(r *Runner, f *doc.Fragment, input interface{}) error {
	var err error
	var results []driver.CheckResult

	// TODO(jpeach): this retry loop is clearly super hacky:
	//
	// 1. It is possible that the checks erroneously succeed on the
	//    first pass (and would have subsequently failed).
	// 2. Hard-coding the retries is gauche, but we could extract
	//    that policy from the Rego document.
	// 3. Every failure is guaranteed to hit the timeout, so failing
	//    tests will suck.

	var ops []func(*rego.Rego)

	if input != nil {
		ops = append(ops, rego.Input(input))
	}

	for tries := 10; tries > 0; tries-- {
		results, err = r.Rego.Eval(f.Rego(), ops...)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		time.Sleep(time.Millisecond * 500)
	}

	printResults(results)
	return err
}

// Resources in the default namespace are stored as:
//	/resources/$resource/$name
//
// Namespaced resources are stored as:
//     /resources/$namespace/$resource/$name
func pathForResource(resource string, u *unstructured.Unstructured) string {
	if u.GetNamespace() == "default" {
		return path.Join("/", "resources", resource, u.GetName())
	}

	return path.Join("/", "resources", u.GetNamespace(), resource, u.GetName())
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
// of the Rego data document.
func storeResource(r *Runner, u *unstructured.Unstructured) error {
	gvr, err := r.Kube.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	// NOTE(jpeach): we have to marshall the inner object into
	// the store because we don't want the resource enclosed in
	// a dictionary with the key "Object".
	return storeItem(r, pathForResource(gvr.Resource, u), u.UnstructuredContent())
}

// removeResource removes a Kubernetes object from the resources hierarchy
// of the Rego data document.
func removeResource(r *Runner, u *unstructured.Unstructured) error {
	gvr, err := r.Kube.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	return r.Rego.RemovePath(pathForResource(gvr.Resource, u))
}
