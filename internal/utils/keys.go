package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadPrivateKey attempts to load the private key from common locations
func LoadPrivateKey(ctx context.Context) (string, error) {
	// Try environment variable first
	if key := os.Getenv("KILN_PRIVATE_KEY"); key != "" {
		key = strings.TrimSpace(key)
		if key != "" {
			return key, nil
		}
	}

	// Try environment variable pointing to file
	if keyFile := os.Getenv("KILN_PRIVATE_KEY_FILE"); keyFile != "" {
		keyFile = ExpandPath(keyFile)
		data, err := readFileWithContext(ctx, keyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read private key file %s: %w", keyFile, err)
		}

		key := strings.TrimSpace(string(data))
		WipeData(data) // Clear file data
		if key != "" {
			return key, nil
		}
	}

	// Try common file locations
	locations := getKeyLocations()
	for _, location := range locations {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		data, err := readFileWithContext(ctx, location)
		if err == nil {
			key := strings.TrimSpace(string(data))
			WipeData(data) // Clear file data
			if key != "" {
				return key, nil
			}
		}
	}

	return "", fmt.Errorf(`private key not found

Searched locations:
  Environment variable: KILN_PRIVATE_KEY
  Environment file: KILN_PRIVATE_KEY_FILE
  Files: %s

Solutions:
  1. Set KILN_PRIVATE_KEY environment variable with your private key
  2. Set KILN_PRIVATE_KEY_FILE to point to your key file
  3. Place your private key in one of the searched locations
  4. Run 'kiln init' to generate a new key pair`, strings.Join(locations, ", "))
}

// getKeyLocations returns common locations to search for private keys
func getKeyLocations() []string {
	locations := []string{
		"kiln.key",
		".kiln.key",
	}

	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations,
			filepath.Join(home, ".config", "kiln", "kiln.key"),
			filepath.Join(home, ".kiln", "kiln.key"),
		)
	}

	return locations
}

// readFileWithContext reads a file with context cancellation support
func readFileWithContext(ctx context.Context, path string) ([]byte, error) {
	done := make(chan struct{})
	var data []byte
	var err error

	go func() {
		defer close(done)
		data, err = os.ReadFile(path)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return data, err
	}
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if len(path) == 1 {
		return home
	}

	if path[1] == '/' || path[1] == filepath.Separator {
		return filepath.Join(home, path[2:])
	}

	return path
}

// SavePrivateKey saves a private key to a file with secure permissions
func SavePrivateKey(privateKey, filename string) error {
	// Create directory if needed
	if dir := filepath.Dir(filename); dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create file with restrictive permissions
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer file.Close()

	// Write key with newline
	if _, err := file.WriteString(privateKey + "\n"); err != nil {
		os.Remove(filename) // Clean up on write failure
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// SaveFile writes data to a file with secure permissions
func SaveFile(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(filename)+"-")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Cleanup on failure
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Set permissions before writing
	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	// Prevent cleanup
	tmpFile = nil

	// Atomic rename
	if err := os.Rename(tmpPath, filename); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
