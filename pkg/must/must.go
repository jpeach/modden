package must

import "k8s.io/apimachinery/pkg/runtime/schema"

// Bytes panics if the error is set, otherwise returns b.
func Bytes(b []byte, err error) []byte {
	if err != nil {
		panic(err.Error())
	}

	return b
}

// GroupVersion panics if the error is set, otherwise returns b.
func GroupVersion(gv schema.GroupVersion, err error) schema.GroupVersion {
	if err != nil {
		panic(err.Error())
	}

	return gv
}

func String(s string, err error) string {
	if err != nil {
		panic(err.Error())
	}

	return s
}
