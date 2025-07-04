package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thunderbottom/kiln/internal/errors"
)

// KeySource represents where a private key was loaded from
type KeySource string

const (
	KeySourceEnvVar    KeySource = "environment_variable"
	KeySourceEnvFile   KeySource = "environment_file"
	KeySourceLocalFile KeySource = "local_file"
	KeySourceHomeDir   KeySource = "home_directory"
	KeySourceConfigDir KeySource = "config_directory"
)

// KeyInfo contains information about a loaded private key
type KeyInfo struct {
	Source   KeySource
	Path     string
	LoadTime time.Time
}

// LoadPrivateKey attempts to load the private key with context support
func LoadPrivateKey(ctx context.Context) (string, *KeyInfo, error) {
	loadTime := time.Now()

	// Try environment variable first
	if key := os.Getenv("KILN_PRIVATE_KEY"); key != "" {
		key = strings.TrimSpace(key)
		if key != "" {
			return key, &KeyInfo{
				Source:   KeySourceEnvVar,
				Path:     "KILN_PRIVATE_KEY",
				LoadTime: loadTime,
			}, nil
		}
	}

	// Try environment variable pointing to file
	if keyFile := os.Getenv("KILN_PRIVATE_KEY_FILE"); keyFile != "" {
		keyFile = ExpandPath(keyFile)
		data, err := readFileWithContext(ctx, keyFile)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to read private key file %s", keyFile)
		}

		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, &KeyInfo{
				Source:   KeySourceEnvFile,
				Path:     keyFile,
				LoadTime: loadTime,
			}, nil
		}
	}

	// Try common locations
	candidates := getKeyFileCandidates()

	for _, candidate := range candidates {
		if candidate.path == "" {
			continue
		}

		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		data, err := readFileWithContext(ctx, candidate.path)
		if err == nil {
			key := strings.TrimSpace(string(data))
			if key != "" {
				return key, &KeyInfo{
					Source:   candidate.source,
					Path:     candidate.path,
					LoadTime: loadTime,
				}, nil
			}
		}
	}

	// Build helpful error message
	var searchPaths []string
	for _, candidate := range candidates {
		if candidate.path != "" {
			searchPaths = append(searchPaths, candidate.path)
		}
	}

	return "", nil, fmt.Errorf(`private key not found

Searched locations:
  Environment variable: KILN_PRIVATE_KEY
  Environment file: KILN_PRIVATE_KEY_FILE
  Files: %s

Solutions:
  1. Set KILN_PRIVATE_KEY environment variable with your private key
  2. Set KILN_PRIVATE_KEY_FILE to point to your key file
  3. Place your private key in one of the searched locations
  4. Run 'kiln init' to generate a new key pair`, strings.Join(searchPaths, ", "))
}

type keyCandidate struct {
	path   string
	source KeySource
}

func getKeyFileCandidates() []keyCandidate {
	candidates := []keyCandidate{
		{path: "kiln.key", source: KeySourceLocalFile},
		{path: ".kiln.key", source: KeySourceLocalFile},
	}

	// Add home directory paths
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			keyCandidate{
				path:   filepath.Join(home, ".config", "kiln", "kiln.key"),
				source: KeySourceConfigDir,
			},
			keyCandidate{
				path:   filepath.Join(home, ".config", "kiln", "key.txt"),
				source: KeySourceConfigDir,
			},
			keyCandidate{
				path:   filepath.Join(home, ".kiln", "key.txt"),
				source: KeySourceHomeDir,
			},
			keyCandidate{
				path:   filepath.Join(home, ".kiln", "kiln.key"),
				source: KeySourceHomeDir,
			},
		)
	}

	return candidates
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

// CreateSecureTempFile creates a temporary file with restrictive permissions
func CreateSecureTempFile(pattern string) (string, error) {
	// Try to use memory-backed filesystem first
	tempDirs := []string{
		"/dev/shm",   // Linux tmpfs
		"/tmp",       // Traditional tmp
		os.TempDir(), // Go's default
	}

	var lastErr error
	for _, dir := range tempDirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			lastErr = err
			continue
		}

		filename := file.Name()
		file.Close()

		// Set restrictive permissions
		if err := os.Chmod(filename, 0600); err != nil {
			os.Remove(filename)
			lastErr = err
			continue
		}

		return filename, nil
	}

	if lastErr != nil {
		return "", errors.Wrap(lastErr, "failed to create secure temporary file")
	}

	return "", errors.New("utils.CreateSecureTempFile",
		fmt.Errorf("no suitable temporary directory found"))
}

// SavePrivateKey saves a private key to a file with secure permissions
func SavePrivateKey(privateKey, filename string) error {
	// Ensure directory exists
	if dir := filepath.Dir(filename); dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return errors.New("utils.SavePrivateKey", err)
		}
	}

	// Create file with restrictive permissions
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("utils.SavePrivateKey", err)
	}
	defer file.Close()

	// Write key with newline
	if _, err := file.WriteString(privateKey + "\n"); err != nil {
		// Clean up on write failure
		os.Remove(filename)
		return errors.New("utils.SavePrivateKey", err)
	}

	return nil
}

// ValidateFilePath validates a file path for security
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty file path")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal")
	}

	// Check for absolute paths outside allowed directories
	if filepath.IsAbs(cleanPath) {
		home, _ := os.UserHomeDir()
		allowed := []string{
			home,
			os.TempDir(),
			"/dev/shm",
		}

		allowed = append(allowed, getCurrentDir())

		isAllowed := false
		for _, allowedDir := range allowed {
			if allowedDir != "" {
				rel, err := filepath.Rel(allowedDir, cleanPath)
				if err == nil && !strings.HasPrefix(rel, "..") {
					isAllowed = true
					break
				}
			}
		}

		if !isAllowed {
			return fmt.Errorf("absolute path not in allowed directory")
		}
	}

	return nil
}

func getCurrentDir() string {
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// EnsureDirectoryExists creates a directory with secure permissions if it doesn't exist
func EnsureDirectoryExists(path string) error {
	if err := ValidateFilePath(path); err != nil {
		return errors.Wrap(err, "invalid directory path")
	}

	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", path)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to check directory")
	}

	// Create directory with restrictive permissions
	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	return nil
}

// SecureDelete attempts to securely delete a file
func SecureDelete(path string) error {
	if err := ValidateFilePath(path); err != nil {
		return errors.Wrap(err, "invalid file path")
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return errors.Wrap(err, "failed to stat file")
	}

	if info.IsDir() {
		return fmt.Errorf("cannot securely delete directory: %s", path)
	}

	// Overwrite file with random data before deletion
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		// If we can't open for writing, just delete normally
		return os.Remove(path)
	}
	defer file.Close()

	// Overwrite with zeros
	size := info.Size()
	zeros := make([]byte, size)
	if _, err := file.WriteAt(zeros, 0); err != nil {
		// If overwrite fails, still try to delete
		os.Remove(path)
		return errors.Wrap(err, "failed to overwrite file")
	}

	// Sync to disk
	file.Sync()
	file.Close()

	// Delete the file
	return os.Remove(path)
}

// SaveFile writes data to a file with secure permissions
func SaveFile(filename string, data []byte) error {
	if err := ValidateFilePath(filename); err != nil {
		return errors.Wrap(err, "invalid file path")
	}

	// Ensure directory exists
	if dir := filepath.Dir(filename); dir != "." {
		if err := EnsureDirectoryExists(dir); err != nil {
			return err
		}
	}

	// Write file with secure permissions
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return errors.New("utils.SaveFile", err)
	}

	return nil
}
