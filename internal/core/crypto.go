package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// ageManager handles Age encryption/decryption operations (unexported)
type ageManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

// newAgeManager creates a new Age manager with the given public keys (unexported)
func newAgeManager(publicKeys []string) (*ageManager, error) {
	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("no public keys provided")
	}

	recipients := make([]age.Recipient, 0, len(publicKeys))

	for _, key := range publicKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
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

	return &ageManager{
		recipients: recipients,
	}, nil
}

// addIdentity adds a private key identity for decryption (unexported)
func (am *ageManager) addIdentity(privateKey string) error {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return fmt.Errorf("empty private key")
	}

	identity, err := age.ParseX25519Identity(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	am.identities = append(am.identities, identity)
	return nil
}

// encrypt encrypts data using all configured recipients (unexported)
func (am *ageManager) encrypt(ctx context.Context, data []byte) ([]byte, error) {
	if len(am.recipients) == 0 {
		return nil, fmt.Errorf("no recipients configured")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data to encrypt")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, am.recipients...)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return buf.Bytes(), nil
}

// decrypt decrypts data using configured identities (unexported)
func (am *ageManager) decrypt(ctx context.Context, data []byte) ([]byte, error) {
	if len(am.identities) == 0 {
		return nil, fmt.Errorf("no identities configured")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data to decrypt")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r, err := age.Decrypt(bytes.NewReader(data), am.identities...)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return result, nil
}

// ValidatePublicKey validates an Age public key format (exported for init command)
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

// IsValidPublicKey checks if a string is a valid age public key (exported for init command)
func IsValidPublicKey(key string) bool {
	return ValidatePublicKey(key) == nil
}

// IsPrivateKey checks if a string looks like an age private key (exported for init command)
func IsPrivateKey(key string) bool {
	key = strings.TrimSpace(key)
	return strings.HasPrefix(key, "AGE-SECRET-KEY-")
}
