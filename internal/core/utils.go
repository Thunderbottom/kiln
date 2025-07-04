package core

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)

	return err == nil
}

// ReadFile reads a file and returns data
func ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// WriteFile writes data to a file with secure permissions
func WriteFile(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0o600)
}

// WipeData securely clears sensitive data from a byte slice
func WipeData(data []byte) {
	if data == nil {
		return
	}

	for i := range data {
		data[i] = 0
	}

	// Prevent compiler optimizations and trigger GC to clear memory copies
	runtime.KeepAlive(data)
	runtime.GC()
}

// SortedKeys returns sorted keys from environment variables map
func SortedKeys(variables map[string][]byte) []string {
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
