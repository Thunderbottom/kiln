package utils

import (
	"crypto/rand"
	"runtime"
)

// WipeData securely clears sensitive data from a byte slice
func WipeData(data []byte) {
	if data == nil {
		return
	}

	// Multiple passes with different patterns
	for i := range 3 {
		switch i {
		case 0:
			// Fill with zeros
			for j := range data {
				data[j] = 0
			}
		case 1:
			// Fill with 0xFF
			for j := range data {
				data[j] = 0xFF
			}
		case 2:
			// Fill with random data
			rand.Read(data)
		}
	}

	runtime.KeepAlive(data)
}

// WithWipeData executes a function with automatic clearing of byte data
func WithWipeData(data []byte, fn func([]byte) error) error {
	defer WipeData(data)
	return fn(data)
}
