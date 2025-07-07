package core

import (
	"bytes"
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
func LoadPrivateKey(keyPath string) ([]byte, error) {
	if keyPath == "" {
		if envPath := os.Getenv("KILN_PRIVATE_KEY_FILE"); envPath != "" {
			keyPath = envPath
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("get home directory: %w", err)
			}
			keyPath = filepath.Join(home, ".kiln", "kiln.key")
		}
	}

	data, err := ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key file %s: %w", keyPath, err)
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

// LoadPublicKey loads a public key from either a string or file path
func LoadPublicKey(input string) (string, error) {
	// Try as direct public key string first
	if IsValidPublicKey(input) {
		return input, nil
	}

	// Try as file path
	if !FileExists(input) {
		return "", fmt.Errorf("input %s is neither a valid public key nor a readable file", input)
	}

	data, err := ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", input, err)
	}
	defer WipeData(data)

	content := strings.TrimSpace(string(data))

	// Could be a public key
	if IsValidPublicKey(content) {
		return content, nil
	}

	// Could be a plain private key - extract public part
	if IsPrivateKey(content) {
		identity, err := age.ParseX25519Identity(content)
		if err != nil {
			return "", fmt.Errorf("invalid private key format in %s: %w", input, err)
		}
		return identity.Recipient().String(), nil
	}

	// Could be a passphrase-encrypted identity file
	if strings.Contains(content, "age-encryption.org/v1") {
		fmt.Printf("Private key at %s is passphrase-protected.\n", input)

		// Decrypt the private key
		decryptedKey, err := decryptPrivateKey(content)
		if err != nil {
			return "", fmt.Errorf("decrypt private key from %s: %w", input, err)
		}
		defer WipeData(decryptedKey)

		// Extract public key from decrypted private key
		identity, err := age.ParseX25519Identity(string(decryptedKey))
		if err != nil {
			return "", fmt.Errorf("invalid decrypted private key format in %s: %w", input, err)
		}
		return identity.Recipient().String(), nil
	}

	return "", fmt.Errorf("file %s does not contain a valid age key", input)
}

// GenerateKeyPair generates a new age key pair
func GenerateKeyPair() (privateKey []byte, publicKey string, err error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, "", fmt.Errorf("generate key pair: %w", err)
	}

	return []byte(identity.String()), identity.Recipient().String(), nil
}

// EncryptPrivateKey encrypts a private key using age's passphrase protection
func EncryptPrivateKey(privateKey []byte) ([]byte, error) {
	fmt.Print("Enter passphrase (leave empty to autogenerate): ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	defer WipeData(passphrase)

	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(privateKey); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decryptPrivateKey decrypts an age-encrypted private key
func decryptPrivateKey(encryptedKey string) ([]byte, error) {
	fmt.Print("Enter passphrase: ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("read passphrase: %w", err)
	}
	defer WipeData(passphrase)

	// Check for empty passphrase
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("passphrase cannot be empty")
	}

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, fmt.Errorf("create scrypt identity: %w", err)
	}

	reader := strings.NewReader(encryptedKey)
	r, err := age.Decrypt(reader, identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt private key: %w", err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decrypted private key: %w", err)
	}

	return decrypted, nil
}

// SavePrivateKey saves a private key to a file with secure permissions
func SavePrivateKey(privateKey []byte, filename string) error {
	// Add newline to the []byte data
	data := append(privateKey, '\n')
	return WriteFile(filename, data)
}
