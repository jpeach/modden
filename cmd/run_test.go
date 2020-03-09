package cmd

import (
	"testing"

	"github.com/jpeach/modden/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestParamValidation(t *testing.T) {
	opts, err := validateParams([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []test.RunOpt{}, opts)

	opts, err = validateParams([]string{"foo"})
	assert.Error(t, err)
	assert.Equal(t, []test.RunOpt(nil), opts)

	opts, err = validateParams([]string{"foo=bar=baz=fizz"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(opts))

	opts, err = validateParams([]string{"foo=bar", "foo=bar"})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(opts))
}
