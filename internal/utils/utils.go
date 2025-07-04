package utils

import "runtime"

// WipeData securely clears sensitive data from a byte slice
func WipeData(data []byte) {
	if data == nil {
		return
	}

	// Clear with zeros
	for i := range data {
		data[i] = 0
	}

	// Prevent compiler optimization
	runtime.KeepAlive(data)
}

// WithWipeData executes a function with automatic clearing of byte data
func WithWipeData(data []byte, fn func([]byte) error) error {
	defer WipeData(data)
	return fn(data)
}

// WipeString securely clears a string by converting to bytes and clearing
func WipeString(s *string) {
	if s == nil || *s == "" {
		return
	}

	// Convert to byte slice and clear
	data := []byte(*s)
	WipeData(data)
	*s = ""
}
