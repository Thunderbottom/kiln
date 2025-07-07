package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	// Validate formats
	if !IsPrivateKey(string(privateKey)) {
		t.Error("Generated private key has invalid format")
	}

	if !IsValidPublicKey(publicKey) {
		t.Error("Generated public key has invalid format")
	}

	// Test that keys work together
	manager, err := NewAgeManager([]string{publicKey}, privateKey)
	if err != nil {
		t.Fatalf("Keys don't work together: %v", err)
	}

	testData := []byte("test")

	encrypted, err := manager.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := manager.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}
	defer WipeData(decrypted)

	if !bytes.Equal(testData, decrypted) {
		t.Error("Key pair doesn't work correctly")
	}
}

func TestSaveLoadPrivateKey(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath := filepath.Join(tmpDir, "test.key")

	originalKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	defer WipeData(originalKey)

	// Test saving
	err = SavePrivateKey(originalKey, keyPath)
	if err != nil {
		t.Fatalf("SavePrivateKey failed: %v", err)
	}

	// Verify file exists with correct permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Key file not created: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("Key file permissions: expected 0600, got %v", info.Mode().Perm())
	}

	// Test loading
	loadedKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}
	defer WipeData(loadedKey)

	// Compare keys (strip whitespace)
	originalTrimmed := bytes.TrimSpace(originalKey)
	loadedTrimmed := bytes.TrimSpace(loadedKey)

	if !bytes.Equal(originalTrimmed, loadedTrimmed) {
		t.Error("Loaded key doesn't match saved key")
	}
}

func TestLoadPrivateKeyErrors(t *testing.T) {
	tmpDir := createTestDir(t)

	// Non-existent file
	_, err := LoadPrivateKey(filepath.Join(tmpDir, "nonexistent.key"))
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Empty file
	emptyFile := filepath.Join(tmpDir, "empty.key")
	os.WriteFile(emptyFile, []byte(""), 0o600)

	_, err = LoadPrivateKey(emptyFile)
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

func TestLoadPublicKey(t *testing.T) {
	tmpDir := createTestDir(t)

	privateKey, publicKey, _ := GenerateKeyPair()
	defer WipeData(privateKey)

	tests := []struct {
		name     string
		setup    func() string
		expected string
		wantErr  bool
	}{
		{
			name:     "direct public key",
			setup:    func() string { return publicKey },
			expected: publicKey,
		},
		{
			name: "public key from file",
			setup: func() string {
				pubFile := filepath.Join(tmpDir, "public.key")
				os.WriteFile(pubFile, []byte(publicKey), 0o644)

				return pubFile
			},
			expected: publicKey,
		},
		{
			name: "private key from file",
			setup: func() string {
				privFile := filepath.Join(tmpDir, "private.key")
				os.WriteFile(privFile, privateKey, 0o600)

				return privFile
			},
			expected: publicKey,
		},
		{
			name:    "invalid key",
			setup:   func() string { return "invalid" },
			wantErr: true,
		},
		{
			name: "non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.key")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			result, err := LoadPublicKey(input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
