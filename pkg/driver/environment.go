package driver

import "github.com/google/uuid"

type Environment interface {
	// UniqueID returns a unique identifier for this Environment instance.
	UniqueID() string
}

var _ Environment = &environ{}

type environ struct {
	uid string
}

// UniqueID returns a unique identifier for this Environment instance.
func (e *environ) UniqueID() string {
	return e.uid
}

func NewEnvironment() Environment {
	return &environ{
		uid: uuid.New().String(),
	}
}
