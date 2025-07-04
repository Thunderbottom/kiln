package crypto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// AgeManager handles Age encryption/decryption operations
type AgeManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

// NewAgeManager creates a new Age manager with the given public keys
func NewAgeManager(publicKeys []string) (*AgeManager, error) {
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

	return &AgeManager{
		recipients: recipients,
	}, nil
}

// AddIdentity adds a private key identity for decryption
func (am *AgeManager) AddIdentity(privateKey string) error {
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

// Encrypt encrypts data using all configured recipients
func (am *AgeManager) Encrypt(ctx context.Context, data []byte) ([]byte, error) {
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

// Decrypt decrypts data using configured identities
func (am *AgeManager) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
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

// GenerateKeyPair generates a new Age key pair
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", fmt.Errorf("key generation failed: %w", err)
	}

	privateKey = identity.String()
	publicKey = identity.Recipient().String()

	return privateKey, publicKey, nil
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
