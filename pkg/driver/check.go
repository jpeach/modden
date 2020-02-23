package driver

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
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

// CheckDriver is a driver for running Rego policy checks.
type CheckDriver interface {
	// Store returns a pointer to the driver's Rego data store.
	Store() storage.Store
	// Eval evaluates the given module and returns and check results.
	Eval(*ast.Module) ([]CheckResult, error)
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
	store storage.Store
}

func (r *regoDriver) Store() storage.Store {
	return r.store
}

// Eval evaluates checks in the given module.
func (r *regoDriver) Eval(m *ast.Module) ([]CheckResult, error) {
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

		r := rego.New(
			// Scope the query to the current module package.
			rego.Package(pkg),
			// Query for the result of this named rule.
			rego.Query(queryForRuleName(name)),
			rego.Compiler(c),
			rego.ParsedModule(m),
			rego.Store(r.Store()),

			// TODO(jpeach): if tracing is configured, add rego.Tracer().
		)

		resultSet, err := r.Eval(context.Background())
		if err != nil {
			// TODO(jpeach): propagate fatal error result.
			return nil, err
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
