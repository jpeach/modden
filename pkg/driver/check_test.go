package driver

import (
	"testing"

	"github.com/open-policy-agent/opa/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, text string) *ast.Module {
	t.Helper()

	m, err := ast.ParseModule("test", text)
	if err != nil {
		t.Fatalf("failed to parse module: %s", err)
	}

	return m
}

func TestQueryStringResult(t *testing.T) {
	r := NewRegoDriver()

	results, err := r.Eval(parse(t,
		` package test

warn[msg] { msg = "this is the first warning"}
warn[msg] { msg = "this is the second warning"}
error[msg] { msg = "this is the error"}
fatal[msg] { msg = "this is the fatal error"}
`))

	require.NoError(t, err)

	expected := []CheckResult{{
		Severity: SeverityWarn,
		Message:  "this is the first warning",
	}, {
		Severity: SeverityWarn,
		Message:  "this is the second warning",
	}, {
		Severity: SeverityError,
		Message:  "this is the error",
	}, {
		Severity: SeverityFatal,
		Message:  "this is the fatal error",
	}}

	assert.ElementsMatch(t, expected, results)
}

func TestQueryMapResult(t *testing.T) {
	r := NewRegoDriver()

	results, err := r.Eval(parse(t,
		` package test

error [{"msg": msg, "foo": "bar"}] { msg = "this is the nested error"}
`))

	require.NoError(t, err)

	expected := []CheckResult{{
		Severity: SeverityError,
		Message:  "this is the nested error",
	}}

	assert.ElementsMatch(t, expected, results)
}

func TestQueryBoolResult(t *testing.T) {
	r := NewRegoDriver()

	results, err := r.Eval(parse(t,
		` package test

error  { msg = "this error doesn't appear'"}
`))

	require.NoError(t, err)

	expected := []CheckResult{{
		Severity: SeverityError,
		Message:  "",
	}}

	assert.ElementsMatch(t, expected, results)
}

func TestQueryUntypedResult(t *testing.T) {
	r := NewRegoDriver()

	results, err := r.Eval(parse(t,
		` package foo

sites := [
    {"count": 1},
    {"count": 2},
    {"count": 3},
]

error[num] { num := sites[_].count }
`))

	require.NoError(t, err)

	// We expect no results because the type of the result will be []int, which is not supported.
	assert.ElementsMatch(t, []CheckResult{}, results)
}
