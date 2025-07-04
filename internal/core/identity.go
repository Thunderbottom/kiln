package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Identity wraps age.Identity with concrete type safety and enhanced functionality
type Identity struct {
	ageIdentity age.Identity
	publicKey   string
	keyType     string
}

// NewIdentityFromKey creates an identity from a private key file path
func NewIdentityFromKey(keyPath string) (*Identity, error) {
	privateKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	defer WipeData(privateKey)

	keyContent := strings.TrimSpace(string(privateKey))

	// Try age key first
	if strings.HasPrefix(keyContent, "AGE-SECRET-KEY-") {
		return newAgeIdentity(keyContent)
	}

	// Try SSH key
	if isSSHKey(keyContent) {
		return newSSHIdentity(keyPath, privateKey)
	}

	return nil, fmt.Errorf("unsupported key format")
}

// AgeIdentity returns the underlying age.Identity interface required by the age library.
//
//nolint:ireturn
func (i *Identity) AgeIdentity() age.Identity {
	return i.ageIdentity
}

// PublicKey returns the public key string
func (i *Identity) PublicKey() string {
	return i.publicKey
}

// KeyType returns a human-readable key type
func (i *Identity) KeyType() string {
	return i.keyType
}

// Cleanup securely wipes sensitive data if needed
func (i *Identity) Cleanup() {
	if wrapper, ok := i.ageIdentity.(*encryptedSSHIdentityWrapper); ok {
		wrapper.Cleanup()
	}
}

// newAgeIdentity creates identity from age private key
func newAgeIdentity(keyContent string) (*Identity, error) {
	identity, err := age.ParseX25519Identity(keyContent)
	if err != nil {
		return nil, fmt.Errorf("parse age identity: %w", err)
	}

	return &Identity{
		ageIdentity: identity,
		publicKey:   identity.Recipient().String(),
		keyType:     "age",
	}, nil
}

// newSSHIdentity creates identity from SSH private key
func newSSHIdentity(keyPath string, privateKey []byte) (*Identity, error) {
	// Try unencrypted SSH key first
	identity, err := agessh.ParseIdentity(privateKey)
	if err == nil {
		publicKey, pubErr := loadSSHPublicKey(keyPath)
		if pubErr != nil {
			return nil, fmt.Errorf("load SSH public key: %w", pubErr)
		}

		return &Identity{
			ageIdentity: identity,
			publicKey:   publicKey,
			keyType:     "ssh",
		}, nil
	}

	// Check if it's an encrypted SSH key
	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		wrapper := &encryptedSSHIdentityWrapper{
			keyData: append([]byte(nil), privateKey...),
			pubKey:  passphraseErr.PublicKey,
		}

		publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(passphraseErr.PublicKey)))

		return &Identity{
			ageIdentity: wrapper,
			publicKey:   publicKey,
			keyType:     "encrypted-ssh",
		}, nil
	}

	return nil, fmt.Errorf("parse SSH identity: %w", err)
}

// loadSSHPublicKey loads public key from corresponding .pub file
func loadSSHPublicKey(privateKeyPath string) (string, error) {
	pubKeyPath := privateKeyPath + ".pub"

	if !FileExists(pubKeyPath) {
		return "", fmt.Errorf("SSH public key file not found: %s", pubKeyPath)
	}

	pubKeyData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("read public key file: %w", err)
	}

	return strings.TrimSpace(string(pubKeyData)), nil
}

// isSSHKey checks if content appears to be an SSH private key
func isSSHKey(content string) bool {
	return strings.Contains(content, "-----BEGIN") &&
		(strings.Contains(content, "PRIVATE KEY-----") ||
			strings.Contains(content, "OPENSSH PRIVATE KEY-----"))
}

// encryptedSSHIdentityWrapper handles encrypted SSH keys with deferred decryption
type encryptedSSHIdentityWrapper struct {
	keyData  []byte
	pubKey   ssh.PublicKey
	identity age.Identity
}

// Unwrap implements age.Identity interface for encrypted SSH keys
func (w *encryptedSSHIdentityWrapper) Unwrap(stanzas []*age.Stanza) ([]byte, error) {
	if w.identity == nil {
		passphraseFunc := func() ([]byte, error) {
			fmt.Print("Enter passphrase for SSH private key: ")

			passphrase, err := term.ReadPassword(int(syscall.Stdin))

			fmt.Println()

			return passphrase, err
		}

		identity, err := agessh.NewEncryptedSSHIdentity(w.pubKey, w.keyData, passphraseFunc)
		if err != nil {
			return nil, fmt.Errorf("create encrypted SSH identity: %w", err)
		}

		w.identity = identity
	}

	result, err := w.identity.Unwrap(stanzas)
	if err != nil {
		return nil, fmt.Errorf("unwrap SSH identity: %w", err)
	}

	return result, nil
}

// Cleanup wipes sensitive key data from memory
func (w *encryptedSSHIdentityWrapper) Cleanup() {
	if w.keyData != nil {
		WipeData(w.keyData)
		w.keyData = nil
	}

	w.identity = nil
}
