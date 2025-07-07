package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	err := ValidationError("variable name", "must start with letter or underscore")
	expected := "invalid variable name: must start with letter or underscore"

	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestConfigError(t *testing.T) {
	err := ConfigError("no recipients", "add recipients to config")
	expected := "configuration error: no recipients (add recipients to config)"

	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestSecurityError(t *testing.T) {
	err := SecurityError("access denied", "check permissions")
	expected := "security error: access denied (check permissions)"

	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestInputError(t *testing.T) {
	err := InputError("test-input", "invalid format", "use correct format")
	expected := "invalid input 'test-input': invalid format (use correct format)"

	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestOperationError(t *testing.T) {
	baseErr := errors.New("file not found")
	err := OperationError("read", "config.toml", baseErr)

	if !strings.Contains(err.Error(), "read config.toml") {
		t.Errorf("OperationError should contain operation and resource")
	}

	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("OperationError should wrap original error")
	}
}

func TestFileAccessError(t *testing.T) {
	baseErr := errors.New("permission denied")
	err := FileAccessError("write", "test.env", baseErr)

	if !strings.Contains(err.Error(), "write file 'test.env'") {
		t.Errorf("FileAccessError should format file path correctly")
	}
}
