package utils

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrChainUnwrap(t *testing.T) {
	e := ChainErrors(
		errors.New("one"),
		errors.New("two"),
		errors.New("three"),
	)

	assert.Equal(t, e.Error(), "one")

	e = errors.Unwrap(e)
	assert.Equal(t, e.Error(), "two")
	assert.NotNil(t, e)

	e = errors.Unwrap(e)
	assert.Equal(t, e.Error(), "three")
	assert.NotNil(t, e)

	e = errors.Unwrap(e)
	assert.Nil(t, e)
}

func TestErrChainAs(t *testing.T) {
	e := ChainErrors(
		errors.New("one"),
		os.ErrExist,
		&os.PathError{
			Op:   "test",
			Path: "/test/path",
			Err:  errors.New("tested error"),
		},
	)

	var pathError *os.PathError
	assert.True(t, errors.As(e, &pathError))

	var linkError *os.LinkError
	assert.False(t, errors.As(e, &linkError))
}

func TestErrChainIs(t *testing.T) {
	is := fmt.Errorf("this error is")
	isnot := fmt.Errorf("this error is not")

	e := ChainErrors(
		errors.New("one"),
		os.ErrExist,
		is,
	)

	assert.True(t, errors.Is(e, is))
	assert.True(t, errors.Is(e, os.ErrExist))
	assert.False(t, errors.Is(e, isnot))
}
