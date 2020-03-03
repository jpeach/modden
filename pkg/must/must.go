package must

import "k8s.io/apimachinery/pkg/runtime/schema"

// Must panics if the error is set.
func Must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

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

// String panics if the error is set, otherwise returns s.
func String(s string, err error) string {
	if err != nil {
		panic(err.Error())
	}

	return s
}
