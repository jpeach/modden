package utils

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func NamespaceOrDefault(u *unstructured.Unstructured) string {
	if ns := u.GetNamespace(); ns != "" {
		return ns
	}

	return "default"
}
