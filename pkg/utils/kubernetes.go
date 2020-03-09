package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

// NamespaceOrDefault returns the namespace from u, or "default" if u
// has no namespace field.
func NamespaceOrDefault(u *unstructured.Unstructured) string {
	if ns := u.GetNamespace(); ns != "" {
		return ns
	}

	return "default"
}

// NewSelectorFromObject creates a selector to match all the labels in u.
func NewSelectorFromObject(u *unstructured.Unstructured) labels.Selector {
	return labels.SelectorFromSet(labels.Set(u.GetLabels()))
}
