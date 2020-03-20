package utils

import (
	"errors"
	"testing"

	"github.com/open-policy-agent/opa/topdown"
	"github.com/stretchr/testify/assert"
)

func TestAsRegoTopdownErr(t *testing.T) {
	assert.Nil(t, AsRegoTopdownErr(nil))

	e := &topdown.Error{Code: topdown.BuiltinErr}
	assert.Equal(t, e, AsRegoTopdownErr(e))

	assert.Equal(t, e, AsRegoTopdownErr(
		ChainErrors(errors.New("top"), e)))
}
