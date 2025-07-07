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
		return nil, fmt.Errorf("read private key: %w", err)
	}
	defer WipeData(data)

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("private key file is empty")
	}

	if bytes.Contains(trimmed, []byte("age-encryption.org/v1")) {
		fmt.Println("Private key is passphrase-protected")

		decryptedKey, err := decryptPrivateKey(string(trimmed))
		if err != nil {
			return nil, fmt.Errorf("decrypt private key: %w", err)
		}
		defer WipeData(decryptedKey)

		result := make([]byte, len(decryptedKey))
		copy(result, decryptedKey)

		return result, nil
	}

	result := make([]byte, len(trimmed))
	copy(result, trimmed)

	return result, nil
}

// LoadPublicKey loads a public key from either a string or file path
func LoadPublicKey(input string) (string, error) {
	if IsValidPublicKey(input) {
		return input, nil
	}

	if !FileExists(input) {
		return "", fmt.Errorf("file is neither a valid public key nor a readable file")
	}

	data, err := ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", input, err)
	}
	defer WipeData(data)

	content := strings.TrimSpace(string(data))

	if IsValidPublicKey(content) {
		return content, nil
	}

	if IsPrivateKey(content) {
		identity, err := age.ParseX25519Identity(content)
		if err != nil {
			return "", fmt.Errorf("invalid private key format: %w", err)
		}

		return identity.Recipient().String(), nil
	}

	if strings.Contains(content, "age-encryption.org/v1") {
		fmt.Println("Private key is passphrase-protected")

		decryptedKey, err := decryptPrivateKey(content)
		if err != nil {
			return "", fmt.Errorf("decrypt private key: %w", err)
		}
		defer WipeData(decryptedKey)

		identity, err := age.ParseX25519Identity(string(decryptedKey))
		if err != nil {
			return "", fmt.Errorf("invalid decrypted private key format: %w", err)
		}

		return identity.Recipient().String(), nil
	}

	return "", fmt.Errorf("file does not contain a valid age key")
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

	passphrase, err := term.ReadPassword(syscall.Stdin)

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
		return nil, fmt.Errorf("write private key: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	return buf.Bytes(), nil
}

func decryptPrivateKey(encryptedKey string) ([]byte, error) {
	fmt.Print("Enter passphrase: ")

	passphrase, err := term.ReadPassword(syscall.Stdin)

	fmt.Println()

	if err != nil {
		return nil, fmt.Errorf("read passphrase: %w", err)
	}

	defer WipeData(passphrase)

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
	data := make([]byte, len(privateKey)+1)
	copy(data, privateKey)
	data[len(privateKey)] = '\n'

	return WriteFile(filename, data)
}
