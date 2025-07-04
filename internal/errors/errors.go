// Package errors provides simple, semantic error constructors for kiln CLI operations.
package errors

import "fmt"

// ValidationError creates an error for invalid input validation.
func ValidationError(field, reason string) error {
	return fmt.Errorf("invalid %s: %s", field, reason)
}

// ConfigError creates an error for configuration problems with solutions.
func ConfigError(issue, suggestion string) error {
	return fmt.Errorf("configuration error: %s (%s)", issue, suggestion)
}

// SecurityError creates an error for security-related issues
func SecurityError(issue, suggestion string) error {
	return fmt.Errorf("security error: %s (%s)", issue, suggestion)
}

// InputError creates an error for invalid input with specific guidance
func InputError(input, issue, suggestion string) error {
	return fmt.Errorf("invalid input '%s': %s (%s)", input, issue, suggestion)
}

// OperationError creates a standardized error for failed operations
func OperationError(operation, resource string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s %s: %w", operation, resource, err)
}

// FileAccessError creates an error for file access issues
func FileAccessError(operation, filename string, err error) error {
	return OperationError(operation, fmt.Sprintf("file '%s'", filename), err)
}
