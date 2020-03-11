package utils

import "os"

// IsDirPath returns true if path refers to a directory.
func IsDirPath(path string) bool {
	if info, err := os.Stat(path); err == nil {
		return info.IsDir()
	}

	return false
}
