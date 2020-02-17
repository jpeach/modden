package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNamespace(t *testing.T) {
	u := NewNamespace("foo")

	assert.NotNil(t, u)
	assert.Equal(t, u.GetNamespace(), "")
	assert.Equal(t, u.GetName(), "foo")
	assert.Equal(t, u.GetKind(), "Namespace")
	assert.Equal(t, u.GetAPIVersion(), "v1")
}
