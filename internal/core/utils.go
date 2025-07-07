package core

import (
	"os"
	"path/filepath"
	"runtime"
)

// saveFile writes data to a file with secure permissions
func saveFile(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0600)
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// WipeData securely clears sensitive data from a byte slice
func WipeData(data []byte) {
	if data == nil {
		return
	}
	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
}
