package test

import (
	"github.com/jpeach/modden/pkg/builtin"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/must"

	"github.com/open-policy-agent/opa/ast"
)

// DefaultObjectCheckForOperation returns a built-in default check
// for applying Kubernetes objects.
func DefaultObjectCheckForOperation(op driver.ObjectOperationType) *ast.Module {
	var data []byte
	var name string

	switch op {
	case driver.ObjectOperationUpdate:
		name = "pkg/builtin/objectUpdateCheck.rego"
	case driver.ObjectOperationDelete:
		name = "pkg/builtin/objectDeleteCheck.rego"
	}

	data = must.Bytes(builtin.Asset(name))
	return must.Module(ast.ParseModule(name, string(data)))
}
