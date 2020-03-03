package driver

import (
	"context"
	"testing"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
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

	// We expect no results because the type of the result will
	// be []int, which is not supported.
	assert.ElementsMatch(t, []CheckResult{}, results)
}

func TestStorePathItem(t *testing.T) {
	// Use the underlying Rego driver type so we can directly access the Store.
	r := &regoDriver{
		store: inmem.New(),
	}

	ctx := context.TODO()

	// Creating the same path twice is not an error.
	assert.NoError(t, r.StorePath("/test/path/one"))
	assert.NoError(t, r.StorePath("/test/path/one"))

	storedValue := map[string]interface{}{
		"item": map[string]interface{}{
			"first":  "one",
			"second": "two",
		},
	}

	read := func(where string) (interface{}, error) {
		txn := storage.NewTransactionOrDie(ctx, r.store)
		defer r.store.Abort(ctx, txn)

		// Ensure that we can read it back.
		return r.store.Read(ctx, txn, storage.MustParsePath(where))
	}

	// Store an item.
	assert.NoError(t, r.StoreItem("/test/path/two", storedValue))

	// Ensure that we can read it back.
	val, err := read("/test/path/two")
	require.NoError(t, err, "reading store path %q", "/test/path/two")
	assert.Equal(t, storedValue, val)

	// Now re-store the path.
	assert.NoError(t, r.StorePath("/test/path/two"))

	// Ensure that extending the path didn't nuke the value
	val, err = read("/test/path/two")
	require.NoError(t, err, "reading store path %q", "/test/path/two")
	assert.Equal(t, storedValue, val)

	updatedValue := map[string]interface{}{
		"item": map[string]interface{}{
			"first":  "one",
			"second": "two",
			"third":  map[string]interface{}{},
		},
	}

	// Now store a path that traverses the existing item.
	assert.NoError(t, r.StorePath("/test/path/two/item/third"))

	// Ensure that extending the path didn't nuke the value, but created a new field.
	val, err = read("/test/path/two")
	require.NoError(t, err, "reading store path %q", "/test/path/two")
	assert.Equal(t, updatedValue, val)
}

func TestStoreRemoveItem(t *testing.T) {
	// Use the underlying Rego driver type so we can directly access the Store.
	r := &regoDriver{
		store: inmem.New(),
	}

	ctx := context.TODO()

	//nolint(unparam)
	read := func(where string) (interface{}, error) {
		txn := storage.NewTransactionOrDie(ctx, r.store)
		defer r.store.Abort(ctx, txn)

		// Ensure that we can read it back.
		return r.store.Read(ctx, txn, storage.MustParsePath(where))
	}

	assert.NoError(t, r.StorePath("/test/path/one"))

	// Ensure that we can read it back.
	_, err := read("/test/path/one")
	require.NoError(t, err, "reading store path %q", "/test/path/one")

	assert.NoError(t, r.RemovePath("/test/path/one"))

	// Ensure that it is gone can read it back.
	_, err = read("/test/path/one")
	require.True(t, storage.IsNotFound(err), "error is %s", err)

	assert.True(t, storage.IsNotFound(r.RemovePath("/no/such/path")))
}
