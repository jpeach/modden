package utils

import (
	"math/rand"
	"strings"
)

const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandomStringN ...
func RandomStringN(length int) string {
	if length < 1 {
		return ""
	}

	result := make([]byte, length)

	for i := range result {
		result[i] = alpha[rand.Int()%len(alpha)] //nolint(gosec)
	}

	return string(result)
}

// ContainsString checks whether the wanted string is in the values
// slice. This is suitable for short, unsorted slices.
func ContainsString(values []string, wanted string) bool {
	for _, v := range values {
		if v == wanted {
			return true
		}
	}

	return false
}

// JoinLines joins the given strings with "\n".
func JoinLines(lines ...string) string {
	switch len(lines) {
	case 0:
		return ""
	default:
		return strings.Join(lines, "\n")
	}
}
