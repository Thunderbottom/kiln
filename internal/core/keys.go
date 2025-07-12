package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"filippo.io/age"
	"golang.org/x/term"

	"github.com/thunderbottom/kiln/internal/config"
)

// LoadPrivateKey loads a private key from the specified path or default locations
func LoadPrivateKey(keyPath string) ([]byte, error) {
	if keyPath == "" {
		keyPath = GetDefaultKeyPath()
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

	// Handle encrypted age keys
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

	// Return unencrypted key
	result := make([]byte, len(trimmed))
	copy(result, trimmed)

	return result, nil
}

// GetDefaultKeyPath returns the first available key from default locations
// This is used only when no config is available
func GetDefaultKeyPath() string {
	candidates := GetPrivateKeyCandidates()

	for _, path := range candidates {
		if FileExists(path) {
			return path
		}
	}

	return ""
}

// LoadPublicKey loads a public key from either a string or file path
func LoadPublicKey(input string) (string, error) {
	if ValidatePublicKey(input) == nil {
		return input, nil
	}

	data, err := ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", input, err)
	}
	defer WipeData(data)

	content := strings.TrimSpace(string(data))

	if ValidatePublicKey(content) == nil {
		return content, nil
	}

	if !IsPrivateKey(content) {
		return "", fmt.Errorf("file does not contain a valid age key")
	}

	return extractPublicKeyFromPrivate(content)
}

// extractPublicKeyFromPrivate extracts public key from private key content
func extractPublicKeyFromPrivate(content string) (string, error) {
	// Handle encrypted private keys
	if strings.Contains(content, "age-encryption.org/v1") {
		return extractFromEncryptedPrivateKey(content)
	}

	// Handle unencrypted private keys
	return extractFromUnencryptedPrivateKey(content)
}

// extractFromEncryptedPrivateKey handles passphrase-protected keys
func extractFromEncryptedPrivateKey(content string) (string, error) {
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

// extractFromUnencryptedPrivateKey handles unencrypted private keys
func extractFromUnencryptedPrivateKey(content string) (string, error) {
	identity, err := age.ParseX25519Identity(content)
	if err != nil {
		return "", fmt.Errorf("invalid private key format: %w", err)
	}

	return identity.Recipient().String(), nil
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

	// Convert to int since syscall.Stdin is not int on Windows
	//nolint:unconvert
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
		return nil, fmt.Errorf("write private key: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	return buf.Bytes(), nil
}

// decryptPrivateKey decrypts a passphrase-protected age private key using user-provided passphrase.
func decryptPrivateKey(encryptedKey string) ([]byte, error) {
	fmt.Print("Enter passphrase: ")

	// Convert to int since syscall.Stdin is not int on Windows
	//nolint:unconvert
	passphrase, err := term.ReadPassword(int(syscall.Stdin))

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

// SaveKeys saves a private key and optionally its corresponding public key to files
func SaveKeys(privateKey []byte, publicKey, filename string) error {
	// Save private key if provided
	if len(privateKey) > 0 {
		data := make([]byte, len(privateKey)+1)
		copy(data, privateKey)
		data[len(privateKey)] = '\n'

		if err := WriteFile(filename, data); err != nil {
			return fmt.Errorf("save private key: %w", err)
		}
	}

	// Save public key if provided
	if publicKey != "" {
		pubKeyPath := filename + ".pub"
		pubKeyData := []byte(publicKey + "\n")

		if err := os.WriteFile(pubKeyPath, pubKeyData, 0o600); err != nil {
			return fmt.Errorf("save public key: %w", err)
		}
	}

	return nil
}

// FindPrivateKeyForConfig returns the best private key for the given configuration
func FindPrivateKeyForConfig(cfg *config.Config) (string, error) {
	// Environment variable takes precedence
	if envPath := os.Getenv("KILN_PRIVATE_KEY_FILE"); envPath != "" {
		if FileExists(envPath) {
			return envPath, nil
		}

		return "", fmt.Errorf("KILN_PRIVATE_KEY_FILE points to non-existent file: %s", envPath)
	}

	// Find compatible key for config recipients
	configPublicKeys := make([]string, 0, len(cfg.Recipients))
	for _, pubKey := range cfg.Recipients {
		configPublicKeys = append(configPublicKeys, strings.TrimSpace(pubKey))
	}

	candidates := GetPrivateKeyCandidates()
	for _, keyPath := range candidates {
		if !FileExists(keyPath) {
			continue
		}

		if keyMatchesAnyPublicKey(keyPath, configPublicKeys) {
			return keyPath, nil
		}
	}

	// Fallback to first available key
	for _, keyPath := range candidates {
		if FileExists(keyPath) {
			return keyPath, nil
		}
	}

	return "", fmt.Errorf("no private key found")
}

// keyMatchesAnyPublicKey checks if a private key corresponds to any public key
func keyMatchesAnyPublicKey(keyPath string, publicKeys []string) bool {
	if checkSSHKeyMatch(keyPath, publicKeys) {
		return true
	}

	return checkAgeKeyMatch(keyPath, publicKeys)
}

func checkSSHKeyMatch(keyPath string, publicKeys []string) bool {
	if !strings.Contains(keyPath, ".ssh/") {
		return false
	}

	pubPath := keyPath + ".pub"
	if !FileExists(pubPath) {
		return false
	}

	pubContent, err := os.ReadFile(pubPath)
	if err != nil {
		return false
	}

	pubKey := strings.TrimSpace(string(pubContent))

	return slices.Contains(publicKeys, pubKey)
}

func checkAgeKeyMatch(keyPath string, publicKeys []string) bool {
	privateKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		return false
	}
	defer WipeData(privateKey)

	if !strings.HasPrefix(strings.TrimSpace(string(privateKey)), "AGE-SECRET-KEY-") {
		return false
	}

	identity, err := age.ParseX25519Identity(string(privateKey))
	if err != nil {
		return false
	}

	derivedPubKey := identity.Recipient().String()

	return slices.Contains(publicKeys, derivedPubKey)
}

// GetPrivateKeyCandidates returns potential private key locations in discovery order
func GetPrivateKeyCandidates() []string {
	candidates := []string{}

	if envPath := os.Getenv("KILN_PRIVATE_KEY_FILE"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return candidates
	}

	candidates = append(candidates,
		filepath.Join(home, ".kiln", "kiln.key"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
	)

	return candidates
}
