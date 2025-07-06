package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"filippo.io/age"
	"golang.org/x/term"
)

// LoadPrivateKey attempts to load the private key from common locations
func LoadPrivateKey(ctx context.Context) (string, error) {
	keyPath := os.Getenv("KILN_PRIVATE_KEY_FILE")
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		keyPath = filepath.Join(home, ".kiln", "kiln.key")
	}

	return loadPrivateKeyFromFile(ctx, keyPath)
}

// loadPrivateKeyFromFile loads a private key from a file, handling encryption if needed
func loadPrivateKeyFromFile(ctx context.Context, keyFile string) (string, error) {
	data, err := readFileWithContext(ctx, keyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file %s: %w", keyFile, err)
	}
	defer WipeData(data)

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("private key file %s is empty", keyFile)
	}

	// Check if it's an age-encrypted file (passphrase-protected identity file)
	if strings.Contains(key, "age-encryption.org/v1") {
		// This is a passphrase-encrypted identity file - decrypt it
		fmt.Printf("Private key at %s is passphrase-protected.\n", keyFile)
		return decryptPrivateKey(key)
	}

	// Plain private key
	return key, nil
}

// decryptPrivateKey decrypts an age-encrypted private key
func decryptPrivateKey(encryptedKey string) (string, error) {
	fmt.Print("Enter passphrase: ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	defer WipeData(passphrase)

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return "", fmt.Errorf("failed to create scrypt identity: %w", err)
	}

	reader := strings.NewReader(encryptedKey)
	r, err := age.Decrypt(reader, identity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted private key: %w", err)
	}
	defer WipeData(decrypted)

	return string(decrypted), nil
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
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(privateKey+"\n"), 0600)
}

// SaveFile writes data to a file with secure permissions
func SaveFile(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0600)
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// DecryptPrivateKeyFromContent decrypts a passphrase-encrypted private key
func DecryptPrivateKeyFromContent(encryptedContent string) (string, error) {
	return decryptPrivateKey(encryptedContent)
}

// LoadPrivateKeyFromFile loads a private key from a specific file path
func LoadPrivateKeyFromFile(ctx context.Context, keyFile string) (string, error) {
	return loadPrivateKeyFromFile(ctx, keyFile)
}
