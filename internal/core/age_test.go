package core

import (
	"bytes"
	"testing"
)

func TestNewAgeManager(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	defer WipeData(privateKey)

	// Valid manager
	manager, err := NewAgeManager([]string{publicKey}, privateKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Error("Manager is nil")
	}

	// Empty recipients should fail
	_, err = NewAgeManager([]string{}, privateKey)
	if err == nil {
		t.Error("Expected error with empty recipients")
	}

	// Invalid private key should fail
	_, err = NewAgeManager([]string{publicKey}, []byte("invalid"))
	if err == nil {
		t.Error("Expected error with invalid private key")
	}
}

func TestAgeManagerEncryptDecrypt(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	defer WipeData(privateKey)

	manager, err := NewAgeManager([]string{publicKey}, privateKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

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
	privateKey1, publicKey1 := generateTestKeyPair(t)
	defer WipeData(privateKey1)

	privateKey2, publicKey2 := generateTestKeyPair(t)
	defer WipeData(privateKey2)

	// Create manager with multiple recipients
	manager, err := NewAgeManager([]string{publicKey1, publicKey2}, privateKey1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	plaintext := []byte("Multi-recipient test")

	encrypted, err := manager.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Both keys should be able to decrypt
	manager1, _ := NewAgeManager([]string{publicKey1}, privateKey1)

	decrypted1, err := manager1.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decryption with key 1 failed: %v", err)
	} else {
		defer WipeData(decrypted1)

		if !bytes.Equal(decrypted1, plaintext) {
			t.Error("Key 1 decryption mismatch")
		}
	}

	manager2, _ := NewAgeManager([]string{publicKey2}, privateKey2)

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
		{"valid key", validKey, false},
		{"empty key", "", true},
		{"invalid key", "invalid", true},
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
}

// Helper function for core tests
func generateTestKeyPair(t *testing.T) (privateKey []byte, publicKey string) {
	t.Helper()

	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	return privKey, pubKey
}
