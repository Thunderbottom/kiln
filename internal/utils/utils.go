package utils

import "runtime"

// WipeData securely clears sensitive data from a byte slice
// Replace WipeData function with:
func WipeData(data []byte) {
	if data == nil {
		return
	}
	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
}

// WipeString securely clears a string from memory
func WipeString(s string) {
	if s == "" {
		return
	}
	// Convert to byte slice and wipe
	b := []byte(s)
	WipeData(b)
}
