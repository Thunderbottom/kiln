package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestGenerateKeyPair(t *testing.T) {
	tmpDir := createTestDir(t)

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	// Validate formats
	if !IsPrivateKey(string(privateKey)) {
		t.Error("Generated private key has invalid format")
	}

	if ValidatePublicKey(publicKey) != nil {
		t.Error("Generated public key has invalid format")
	}

	// Test that keys work together using file-based approach
	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Verify public key matches
	if identity.PublicKey() != publicKey {
		t.Errorf("Public key mismatch: expected %s, got %s", publicKey, identity.PublicKey())
	}

	// Test encryption/decryption works
	recipients, err := ParseRecipients([]string{publicKey})
	if err != nil {
		t.Fatalf("ParseRecipients failed: %v", err)
	}

	manager := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})

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

func TestSaveLoadKeys(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath := filepath.Join(tmpDir, "test.key")

	originalPrivateKey, originalPublicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	defer WipeData(originalPrivateKey)

	// Test saving both keys
	err = SaveKeys(originalPrivateKey, originalPublicKey, keyPath)
	if err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	// Verify private key file exists with correct permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Private key file not created: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("Private key file permissions: expected 0600, got %v", info.Mode().Perm())
	}

	// Verify public key file exists
	pubKeyPath := keyPath + ".pub"
	if !FileExists(pubKeyPath) {
		t.Error("Public key file not created")
	}

	// Test loading private key
	loadedKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}
	defer WipeData(loadedKey)

	// Compare keys (strip whitespace)
	originalTrimmed := bytes.TrimSpace(originalPrivateKey)
	loadedTrimmed := bytes.TrimSpace(loadedKey)

	if !bytes.Equal(originalTrimmed, loadedTrimmed) {
		t.Error("Loaded key doesn't match saved key")
	}

	// Test saving only private key
	privateOnlyPath := filepath.Join(tmpDir, "private_only.key")

	err = SaveKeys(originalPrivateKey, "", privateOnlyPath)
	if err != nil {
		t.Fatalf("SaveKeys with private key only failed: %v", err)
	}

	if !FileExists(privateOnlyPath) {
		t.Error("Private key file not created when saving private key only")
	}

	if FileExists(privateOnlyPath + ".pub") {
		t.Error("Public key file should not be created when public key is empty")
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
			name: "public key from file with newline",
			setup: func() string {
				pubFile := filepath.Join(tmpDir, "public_with_newline.key")
				os.WriteFile(pubFile, []byte(publicKey+"\n"), 0o644)

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
			name:    "invalid key string",
			setup:   func() string { return "invalid" },
			wantErr: true,
		},
		{
			name: "file with invalid content",
			setup: func() string {
				invalidFile := filepath.Join(tmpDir, "invalid.key")
				os.WriteFile(invalidFile, []byte("invalid key content"), 0o644)

				return invalidFile
			},
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

func TestGetPrivateKeyCandidates(t *testing.T) {
	// Test that function returns expected candidate paths
	candidates := GetPrivateKeyCandidates()

	if len(candidates) == 0 {
		t.Error("Expected at least one candidate key path")
	}

	// Should include standard paths
	expectedPaths := []string{
		".kiln/kiln.key",
		".ssh/id_ed25519",
		".ssh/id_rsa",
	}

	foundExpected := 0

	for _, candidate := range candidates {
		for _, expected := range expectedPaths {
			if strings.Contains(candidate, expected) {
				foundExpected++

				break
			}
		}
	}

	if foundExpected == 0 {
		t.Error("No expected standard paths found in candidates")
	}
}
