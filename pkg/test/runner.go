package test

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jpeach/modden/pkg/builtin"
	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/filter"
	"github.com/jpeach/modden/pkg/must"
	"github.com/jpeach/modden/pkg/utils"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		tc.regoDriver.Trace(driver.NewRegoTracer(os.Stdout))
	})
}

// RegoParamOpt writes a parameter into the Rego store, rooted at
// the path `/test/params`. If the parameter name contains interior
// dots (e.g. "foo.bar.baz"), those are converted into path separators.
func RegoParamOpt(key string, val string) RunOpt {
	return RunOpt(func(tc *testContext) {
		parts := []string{"/", "test", "params"}
		parts = append(parts, strings.Split(key, ".")...)
		p := path.Join(parts...)
		must.Must(tc.regoDriver.StorePath(p))
		must.Must(tc.regoDriver.StoreItem(p, val))
	})
}

// RegoModuleOpt makes the given module available to the Rego evaluation.
func RegoModuleOpt(m *ast.Module) RunOpt {
	return RunOpt(func(tc *testContext) {
		// We assume that the caller has already validated
		// the file and that it can be read and parsed.
		tc.policyModules = append(tc.policyModules, m)
	})
}

// PreserveObjectsOpt disables automatic object deletion.
func PreserveObjectsOpt() RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.preserve = true
	})
}

// WatchResourceOpt disables automatic object deletion.
func WatchResourceOpt(gvr schema.GroupVersionResource) RunOpt {
	return RunOpt(func(tc *testContext) {
		tc.watchedResources = append(tc.watchedResources, gvr)
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
	regoDriver   driver.RegoDriver
	envDriver    driver.Environment
	recorder     Recorder

	dryRun           bool
	preserve         bool
	checkTimeout     time.Duration
	watchedResources []schema.GroupVersionResource
	policyModules    []*ast.Module
}

// Run executes a test document.
//
// nolint(gocognit)
func Run(testDoc *doc.Document, opts ...RunOpt) error {
	var compiler *ast.Compiler
	var err error

	tc := testContext{
		envDriver:    driver.NewEnvironment(),
		regoDriver:   driver.NewRegoDriver(),
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
				must.Must(storeResource(tc.kubeDriver, tc.regoDriver, u))
			}
		}, UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if u, ok := newObj.(*unstructured.Unstructured); ok {
				must.Must(storeResource(tc.kubeDriver, tc.regoDriver, u))
			}
		}, DeleteFunc: func(o interface{}) {
			if u, ok := o.(*unstructured.Unstructured); ok {
				must.Must(removeResource(tc.kubeDriver, tc.regoDriver, u))
			}
		},
	})

	defer cancelWatch()

	for _, gvr := range tc.watchedResources {
		tc.objectDriver.InformOn(gvr)
	}

	tc.regoDriver.StoreItem("/test/params/run-id", tc.envDriver.UniqueID())

	step(tc.recorder, "compiling test document", func() {
		compiler, err = compileDocument(testDoc, tc.policyModules)
		if err != nil {
			tc.recorder.Errorf(SeverityFatal, "%s", err.Error())
		}
	})

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
			step(tc.recorder, "matching anonymous Kubernetes object", func() {
				if obj.Object.GetName() != "" {
					return
				}

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

			step(tc.recorder, "updating Kubernetes object", func() {
				tc.recorder.Messagef("performing %s operation on %s '%s/%s'",
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
					if err := storeItem(tc.regoDriver, "/resources/applied/last",
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

				check := obj.Check
				opts := []driver.RegoOpt{
					rego.Compiler(compiler),
					rego.Input(result),
				}

				// If we have a check from the object,
				// it has not been added to the compiler,
				// so we need to pass it in as a parsed
				// module. Otherwise, we can use the
				// default check which the compiler had
				// already compiled.
				if check != nil {
					opts = append(opts, rego.ParsedModule(check))
				} else {
					check = DefaultObjectCheckForOperation(obj.Operation)
				}

				checkResults, err := runCheck(
					tc.regoDriver, check, tc.checkTimeout, opts...)
				if err != nil {
					tc.recorder.Errorf(SeverityFatal, "%s", err)
				}

				recordResults(tc.recorder, checkResults)
			})

		case doc.FragmentTypeModule:
			step(tc.recorder, "running Rego check", func() {
				checkResults, err := runCheck(
					tc.regoDriver, p.Rego(), tc.checkTimeout, rego.Compiler(compiler))
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

// compileDocument compiles all the Rego policies in the test document.
func compileDocument(d *doc.Document, modules []*ast.Module) (*ast.Compiler, error) {
	compiler := ast.NewCompiler()
	modmap := map[string]*ast.Module{}

	// Compile all the built-in Rego files. We require that
	// each file has a unique module name.
	for _, a := range builtin.AssetNames() {
		if !strings.HasSuffix(a, ".rego") {
			continue
		}

		str := string(must.Bytes(builtin.Asset(a)))
		m := must.Module(ast.ParseModule(a, str))

		if _, ok := modmap[a]; ok {
			return nil, fmt.Errorf("duplicate builtin Rego module asset %q", a)
		}

		modmap[a] = m
	}

	// Add all the modules that the user specified on the commandline.
	for _, m := range modules {
		name := m.Package.Loc().File
		if _, ok := modmap[name]; ok {
			return nil, fmt.Errorf("duplicate Rego module file %q", name)
		}

		modmap[name] = m
	}

	// Finally, add all the check modules in the document.
	for _, p := range d.Parts {
		switch p.Type {
		case doc.FragmentTypeModule:
			name := fmt.Sprintf("doc/%s", p.Rego().Package.Path.String())
			if _, ok := modmap[name]; ok {
				return nil, fmt.Errorf("duplicate Rego fragment file %q", name)
			}

			modmap[name] = p.Rego()
		}
	}

	if compiler.Compile(modmap); compiler.Failed() {
		return nil, compiler.Errors
	}

	return compiler, nil
}

func runCheck(
	c driver.RegoDriver,
	m *ast.Module,
	timeout time.Duration,
	opts ...driver.RegoOpt) ([]driver.CheckResult, error) {
	var err error
	var results []driver.CheckResult

	startTime := time.Now()

	for time.Since(startTime) < timeout {
		results, err = c.Eval(m, opts...)
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
func storeItem(c driver.RegoDriver, where string, what interface{}) error {
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
func storeResource(k *driver.KubeClient, c driver.RegoDriver, u *unstructured.Unstructured) error {
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
func removeResource(k *driver.KubeClient, c driver.RegoDriver, u *unstructured.Unstructured) error {
	gvr, err := k.ResourceForKind(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return err
	}

	return c.RemovePath(pathForResource(gvr.Resource, u))
}
