package test

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/filter"
	"github.com/jpeach/modden/pkg/must"
	"github.com/jpeach/modden/pkg/utils"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

// RunOpt sets options for the test run.
type RunOpt func(*testContext)

// KubeClientOpt sets the Kubernetes client.
func KubeClientOpt(kube *driver.KubeClient) RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.kubeDriver = kube
		tc.objectDriver = driver.NewObjectDriver(kube)
	})
}

// RecorderOpt sets the test recorder.
func RecorderOpt(r Recorder) RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.recorder = r
	})
}

// TraceRegoOpt enables Rego tracing.
func TraceRegoOpt() RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.checkDriver.Trace(driver.NewCheckTracer(os.Stdout))
	})
}

// RegoParamOpt writes a parameter into the Rego store, rooted at
// the path `/test/params`.
func RegoParamOpt(key string, val string) RunOpt {
	return RunOpt(func(tc *testContext) {
		p := path.Join("/", "test", "params", key)
		must.Must(tc.checkDriver.StorePath(p))
		must.Must(tc.checkDriver.StoreItem(p, val))
	})
}

// PreserveObjectsOpt disables automatic object deletion.
func PreserveObjectsOpt() RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.preserve = true
	})
}

// DryRunOpt enables Kuberentes dry-run mode (TODO).
func DryRunOpt() RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.dryRun = true
	})
}

// CheckTimeoutOpt sets the check timeout.
func CheckTimeoutOpt(timeout time.Duration) RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.checkTimeout = timeout
	})
}

func step(tc Recorder, stepDesc string, f func()) {
	stepCloser := tc.NewStep(stepDesc)
	defer stepCloser.Close()

	if !tc.ShouldContinue() {
		tc.Errorf(SeverityError, "skipping")
		return
	}

	f()
}

type testContext struct {
	kubeDriver   *driver.KubeClient
	objectDriver driver.ObjectDriver
	checkDriver  driver.CheckDriver
	envDriver    driver.Environment
	recorder     Recorder

	dryRun       bool
	preserve     bool
	checkTimeout time.Duration
}

// Run executes a test document.
//
// nolint(gocognit)
func Run(testDoc *doc.Document, opts ...RunOpt) error {
	tc := testContext{
		envDriver:    driver.NewEnvironment(),
		checkDriver:  driver.NewRegoDriver(),
		checkTimeout: time.Second * 10,
	}

	for _, o := range opts {
		o(&tc)
	}

	if tc.objectDriver == nil {
		return fmt.Errorf("missing Kubernetes object driver")
	}

	defer tc.objectDriver.Done()

	// Start receiving Kubernetes objects and adding them to the
	// store. We currently don't need any locking around this since
	// the Rego store is transactional and this path doesn't touch
	// any other shared data.
	cancelWatch := tc.objectDriver.Watch(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			if u, ok := o.(*unstructured.Unstructured); ok {
				must.Must(storeResource(tc.kubeDriver, tc.checkDriver, u))
			}
		}, UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if u, ok := newObj.(*unstructured.Unstructured); ok {
				must.Must(storeResource(tc.kubeDriver, tc.checkDriver, u))
			}
		}, DeleteFunc: func(o interface{}) {
			if u, ok := o.(*unstructured.Unstructured); ok {
				must.Must(removeResource(tc.kubeDriver, tc.checkDriver, u))
			}
		},
	})

	defer cancelWatch()

	for _, p := range testDoc.Parts {
		if !tc.recorder.ShouldContinue() {
			break
		}

		// TODO(jpeach): this is a step, record actions, errors, results.

		// TODO(jpeach): if there are any pending fatal
		// actions, stop the test. Depending on config
		// we may have to clean up.

		// TODO(jpeach): update Runner.Rego.Store() with the current state
		// from the object driver.

		switch p.Type {
		case doc.FragmentTypeObject:
			var err error
			var obj *driver.Object
			var result *driver.OperationResult

			step(tc.recorder, "hydrating Kubernetes object", func() {
				obj, err = tc.envDriver.HydrateObject(p.Bytes)
				if err != nil {
					tc.recorder.Errorf(SeverityFatal, "failed to hydrate object: %s", err)
					return
				}

				if obj.Object.GetName() == "" {
					tc.recorder.Messagef("hydrated anonymous %s:%s object",
						obj.Object.GetAPIVersion(),
						obj.Object.GetKind())
				} else {
					tc.recorder.Messagef("hydrated %s:%s object '%s/%s'",
						obj.Object.GetAPIVersion(),
						obj.Object.GetKind(),
						utils.NamespaceOrDefault(obj.Object),
						obj.Object.GetName())
				}
			})

			// If we don't have an object name, try to
			// select it using the labels. Note that we
			// may have to wait here, because the objects
			// we want to select may not have been created
			// yet.
			if obj.Object.GetName() == "" {
				step(tc.recorder, "matching anonymous Kubernetes object", func() {
					s := utils.NewSelectorFromObject(obj.Object)

					tc.recorder.Messagef("matching anonymous %s:%s object",
						obj.Object.GetAPIVersion(), obj.Object.GetKind())

					tc.recorder.Messagef("selector %q", s.String())

					// TODO(jpeach): select on namespace if present?

					candidates, err := tc.kubeDriver.SelectObjects(
						obj.Object.GroupVersionKind(),
						utils.NewSelectorFromObject(obj.Object))
					if err != nil {
						tc.recorder.Errorf(SeverityFatal, "listing %s:%s objects: %s",
							obj.Object.GetAPIVersion(), obj.Object.GetKind(), err)
						return
					}

					var match *unstructured.Unstructured
					for _, u := range candidates {
						if filter.ObjectRunID(u) == tc.envDriver.UniqueID() {
							match = u
							break
						}
					}

					if match == nil {
						tc.recorder.Errorf(SeverityFatal,
							"failed to match object with run ID %s",
							tc.envDriver.UniqueID())
						return
					}

					obj.Object = match
					tc.recorder.Messagef("matched %s:%s object '%s/%s'",
						obj.Object.GetAPIVersion(),
						obj.Object.GetKind(),
						utils.NamespaceOrDefault(obj.Object),
						obj.Object.GetName())

				})
			}

			step(tc.recorder, "updating Kubernetes object", func() {
				tc.recorder.Messagef("performing %s on %s '%s/%s'",
					obj.Operation,
					obj.Object.GetKind(),
					utils.NamespaceOrDefault(obj.Object),
					obj.Object.GetName())

				switch obj.Operation {
				case driver.ObjectOperationUpdate:
					result, err = applyObject(tc.kubeDriver, tc.objectDriver, obj.Object)
				case driver.ObjectOperationDelete:
					result, err = tc.objectDriver.Delete(obj.Object)
				}

				if err != nil {
					// TODO(jpeach): this should be treated as a fatal test error.
					tc.recorder.Errorf(SeverityFatal, "unable to %s object: %s", obj.Operation, err)
					return
				}

				if result.Latest != nil {
					// First, push the result into the store.
					if err := storeItem(tc.checkDriver, "/resources/applied/last",
						result.Latest.UnstructuredContent()); err != nil {
						tc.recorder.Errorf(SeverityFatal, "failed to store result: %s", err)
						return
					}

					// TODO(jpeach): create an array at `/resources/applied/log` and append this.
				}
			})

			step(tc.recorder, "running object update check", func() {
				tc.recorder.Messagef("checking %s of %s '%s/%s'",
					obj.Operation,
					obj.Object.GetKind(),
					utils.NamespaceOrDefault(obj.Object),
					obj.Object.GetName())

				// Now, if this object has a specific check, run it. Otherwise, we can
				if obj.Check == nil {
					obj.Check = DefaultObjectCheckForOperation(obj.Operation)
				}

				checkResults, err := runCheck(tc.checkDriver, obj.Check, tc.checkTimeout, result)
				if err != nil {
					tc.recorder.Errorf(SeverityFatal, "%s", err)
				}

				recordResults(tc.recorder, checkResults)
			})

		case doc.FragmentTypeRego:

			step(tc.recorder, "running Rego check", func() {
				checkResults, err := runCheck(tc.checkDriver, &p, tc.checkTimeout, nil)
				if err != nil {
					tc.recorder.Errorf(SeverityFatal, "%s", err)
				}

				recordResults(tc.recorder, checkResults)
			})

		case doc.FragmentTypeUnknown:
			// Ignore unknown fragments.

		case doc.FragmentTypeInvalid:
			// XXX(jpeach): We can't get here because
			// fragments never store an invalid type. Any
			// invalid fragments should already have been
			// fatally handled.
		}
	}

	if !tc.preserve {
		must.Must(tc.objectDriver.DeleteAll())
	}

	// TODO(jpeach): return a structured test result object.
	return nil
}

