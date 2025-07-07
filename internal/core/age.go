package core

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"filippo.io/age"
	"github.com/alecthomas/kong"
)

// AgeManager handles all Age encryption/decryption operations
type AgeManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

// NewAgeManager creates manager with public keys and private key
func NewAgeManager(publicKeys []string, privateKey []byte) (*AgeManager, error) {
	recipients, err := parseRecipients(publicKeys)
	if err != nil {
		return nil, err
	}

	identity, err := age.ParseX25519Identity(string(privateKey))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return &AgeManager{
		recipients: recipients,
		identities: []age.Identity{identity},
	}, nil
}

// Encrypt encrypts data using all configured recipients
func (am *AgeManager) Encrypt(data []byte) ([]byte, error) {
	if len(am.recipients) == 0 {
		return nil, fmt.Errorf("no recipients configured")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data to encrypt")
	}

	var buf bytes.Buffer

	w, err := age.Encrypt(&buf, am.recipients...)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	return buf.Bytes(), nil
}

// Decrypt decrypts data using configured identities
func (am *AgeManager) Decrypt(data []byte) ([]byte, error) {
	if len(am.identities) == 0 {
		return nil, fmt.Errorf("no identities configured")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data to decrypt")
	}

	r, err := age.Decrypt(bytes.NewReader(data), am.identities...)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return result, nil
}

// parseRecipients converts a slice of public key strings into age.Recipient objects for encryption.
func parseRecipients(publicKeys []string) ([]age.Recipient, error) {
	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("no public keys provided")
	}

	recipients := make([]age.Recipient, 0, len(publicKeys))

	for _, key := range publicKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if err := ValidatePublicKey(key); err != nil {
			return nil, fmt.Errorf("invalid public key %s: %w", key, err)
		}

		recipient, err := age.ParseX25519Recipient(key)
		if err != nil {
			return nil, fmt.Errorf("invalid public key %s: %w", key, err)
		}

		recipients = append(recipients, recipient)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no valid public keys found")
	}

	return recipients, nil
}

// ValidatePublicKey validates an Age public key format
func ValidatePublicKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("empty public key")
	}

	_, err := age.ParseX25519Recipient(key)
	if err != nil {
		return fmt.Errorf("invalid public key format: %w", err)
	}

	return nil
}

// IsValidPublicKey checks if a string is a valid age public key
func IsValidPublicKey(key string) bool {
	return ValidatePublicKey(key) == nil
}

// IsPrivateKey checks if a string looks like an age private key
func IsPrivateKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), "AGE-SECRET-KEY-")
}

// AgePublicKeyMapper is a Kong mapper that validates and resolves age public keys.
// It accepts either a direct age public key string or a file path containing a public key.
// If a file path is provided, it loads and validates the key from the file.
// This mapper enables the "agepubkey" type tag in Kong CLI definitions.
var AgePublicKeyMapper = kong.MapperFunc(func(ctx *kong.DecodeContext, target reflect.Value) error {
	var value string
	if err := ctx.Scan.PopValueInto("string", &value); err != nil {
		return err
	}

	// Try as direct public key first
	if IsValidPublicKey(value) {
		target.SetString(value)

		return nil
	}

	// Try loading from file
	publicKey, err := LoadPublicKey(value)
	if err != nil {
		return fmt.Errorf("invalid public key or file: %w", err)
	}

	target.SetString(publicKey)

	return nil
})
