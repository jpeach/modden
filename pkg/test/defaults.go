package test

import (
	"github.com/jpeach/modden/pkg/builtin"
	"github.com/jpeach/modden/pkg/doc"
)

func DefaultObjectUpdateCheck() *doc.Fragment {
	data, err := builtin.Asset("pkg/builtin/objectCheck.rego")
	if err != nil {
		// TODO(jpeach): send to test listener.
		panic(err)
	}

	frag, err := doc.NewRegoFragment(data)
	if err != nil {
		// TODO(jpeach): send to test listener.
		panic(err)
	}

	return frag
}
