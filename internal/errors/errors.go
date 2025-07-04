package errors

import (
	"errors"
	"fmt"
)

var (
	ErrConfigNotFound     = errors.New("kiln configuration not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrInvalidKey         = errors.New("invalid age key")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrFileNotFound       = errors.New("file not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrInvalidEnvironment = errors.New("invalid environment data")
)

func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
