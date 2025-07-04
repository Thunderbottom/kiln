package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var validVarNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// IsValidVarName validates environment variable names using uppercase letters, numbers, and underscores.
func IsValidVarName(name string) bool {
	return name != "" && validVarNameRegex.MatchString(name)
}

// IsValidFileName validates file names to prevent directory traversal attacks.
func IsValidFileName(name string) bool {
	if name == "" {
		return false
	}

	return !strings.Contains(name, "..") && !strings.Contains(name, "/")
}

// IsValidFilePath validates file paths to prevent directory traversal attacks.
func IsValidFilePath(path string) bool {
	if path == "" {
		return false
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	if strings.Contains(abs, "..") {
		return false
	}

	return true
}

// IsValidTimeout validates command execution timeout values
func IsValidTimeout(timeout time.Duration) bool {
	return timeout > 0 && timeout <= 24*time.Hour
}

// IsValidEditor checks if the specified editor is executable
func IsValidEditor(editor string) error {
	if editor == "" {
		return fmt.Errorf("editor cannot be empty")
	}

	if strings.Contains(editor, "..") {
		return fmt.Errorf("editor path cannot contain '..'")
	}

	_, err := exec.LookPath(editor)
	if err != nil {
		return fmt.Errorf("editor '%s' not found in PATH", editor)
	}

	return nil
}

// IsValidEnvValue validates environment variable value content
func IsValidEnvValue(value []byte) error {
	if len(value) > 1048576 { // 1MB limit
		return fmt.Errorf("value too large (max 1MB)")
	}

	for i, b := range value {
		if b == 0 {
			return fmt.Errorf("null byte at position %d", i)
		}

		if b < 32 && b != 9 && b != 10 && b != 13 {
			return fmt.Errorf("invalid control character at position %d", i)
		}
	}

	return nil
}

// IsValidWorkingDirectory validates working directory path and existence
func IsValidWorkingDirectory(path string) error {
	if !IsValidFilePath(path) {
		return fmt.Errorf("invalid directory path")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("directory does not exist")
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	return nil
}

// SanitizeEnvValue removes dangerous content from environment variable values
func SanitizeEnvValue(value []byte) []byte {
	result := make([]byte, 0, len(value))

	for _, b := range value {
		if b >= 32 || b == 9 || b == 10 || b == 13 {
			result = append(result, b)
		}
	}

	return result
}

// IsValidCommand validates command arguments for potential injection
func IsValidCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	for i, arg := range args {
		if len(arg) > 4096 {
			return fmt.Errorf("argument %d too long (max 4096 chars)", i)
		}

		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("argument %d contains null byte", i)
		}
	}

	return nil
}
