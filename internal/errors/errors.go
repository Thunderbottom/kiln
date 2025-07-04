package errors

import (
	"errors"
	"fmt"
)

// Common error types
var (
	ErrConfigNotFound     = errors.New("kiln configuration not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrInvalidKey         = errors.New("invalid age key")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrFileNotFound       = errors.New("file not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrInvalidEnvironment = errors.New("invalid environment data")
	ErrPrivateKeyNotFound = errors.New("private key not found")
	ErrUnsupportedFormat  = errors.New("unsupported format")
)

// KilnError represents a kiln-specific error with context
type KilnError struct {
	Op  string // operation that failed
	Err error  // underlying error
}

func (e *KilnError) Error() string {
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *KilnError) Unwrap() error {
	return e.Err
}

// New creates a new KilnError
func New(op string, err error) error {
	if err == nil {
		return nil
	}
	return &KilnError{Op: op, Err: err}
}

// Wrap wraps an error with a message
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf wraps an error with a formatted message
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
