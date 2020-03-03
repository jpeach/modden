package driver

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/open-policy-agent/opa/topdown"
)

// Severity indicated the seriousness of a test failure.
// TODO(jpeach): Severity belongs in the test runner package.
type Severity string

// SeverityNone ...
const SeverityNone Severity = "None"

// SeverityWarn ...
const SeverityWarn Severity = "Warn"

// SeverityError ...
const SeverityError Severity = "Error"

// SeverityFatal ...
const SeverityFatal Severity = "Fatal"

// CheckResult ...
type CheckResult struct {
	Severity Severity
	Message  string
}

// CheckTracer is a tracer for check execution.
type CheckTracer interface {
	topdown.Tracer
	Write()
}

type defaultTracer struct {
	*topdown.BufferTracer
	writer io.Writer
}

func (d *defaultTracer) Write() {
	topdown.PrettyTrace(d.writer, *d.BufferTracer)
}

var _ CheckTracer = &defaultTracer{}

// NewCheckTracer returns a new CheckTracer that traces to w.
func NewCheckTracer(w io.Writer) CheckTracer {
	return &defaultTracer{
		BufferTracer: topdown.NewBufferTracer(),
		writer:       w,
	}
}

// CheckDriver is a driver for running Rego policy checks.
type CheckDriver interface {
	// Eval evaluates the given module and returns and check results.
	Eval(*ast.Module, ...func(*rego.Rego)) ([]CheckResult, error)

	Trace(CheckTracer)

	// StoreItem stores the value at the given path in the Rego data document.
	StoreItem(string, interface{}) error

	// StorePath creates the given path in the Rego data document.
	StorePath(where string) error

	// RemovePath remove any object at the given path in the Rego data document.
	RemovePath(where string) error
}

// NewRegoDriver creates a new CheckDriver that evaluates checks
// written in Rego.
//
// See https://www.openpolicyagent.org/docs/latest/policy-language/
func NewRegoDriver() CheckDriver {
	return &regoDriver{
		store: inmem.New(),
	}
}

var _ CheckDriver = &regoDriver{}

type regoDriver struct {
	store  storage.Store
	tracer CheckTracer
}

func (r *regoDriver) Trace(tracer CheckTracer) {
	r.tracer = tracer
}

// StoreItem stores the value at the given Rego store path.
func (r *regoDriver) StoreItem(where string, what interface{}) error {
	ctx := context.Background()
	txn := storage.NewTransactionOrDie(ctx, r.store, storage.WriteParams)

	path := storage.MustParsePath(where)

	err := r.store.Write(ctx, txn, storage.ReplaceOp, path, what)
	if storage.IsNotFound(err) {
		err = r.store.Write(ctx, txn, storage.AddOp, path, what)
	}

	if err != nil {
		r.store.Abort(ctx, txn)
		return err
	}

	if err := r.store.Commit(ctx, txn); err != nil {
		return err
	}

	return nil
}

// StorePath creates the given path in the Rego data document.
func (r *regoDriver) StorePath(where string) error {
	ctx := context.Background()
	txn := storage.NewTransactionOrDie(ctx, r.store, storage.WriteParams)

	var currentPath storage.Path

	for _, p := range storage.MustParsePath(where) {
		currentPath = append(currentPath, p)

		_, err := r.store.Read(ctx, txn, currentPath)
		switch {
		case err == nil:
			// If the read succeeded, there was an element.
			continue
		case storage.IsNotFound(err):
			// If the path element isn't there, cover it with am empty node.
			val := map[string]interface{}{}
			if err := r.store.Write(ctx, txn, storage.AddOp, currentPath, val); err != nil {
				r.store.Abort(ctx, txn)
				return err
			}
		default:
			// Any other error, abort and propagate it.
			r.store.Abort(ctx, txn)
			return err
		}
	}

	if err := r.store.Commit(ctx, txn); err != nil {
		return err
	}

	return nil
}

