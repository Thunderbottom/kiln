package core

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/alecthomas/kong"
)

// AgeManager handles all Age encryption/decryption operations
type AgeManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

// NewAgeManager creates manager with pre-parsed recipients and identities
func NewAgeManager(recipients []age.Recipient, identities []age.Identity) *AgeManager {
	return &AgeManager{
		recipients: recipients,
		identities: identities,
	}
}

// Encrypt encrypts data using all configured recipients
func (am *AgeManager) Encrypt(data []byte) ([]byte, error) {
	if len(am.recipients) == 0 {
		return nil, fmt.Errorf("no recipients configured")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data to encrypt")
	}

	estimatedSize := len(data) + 200 + (len(am.recipients) * 50)

	var buf bytes.Buffer

	buf.Grow(estimatedSize)

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

	result := make([]byte, 0, len(data))

	// Use a fixed buffer for reading to avoid many small allocations
	buf := make([]byte, 32*1024) // 32KB read buffer

	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decrypt: %w", err)
		}
	}

	return result, nil
}

// ParseRecipients converts public key strings into age.Recipient objects
func ParseRecipients(publicKeys []string) ([]age.Recipient, error) {
	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("no public keys provided")
	}

	// Pre-allocate slice with exact capacity to avoid reallocations
	recipients := make([]age.Recipient, 0, len(publicKeys))

	for _, key := range publicKeys {
		// Avoid string allocation by checking prefix directly
		if len(key) == 0 {
			continue
		}

		// Trim whitespace in place without allocation
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		var recipient age.Recipient

		var err error

		if strings.HasPrefix(key, "age1") {
			recipient, err = age.ParseX25519Recipient(key)
		} else {
			recipient, err = agessh.ParseRecipient(key)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse key %s: %w", key, err)
		}

		recipients = append(recipients, recipient)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no valid public keys found")
	}

	return recipients, nil
}

// ValidatePublicKey validates age or SSH public key format
func ValidatePublicKey(key string) error {
	if len(strings.TrimSpace(key)) == 0 {
		return fmt.Errorf("empty public key")
	}

	// Check for common mistakes
	if strings.HasPrefix(key, "AGE-SECRET-KEY-") {
		return fmt.Errorf("private key provided instead of public key - use the corresponding public key")
	}

	if strings.Contains(key, "PRIVATE KEY") {
		return fmt.Errorf("private key provided instead of public key - use the corresponding public key")
	}

	if strings.HasPrefix(key, "age1") {
		if len(key) < 60 || len(key) > 70 {
			return fmt.Errorf("invalid age public key format")
		}

		return nil
	}

	if len(key) >= 4 && key[:4] == "ssh-" {
		parts := strings.Fields(key)
		if len(parts) < 2 {
			return fmt.Errorf("invalid SSH public key format")
		}

		return nil
	}

	return fmt.Errorf("unsupported key format - must start with 'age1' or 'ssh-'")
}

// IsPrivateKey checks if a string looks like an age private key
func IsPrivateKey(key string) bool {
	key = strings.TrimSpace(key)

	return strings.HasPrefix(key, "AGE-SECRET-KEY-") ||
		strings.Contains(key, "PRIVATE KEY") ||
		strings.Contains(key, "-----BEGIN") ||
		strings.Contains(key, "-----END")
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

	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'name=key', got '%s'", value)
	}

	name := strings.TrimSpace(parts[0])
	keyOrPath := strings.TrimSpace(parts[1])

	if name == "" {
		return fmt.Errorf("recipient name cannot be empty")
	}

	// Validate and resolve the public key (either direct key or from file)
	publicKey := keyOrPath
	if ValidatePublicKey(keyOrPath) != nil {
		// Try loading from file
		var err error
		publicKey, err = LoadPublicKey(keyOrPath)
		if err != nil {
			return fmt.Errorf("invalid public key or file '%s': %w", keyOrPath, err)
		}
	}

	switch target.Kind() {
	case reflect.Map:
		if target.Type().Key().Kind() != reflect.String || target.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("map target must be map[string]string, got %v", target.Type())
		}

		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}

		target.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(publicKey))

		return nil

	default:
		return fmt.Errorf("unsupported target type for age public key mapper: %v", target.Kind())
	}
})
