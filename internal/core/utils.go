package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"filippo.io/age"
	"golang.org/x/term"
)

// loadPrivateKey loads a private key from the specified path or default locations
func loadPrivateKey(keyPath string) ([]byte, error) {
	if keyPath == "" {
		if envPath := os.Getenv("KILN_PRIVATE_KEY_FILE"); envPath != "" {
			keyPath = envPath
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			keyPath = filepath.Join(home, ".kiln", "kiln.key")
		}
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %s: %w", keyPath, err)
	}
	defer WipeData(data)

	key := strings.TrimSpace(string(data))
	if key == "" {
		return nil, fmt.Errorf("private key file %s is empty", keyPath)
	}

	// Check if it's passphrase-protected
	if strings.Contains(key, "age-encryption.org/v1") {
		fmt.Printf("Private key at %s is passphrase-protected.\n", keyPath)
		return decryptPrivateKey(key)
	}

	return []byte(key), nil
}

// decryptPrivateKey decrypts an age-encrypted private key
func decryptPrivateKey(encryptedKey string) ([]byte, error) {
	fmt.Print("Enter passphrase: ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}
	defer WipeData(passphrase)

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to create scrypt identity: %w", err)
	}

	reader := strings.NewReader(encryptedKey)
	r, err := age.Decrypt(reader, identity)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted private key: %w", err)
	}

	return decrypted, nil
}

// SavePrivateKey saves a private key to a file with secure permissions
func SavePrivateKey(privateKey []byte, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Add newline to the []byte data
	data := append(privateKey, '\n')
	return os.WriteFile(filename, data, 0600)
}

// saveFile writes data to a file with secure permissions
func saveFile(filename string, data []byte) error {
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

// WipeData securely clears sensitive data from a byte slice
func WipeData(data []byte) {
	if data == nil {
		return
	}
	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
}

// WipeString securely clears a string from memory
func WipeString(s string) {
	if s == "" {
		return
	}
	// Convert to byte slice and wipe
	b := []byte(s)
	WipeData(b)
}
