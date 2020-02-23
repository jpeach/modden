package driver

import (
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

type ruleInfo struct {
	name     string
	prefix   string
	severity Severity
}

var rules = []ruleInfo{
	// The following rules cause a tet failure if they are ever true.
	{name: "warn", prefix: "warn_", severity: SeverityWarn},
	{name: "warning", prefix: "warning_", severity: SeverityWarn},
	{name: "error", prefix: "error_", severity: SeverityError},
	{name: "fatal", prefix: "fatal_", severity: SeverityFatal},
}

// matchRuleByName finds the ruleInfo that matches the given query
// name exactly, or by prefix.
func matchRuleByName(name string) *ruleInfo {
	for _, q := range rules {
		if name == q.name || strings.HasPrefix(name, q.prefix) {
			return &q
		}
	}

	return nil
}

// severityForRuleName returns the test severity for a given rule name.
func severityForRuleName(name string) Severity {
	if q := matchRuleByName(name); q != nil {
		return q.severity
	}

	return SeverityNone
}

// queryForRuleName returns a Rego query for the given rule name. This
// is currently a no-op, but is a placeholder for allowing non-identity
// queries against rules.
func queryForRuleName(name string) string {
	if q := matchRuleByName(name); q != nil {
		return name
	}

	return ""
}

// findAssertionRules searches the module for rules that match a
// test assertion severity.
func findAssertionRules(m *ast.Module) []string {
	// The rule names we match in a hash because the same rule
	// name can appear more than once in a policy document.
	found := map[string]struct{}{}

	for _, rule := range m.Rules {
		name := rule.Head.Name.String()

		if severityForRuleName(name) == SeverityNone {
			continue
		}

		found[name] = struct{}{}
	}

	// Flatten query names back into the slice.
	result := make([]string, 0, len(found))
	for k := range found {
		result = append(result, k)
	}

	return result
}
