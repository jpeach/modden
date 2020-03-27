package utils

import (
	"math/rand"
	"strings"
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

func TestJoinLines(t *testing.T) {
	lines := []string{"one", "two", "three"}
	assert.Equal(t, strings.Join(lines, "\n"), JoinLines(lines...))
}
