package core

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// AgeManager handles all Age encryption/decryption operations
type AgeManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

// NewAgeManager creates manager with public keys and private key
func NewAgeManager(publicKeys []string, privateKey []byte) (*AgeManager, error) {
	// Validate and parse recipients
	recipients, err := parseRecipients(publicKeys)
	if err != nil {
		return nil, err
	}

	// Parse identity
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

// parseRecipients parses public keys into age recipients
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

	if !strings.HasPrefix(key, "age1") {
		return fmt.Errorf("public key must start with 'age1'")
	}

	if len(key) != 62 { // age1 + 58 characters
		return fmt.Errorf("invalid public key length")
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
