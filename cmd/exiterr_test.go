package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitErr(t *testing.T) {
	err := ExitErrorf(EX_USAGE, "usage error")

	assert.Error(t, err)
	assert.Equal(t, "usage error", err.Error())

	var exit *ExitError
	assert.True(t, errors.As(err, &exit))
	assert.Equal(t, EX_USAGE, exit.Code)
}
