package test

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestPathforResource(t *testing.T) {
	assert.Equal(t,
		pathForResource("pods",
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "one",
						"namespace": "system",
					},
				},
			}),
		"/resources/system/pods/one",
	)

	assert.Equal(t,
		pathForResource("services",
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "two",
						"namespace": "default",
					},
				},
			}),
		"/resources/services/two",
	)
}
