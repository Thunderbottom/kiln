package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"filippo.io/age"
	"golang.org/x/term"
)

// LoadPrivateKey loads a private key from the specified path or default locations
func LoadPrivateKey(keyPath string) (string, error) {
	if keyPath == "" {
		if envPath := os.Getenv("KILN_PRIVATE_KEY_FILE"); envPath != "" {
			keyPath = envPath
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			keyPath = filepath.Join(home, ".kiln", "kiln.key")
		}
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file %s: %w", keyPath, err)
	}
	defer WipeData(data)

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("private key file %s is empty", keyPath)
	}

	// Check if it's passphrase-protected
	if strings.Contains(key, "age-encryption.org/v1") {
		fmt.Printf("Private key at %s is passphrase-protected.\n", keyPath)
		return DecryptPrivateKey(key)
	}

	return key, nil
}

// DecryptPrivateKey decrypts an age-encrypted private key
func DecryptPrivateKey(encryptedKey string) (string, error) {
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