// RemovePath removes the given path in the Rego data document.
func (r *regoDriver) RemovePath(where string) error {
	ctx := context.Background()
	txn := storage.NewTransactionOrDie(ctx, r.store, storage.WriteParams)

	if err := r.store.Write(ctx, txn, storage.RemoveOp, storage.MustParsePath(where), nil); err != nil {
		r.store.Abort(ctx, txn)
		return err
	}

	if err := r.store.Commit(ctx, txn); err != nil {
		return err
	}

	return nil
}

// Eval evaluates checks in the given module.
func (r *regoDriver) Eval(m *ast.Module, opts ...func(*rego.Rego)) ([]CheckResult, error) {
	c := ast.NewCompiler()

	// Find the unique set of assertion rules to query.
	ruleNames := findAssertionRules(m)
	checkResults := make([]CheckResult, 0, len(ruleNames))

	for _, name := range ruleNames {
		// The package path will be an absolute path through the
		// data document, so to convert that into the package
		// name, we trim the leading "data." component. We need
		// the literal package name of the module in the query
		// context so names resolve correctly.
		pkg := strings.TrimPrefix(m.Package.Path.String(), "data.")

		log.Printf("evaluating query %q with package %q",
			queryForRuleName(name), pkg)

		regoObj := rego.New(
			// Scope the query to the current module package.
			rego.Package(pkg),
			// Query for the result of this named rule.
			rego.Query(queryForRuleName(name)),
			rego.Compiler(c),
			rego.ParsedModule(m),
			rego.Store(r.store),

			// TODO(jpeach): if tracing is configured, add rego.Tracer().
		)

		for _, o := range opts {
			o(regoObj)
		}

		if r.tracer != nil {
			rego.Tracer(r.tracer)(regoObj)
		}

		resultSet, err := regoObj.Eval(context.Background())
		if err != nil {
			// TODO(jpeach): propagate fatal error result.
			return nil, err
		}

		if r.tracer != nil {
			r.tracer.Write()
		}

		// In each result, the Text is the expression that we
		// queried, and value is one or more bound messages.
		for _, result := range resultSet {
			for _, e := range result.Expressions {
				for _, m := range findResultMessage(e) {
					checkResults = append(checkResults,
						CheckResult{
							Severity: severityForRuleName(e.Text),
							Message:  fmt.Sprint(m),
						})
				}
			}
		}
	}

	return checkResults, nil
}

// findResultMessage examines a rego.ExpressionValue to find the result
// (message) of a rule that we queried . A Rego query has an optional
// key term that can be of any type. In most cases, the term will be
// a string, like this:
// 	`error[msg]{ ... }`
// but it could be anything. For example, a map like this:
// 	`error[{"msg": "foo", "sev": "bad"}]{ ... }`
// So here, we follow the example of conftest and accept a key term
// that is either a string or a map with a string-valued key names
// "msg". In the future, we could accept other types, but
//
// See also https://github.com/instrumenta/conftest/pull/243.
func findResultMessage(result *rego.ExpressionValue) []string {
	var messages []string

	// This might be a boolean if the rule was this:
	//	`error { ... }`
	if _, ok := result.Value.(bool); ok {
		// Rego only returns the results of boolean rules
		// if the rule was true, so the value of the bool
		// result doesn't matter. We just know there's no
		// message.
		return []string{""}
	}

	// If the value isn't some kind of slice, then we don't
	// know how to deal with it.
	if _, ok := result.Value.([]interface{}); !ok {
		// TODO(jpeach): this should be a fatal error.
		return messages
	}

	// Extract messages from the value slice. The reason there is
	// a slice is that there can be many matching cases for this
	// rule and the query evaluates them all simultaneously. Each
	// matching case might emit a message.
	for _, v := range result.Value.([]interface{}) {
		switch value := v.(type) {
		case string:
			messages = append(messages, value)
		case map[string]interface{}:
			if _, ok := value["msg"]; ok {
				if m, ok := value["msg"].(string); ok {
					messages = append(messages, m)
				}
			}
		}
	}

	return messages
}
