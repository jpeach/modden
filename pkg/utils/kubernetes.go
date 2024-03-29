package utils

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

// ImmediateDeletionOptions returns metav1.DeleteOptions specifying
// that the caller requires immediate foreground deletion semantics.
func ImmediateDeletionOptions() *metav1.DeleteOptions {
	fg := metav1.DeletePropagationForeground

	return &metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0),
		PropagationPolicy:  &fg,
	}
}

// NamespaceOrDefault returns the namespace from u, or "default" if u
// has no namespace field.
func NamespaceOrDefault(u *unstructured.Unstructured) string {
	if ns := u.GetNamespace(); ns != "" {
		return ns
	}

	return metav1.NamespaceDefault
}

// NewSelectorFromObject creates a selector to match all the labels in u.
func NewSelectorFromObject(u *unstructured.Unstructured) labels.Selector {
	return labels.SelectorFromSet(labels.Set(u.GetLabels()))
}

// SplitObjectName splits a string into namespace and name.
func SplitObjectName(fullName string) (string, string) {
	parts := strings.SplitN(fullName, "/", 2)
	switch len(parts) {
	case 1:
		return metav1.NamespaceDefault, parts[0]
	case 2:
		return parts[0], parts[1]
	default:
		panic(fmt.Sprintf("failed to split %q", fullName))
	}
}
