package utils

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomStringN(t *testing.T) {
	rand.Seed(1)

	assert.Equal(t, "", RandomStringN(-1))
	assert.Equal(t, "", RandomStringN(0))
	assert.Equal(t, "oJnNPG", RandomStringN(6))
	assert.Equal(t, "siuzytMOJPa", RandomStringN(11))
}
