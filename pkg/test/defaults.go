package test

import (
	"github.com/jpeach/modden/pkg/builtin"
	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/must"
)

func DefaultObjectUpdateCheck() *doc.Fragment {
	data := must.Bytes(builtin.Asset("pkg/builtin/objectCheck.rego"))

	frag, err := doc.NewRegoFragment(data)
	if err != nil {
		// TODO(jpeach): send to test listener.
		panic(err)
	}

	return frag
}
