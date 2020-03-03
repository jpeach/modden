package utils

// CopyBytes duplicates a slice of bytes.
func CopyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
