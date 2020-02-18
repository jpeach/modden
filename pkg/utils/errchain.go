package utils

import (
	"errors"
)

type chain struct {
	err  error
	next *chain
}

func (c *chain) Unwrap() error {
	if c.next == nil {
		return nil
	}

	return c.next
}

func (c *chain) Is(target error) bool {
	return errors.Is(c.err, target)
}

func (c *chain) As(target interface{}) bool {
	return errors.As(c.err, target)
}

func (c *chain) Error() string {
	return c.err.Error()
}

// ChainErrors takes the slice of errors and constructs a single chained
// error from is. The captures errors can be retrieved by inspecting the
// result with errors.As and errors.Is.
func ChainErrors(errs ...error) error {
	var head *chain
	var tail *chain

	for _, e := range errs {
		if tail == nil {
			head = &chain{err: e}
			tail = head
		} else {
			tail.next = &chain{err: e}
			tail = tail.next
		}
	}

	return head
}