func applyObject(k *driver.KubeClient,
	o driver.ObjectDriver,
	u *unstructured.Unstructured) (*driver.OperationResult, error) {
	// Implicitly create the object namespace to reduce test document boilerplate.
	if nsName := u.GetNamespace(); nsName != "" {
		exists, err := k.NamespaceExists(nsName)
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
			result, err := o.Apply(nsObject)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to create implicit namespace %q: %w", nsName, err)
			}

			if !result.Succeeded() {
				return result, nil
			}
		}
	}

	return o.Apply(u)
}

func recordResults(recorder Recorder, resultSet []driver.CheckResult) {
	for _, r := range resultSet {
		recorder.Errorf(Severity(r.Severity), "%s", r.Message)
	}
}

func runCheck(
	c driver.CheckDriver,
	f *doc.Fragment,
	timeout time.Duration,
	input interface{}) ([]driver.CheckResult, error) {
	var err error
	var results []driver.CheckResult
	var ops []func(*rego.Rego)

	if input != nil {
		ops = append(ops, rego.Input(input))
	}

	startTime := time.Now()

	for time.Since(startTime) < timeout {

		results, err = c.Eval(f.Rego(), ops...)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, nil
		}

		time.Sleep(time.Millisecond * 500)
	}

	return results, err
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
func storeItem(c driver.CheckDriver, where string, what interface{}) error {
	err := c.StoreItem(where, what)
	if storage.IsNotFound(err) {
		err = c.StorePath(where)
		if err != nil {
			return err
		}

		err = c.StoreItem(where, what)
	}

	return err
}

// storeResource stores a Kubernetes object in the resources hierarchy
// of the Rego data document.
func storeResource(k *driver.KubeClient, c driver.CheckDriver, u *unstructured.Unstructured) error {
	gvr, err := k.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	// NOTE(jpeach): we have to marshall the inner object into
	// the store because we don't want the resource enclosed in
	// a dictionary with the key "Object".
	return storeItem(c, pathForResource(gvr.Resource, u), u.UnstructuredContent())
}

// removeResource removes a Kubernetes object from the resources hierarchy
// of the Rego data document.
func removeResource(k *driver.KubeClient, c driver.CheckDriver, u *unstructured.Unstructured) error {
	gvr, err := k.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	return c.RemovePath(pathForResource(gvr.Resource, u))
}
