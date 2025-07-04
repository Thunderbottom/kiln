package crypto

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// SecureBuffer provides secure memory handling for sensitive data
type SecureBuffer struct {
	data   []byte
	size   int
	locked bool
}

// NewSecureBuffer creates a new secure buffer with the specified size
func NewSecureBuffer(size int) (*SecureBuffer, error) {
	if size <= 0 {
		return nil, ErrInvalidBufferSize
	}

	data := make([]byte, size)

	sb := &SecureBuffer{
		data: data,
		size: size,
	}

	// Try to lock memory to prevent swapping
	if err := sb.lock(); err != nil {
		// Non-fatal on some systems
		_ = err
	}

	// Set finalizer for cleanup
	runtime.SetFinalizer(sb, (*SecureBuffer).finalize)
	return sb, nil
}

// Write writes data to the secure buffer
func (sb *SecureBuffer) Write(data []byte) error {
	if len(data) > sb.size {
		return ErrBufferTooSmall
	}

	// Clear buffer first
	sb.clear()

	// Copy data
	copy(sb.data, data)
	return nil
}

// Read returns a copy of the buffer data
func (sb *SecureBuffer) Read() []byte {
	result := make([]byte, sb.size)
	copy(result, sb.data)
	return result
}

// Bytes returns the underlying byte slice (use with caution)
func (sb *SecureBuffer) Bytes() []byte {
	return sb.data
}

// Size returns the buffer size
func (sb *SecureBuffer) Size() int {
	return sb.size
}

// Clear securely clears the buffer contents
func (sb *SecureBuffer) Clear() {
	sb.clear()
}

// Close cleans up the secure buffer
func (sb *SecureBuffer) Close() error {
	sb.clear()

	if sb.locked {
		if err := sb.unlock(); err != nil {
			return err
		}
	}

	// Remove finalizer
	runtime.SetFinalizer(sb, nil)
	return nil
}

// clear securely overwrites buffer contents
func (sb *SecureBuffer) clear() {
	if sb.data == nil {
		return
	}

	// Multiple-pass overwrite for paranoid security
	// Pass 1: zeros
	for i := range sb.data {
		sb.data[i] = 0
	}

	// Pass 2: random data
	if _, err := rand.Read(sb.data); err != nil {
		// Fallback to pattern if random fails
		for i := range sb.data {
			sb.data[i] = 0xFF
		}
	}

	// Pass 3: zeros again
	for i := range sb.data {
		sb.data[i] = 0
	}

	// Memory barrier to prevent compiler optimization
	runtime.KeepAlive(sb.data)
}

// lock attempts to lock memory pages to prevent swapping
func (sb *SecureBuffer) lock() error {
	if len(sb.data) == 0 {
		return nil
	}

	// Get page-aligned address and size
	pageSize := unix.Getpagesize()
	addr := uintptr(unsafe.Pointer(&sb.data[0]))
	alignedAddr := addr & ^(uintptr(pageSize) - 1)
	size := uintptr(len(sb.data)) + (addr - alignedAddr)

	// Round up to page boundary
	if size%uintptr(pageSize) != 0 {
		size += uintptr(pageSize) - (size % uintptr(pageSize))
	}

	err := unix.Mlock((*[1]byte)(unsafe.Pointer(alignedAddr))[:size])
	if err == nil {
		sb.locked = true
	}
	return err
}

// unlock unlocks previously locked memory
func (sb *SecureBuffer) unlock() error {
	if !sb.locked || len(sb.data) == 0 {
		return nil
	}

	pageSize := unix.Getpagesize()
	addr := uintptr(unsafe.Pointer(&sb.data[0]))
	alignedAddr := addr & ^(uintptr(pageSize) - 1)
	size := uintptr(len(sb.data)) + (addr - alignedAddr)

	if size%uintptr(pageSize) != 0 {
		size += uintptr(pageSize) - (size % uintptr(pageSize))
	}

	err := unix.Munlock((*[1]byte)(unsafe.Pointer(alignedAddr))[:size])
	if err == nil {
		sb.locked = false
	}
	return err
}

// finalize is called by the GC finalizer
func (sb *SecureBuffer) finalize() {
	_ = sb.Close()
}

// Errors
var (
	ErrBufferTooSmall    = fmt.Errorf("buffer too small")
	ErrInvalidBufferSize = fmt.Errorf("invalid buffer size")
)

// SecureString provides secure string handling
type SecureString struct {
	buffer *SecureBuffer
}

// NewSecureString creates a new secure string from bytes
func NewSecureString(data []byte) (*SecureString, error) {
	buffer, err := NewSecureBuffer(len(data))
	if err != nil {
		return nil, err
	}

	if err := buffer.Write(data); err != nil {
		buffer.Close()
		return nil, err
	}

	return &SecureString{buffer: buffer}, nil
}

// NewSecureStringFromString creates a new secure string from a regular string
func NewSecureStringFromString(s string) (*SecureString, error) {
	return NewSecureString([]byte(s))
}

// String returns the string value (creates a copy)
func (ss *SecureString) String() string {
	if ss.buffer == nil {
		return ""
	}
	return string(ss.buffer.Read())
}

// Bytes returns the byte slice value (creates a copy)
func (ss *SecureString) Bytes() []byte {
	if ss.buffer == nil {
		return nil
	}
	return ss.buffer.Read()
}

// Clear securely clears the string
func (ss *SecureString) Clear() {
	if ss.buffer != nil {
		ss.buffer.Clear()
	}
}

// Close cleans up the secure string
func (ss *SecureString) Close() error {
	if ss.buffer != nil {
		err := ss.buffer.Close()
		ss.buffer = nil
		return err
	}
	return nil
}

// Len returns the length of the secure string
func (ss *SecureString) Len() int {
	if ss.buffer == nil {
		return 0
	}
	return ss.buffer.Size()
}

// IsEmpty returns true if the secure string is empty
func (ss *SecureString) IsEmpty() bool {
	return ss.Len() == 0
}

// SecureCopy securely copies data and ensures source is cleared
func SecureCopy(dst, src []byte) int {
	n := copy(dst, src)

	// Clear source
	for i := range src {
		src[i] = 0
	}

	// Overwrite with random data
	rand.Read(src)

	// Clear again
	for i := range src {
		src[i] = 0
	}

	return n
}

// SecureCompare performs constant-time comparison of two byte slices
func SecureCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := range len(a) {
		result |= a[i] ^ b[i]
	}

	return result == 0
}
