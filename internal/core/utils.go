package core

import (
	"fmt"
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

	tempFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp.*")
	if err != nil {
		return err
	}

	tempName := tempFile.Name()

	var renamed bool
	defer func() {
		if !renamed {
			if err := tempFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: close error: %v\n", err)
			}

			if err := os.Remove(tempName); err != nil {
				fmt.Fprintf(os.Stderr, "warning: remove error: %v\n", err)
			}
		}
	}()

	if err := tempFile.Chmod(0o600); err != nil {
		return err
	}

	if _, err := tempFile.Write(data); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempName, filename); err != nil {
		return err
	}

	renamed = true

	return nil
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
