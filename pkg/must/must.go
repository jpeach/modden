package must

// Bytes panics if the error is set, otherwise returns b.
func Bytes(b []byte, err error) []byte {
	if err != nil {
		panic(err.Error())
	}

	return b
}
