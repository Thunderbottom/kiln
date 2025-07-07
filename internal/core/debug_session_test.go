package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
)

func TestMinimalSession(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "kiln-debug-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate key pair
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	defer WipeData(privateKey)

	t.Logf("Generated public key: %s", publicKey)

	// Save private key
	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SavePrivateKey(privateKey, keyPath); err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	t.Logf("Saved private key to: %s", keyPath)

	// Create config
	cfg := config.NewConfig()
	cfg.AddRecipient(publicKey)
	cfg.Files["default"] = filepath.Join(tmpDir, ".kiln.env")

	configPath := filepath.Join(tmpDir, "config.toml")
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	t.Logf("Saved config to: %s", configPath)
	t.Logf("Config recipients: %v", cfg.Recipients)

	// Try to create session
	session, err := NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	t.Logf("Session created successfully")

	// Try a simple operation
	testVars := map[string][]byte{"TEST": []byte("value")}
	if err := session.SaveVars("default", testVars); err != nil {
		t.Fatalf("SaveVars failed: %v", err)
	}

	t.Logf("SaveVars succeeded")

	vars, cleanup, err := session.LoadVars("default")
	if err != nil {
		t.Fatalf("LoadVars failed: %v", err)
	}
	defer cleanup()

	t.Logf("LoadVars succeeded, got %d variables", len(vars))

	if len(vars) != 1 || string(vars["TEST"]) != "value" {
		t.Fatalf("Variable mismatch: expected TEST=value, got %v", vars)
	}

	t.Logf("All operations successful!")
}
