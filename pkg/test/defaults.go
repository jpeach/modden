package test

import (
	"github.com/jpeach/modden/pkg/builtin"
	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/driver"
	"github.com/jpeach/modden/pkg/must"
)

// DefaultObjectUpdateCheck returns a built-in default check for applying
// Kubernetes objects.
func DefaultObjectCheckForOperation(op driver.ObjectOperationType) *doc.Fragment {
	var data []byte

	switch op {
	case driver.ObjectOperationUpdate:
		data = must.Bytes(builtin.Asset("pkg/builtin/objectUpdateCheck.rego"))
	case driver.ObjectOperationDelete:
		data = must.Bytes(builtin.Asset("pkg/builtin/objectDeleteCheck.rego"))
	}

	frag, err := doc.NewRegoFragment(data)
	if err != nil {
		// TODO(jpeach): send to test listener.
		panic(err)
	}

	return frag
}
