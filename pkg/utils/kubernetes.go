package utils

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// NamespaceOrDefault returns the namespace from u, or "default" if u
// has no namespace field.
func NamespaceOrDefault(u *unstructured.Unstructured) string {
	if ns := u.GetNamespace(); ns != "" {
		return ns
	}

	return "default"
}
