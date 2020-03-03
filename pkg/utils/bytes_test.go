package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyBytes(t *testing.T) {
	src := []byte{'a', 'b', 'c'}
	dst := CopyBytes(src)

	assert.Equal(t, dst, src)

	dst[0] = 'e'
	dst[1] = 'f'
	dst[2] = 'g'

	assert.NotEqual(t, dst, src)
}
