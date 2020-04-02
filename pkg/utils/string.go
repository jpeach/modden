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

// AsStringSlice tries to coerce an interface that may actually be a []string.
func AsStringSlice(val interface{}) ([]string, bool) {
	switch val := val.(type) {
	case []string:
		return val, true
	case []interface{}:
		if len(val) == 0 {
			return nil, false
		}

		str := make([]string, len(val))

		for i := 0; i < len(str); i++ {
			if s, ok := val[i].(string); ok {
				str[i] = s
			} else {
				// If any element is not a string,
				// this isn't a string slice.
				return nil, false
			}
		}

		return str, true
	default:
		return nil, false
	}
}
