package core

import (
	"bytes"
	"path/filepath"
	"testing"

	"filippo.io/age"
)

func TestParseRecipients(t *testing.T) {
	_, publicKey := generateTestKeyPair(t)

	tests := []struct {
		name        string
		keys        []string
		expectError bool
	}{
		{
			name: "valid age key",
			keys: []string{publicKey},
		},
		{
			name: "multiple age keys",
			keys: []string{publicKey, publicKey}, // Use same key twice for simplicity
		},
		{
			name:        "empty keys",
			keys:        []string{},
			expectError: true,
		},
		{
			name:        "invalid key",
			keys:        []string{"invalid"},
			expectError: true,
		},
		{
			name:        "private key instead of public",
			keys:        []string{"AGE-SECRET-KEY-1234567890"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipients, err := ParseRecipients(tt.keys)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if len(recipients) != len(tt.keys) {
				t.Errorf("Expected %d recipients, got %d", len(tt.keys), len(recipients))
			}
		})
	}
}

func TestNewAgeManager(t *testing.T) {
	tmpDir := createTestDir(t)

	privateKey, publicKey := generateTestKeyPair(t)
	defer WipeData(privateKey)

	// Save key to file and create identity
	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	recipients, err := ParseRecipients([]string{publicKey})
	if err != nil {
		t.Fatalf("Failed to parse recipients: %v", err)
	}

	// Valid manager
	manager := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})
	if manager == nil {
		t.Error("Manager is nil")
	}

	// Empty recipients
	emptyManager := NewAgeManager([]age.Recipient{}, []age.Identity{identity.AgeIdentity()})
	if emptyManager == nil {
		t.Error("Manager should be created even with empty recipients")
	}
}

func TestAgeManagerEncryptDecrypt(t *testing.T) {
	tmpDir := createTestDir(t)

	// Generate key pair and save to file
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	recipients, err := ParseRecipients([]string{publicKey})
	if err != nil {
		t.Fatalf("ParseRecipients failed: %v", err)
	}

	manager := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})

	plaintext := []byte("Hello, World!")

	// Test encryption
	encrypted, err := manager.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Encrypted data should be different
	if bytes.Equal(encrypted, plaintext) {
		t.Error("Encrypted data is same as plaintext")
	}

	// Test decryption
	decrypted, err := manager.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}
	defer WipeData(decrypted)

	// Decrypted should match original
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted data doesn't match: expected %q, got %q", plaintext, decrypted)
	}

	// Empty data should fail
	_, err = manager.Encrypt([]byte(""))
	if err == nil {
		t.Error("Expected error with empty data")
	}
}

func TestAgeManagerMultipleRecipients(t *testing.T) {
	tmpDir := createTestDir(t)

	// Generate first key pair
	privateKey1, publicKey1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair 1 failed: %v", err)
	}
	defer WipeData(privateKey1)

	keyPath1 := filepath.Join(tmpDir, "key1.key")
	if err := SaveKeys(privateKey1, publicKey1, keyPath1); err != nil {
		t.Fatalf("SaveKeys 1 failed: %v", err)
	}

	// Generate second key pair
	privateKey2, publicKey2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair 2 failed: %v", err)
	}
	defer WipeData(privateKey2)

	keyPath2 := filepath.Join(tmpDir, "key2.key")
	if err := SaveKeys(privateKey2, publicKey2, keyPath2); err != nil {
		t.Fatalf("SaveKeys 2 failed: %v", err)
	}

	// Create identities
	identity1, err := NewIdentityFromKey(keyPath1)
	if err != nil {
		t.Fatalf("NewIdentityFromKey 1 failed: %v", err)
	}

	identity2, err := NewIdentityFromKey(keyPath2)
	if err != nil {
		t.Fatalf("NewIdentityFromKey 2 failed: %v", err)
	}

	// Create manager with multiple recipients
	recipients, err := ParseRecipients([]string{publicKey1, publicKey2})
	if err != nil {
		t.Fatalf("ParseRecipients failed: %v", err)
	}

	manager := NewAgeManager(recipients, []age.Identity{identity1.AgeIdentity()})

	plaintext := []byte("Multi-recipient test")

	encrypted, err := manager.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// First key should be able to decrypt
	manager1 := NewAgeManager(recipients, []age.Identity{identity1.AgeIdentity()})

	decrypted1, err := manager1.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decryption with key 1 failed: %v", err)
	} else {
		defer WipeData(decrypted1)

		if !bytes.Equal(decrypted1, plaintext) {
			t.Error("Key 1 decryption mismatch")
		}
	}

	// Second key should also be able to decrypt
	manager2 := NewAgeManager(recipients, []age.Identity{identity2.AgeIdentity()})

	decrypted2, err := manager2.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decryption with key 2 failed: %v", err)
	} else {
		defer WipeData(decrypted2)

		if !bytes.Equal(decrypted2, plaintext) {
			t.Error("Key 2 decryption mismatch")
		}
	}
}

func TestValidatePublicKey(t *testing.T) {
	_, validKey := generateTestKeyPair(t)

	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{"valid age key", validKey, false},
		{"empty key", "", true},
		{"invalid key", "invalid", true},
		{"private key instead of public", "AGE-SECRET-KEY-1234567890", true},
		{"malformed age key", "age1invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicKey(tt.key)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePublicKey(%s): expected error=%v, got=%v", tt.key, tt.expectError, err != nil)
			}
		})
	}
}

func TestIsPrivateKey(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	defer WipeData(privateKey)

	if !IsPrivateKey(string(privateKey)) {
		t.Error("Valid private key not recognized")
	}

	if IsPrivateKey(publicKey) {
		t.Error("Public key incorrectly identified as private")
	}

	if IsPrivateKey("") {
		t.Error("Empty string incorrectly identified as private key")
	}

	// Test SSH private key format
	if !IsPrivateKey("-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("SSH private key format not recognized")
	}
}

// Helper function for age tests
func generateTestKeyPair(t *testing.T) (privateKey []byte, publicKey string) {
	t.Helper()

	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	return privKey, pubKey
}
