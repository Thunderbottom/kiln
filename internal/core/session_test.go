package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
)

func TestNewSession(t *testing.T) {
	configPath, keyPath := setupTestSession(t)

	session, err := NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("Session is nil")
	}

	cfg := session.Config()
	if cfg == nil {
		t.Fatal("Config is nil")
	}

	if len(cfg.Recipients) == 0 {
		t.Error("No recipients in config")
	}
}

func TestNewSessionErrors(t *testing.T) {
	tmpDir := setupTempDir(t)

	// Non-existent config
	_, err := NewSession(filepath.Join(tmpDir, "nonexistent.toml"), "dummy.key")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}

	// Non-existent key
	cfg := config.NewConfig()
	cfg.AddRecipient("age1234567890")

	configPath := filepath.Join(tmpDir, "config.toml")
	cfg.Save(configPath)

	_, err = NewSession(configPath, filepath.Join(tmpDir, "nonexistent.key"))
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestSessionSaveLoadVars(t *testing.T) {
	configPath, keyPath := setupTestSession(t)

	session, err := NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost/test"),
		"API_KEY":      []byte("secret-123"),
		"PORT":         []byte("8080"),
	}

	// Save variables
	err = session.SaveVars("default", testVars)
	if err != nil {
		t.Fatalf("SaveVars failed: %v", err)
	}

	// Load variables
	vars, cleanup, err := session.LoadVars("default")
	if err != nil {
		t.Fatalf("LoadVars failed: %v", err)
	}
	defer cleanup()

	// Verify variables
	if len(vars) != len(testVars) {
		t.Errorf("Variable count mismatch: expected %d, got %d", len(testVars), len(vars))
	}

	for key, expectedValue := range testVars {
		actualValue, exists := vars[key]
		if !exists {
			t.Errorf("Missing variable %s", key)

			continue
		}

		if string(actualValue) != string(expectedValue) {
			t.Errorf("Variable %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

func TestSessionSetGetVar(t *testing.T) {
	configPath, keyPath := setupTestSession(t)

	session, err := NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	key := "TEST_VAR"
	value := []byte("test value")

	// Set variable
	err = session.SetVar("default", key, value)
	if err != nil {
		t.Fatalf("SetVar failed: %v", err)
	}

	// Get variable
	retrieved, cleanup, err := session.GetVar("default", key)
	if err != nil {
		t.Fatalf("GetVar failed: %v", err)
	}
	defer cleanup()

	if string(retrieved) != string(value) {
		t.Errorf("Retrieved value mismatch: expected %q, got %q", value, retrieved)
	}
}

func TestSessionExportVars(t *testing.T) {
	configPath, keyPath := setupTestSession(t)

	session, err := NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	testVars := map[string][]byte{
		"BASE":   []byte("/app"),
		"CONFIG": []byte("${BASE}/config"),
	}

	err = session.SaveVars("default", testVars)
	if err != nil {
		t.Fatalf("SaveVars failed: %v", err)
	}

	// Export without expansion
	vars, cleanup, err := session.ExportVars("default", false)
	if err != nil {
		t.Fatalf("ExportVars failed: %v", err)
	}
	defer cleanup()

	if string(vars["CONFIG"]) != "${BASE}/config" {
		t.Error("Variable expansion occurred when it shouldn't")
	}
}

func TestSortedKeys(t *testing.T) {
	vars := map[string][]byte{
		"ZEBRA": []byte("z"),
		"ALPHA": []byte("a"),
	}

	keys := SortedKeys(vars)
	expected := []string{"ALPHA", "ZEBRA"}

	if len(keys) != len(expected) {
		t.Errorf("Length mismatch: expected %d, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Key at position %d: expected %q, got %q", i, expected[i], key)
		}
	}
}

// Helper functions - completely clean without any verification
func setupTestSession(t *testing.T) (configPath, keyPath string) {
	t.Helper()
	tmpDir := setupTempDir(t)

	// Generate key pair
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Setup paths
	configPath = filepath.Join(tmpDir, "kiln.toml")
	keyPath = filepath.Join(tmpDir, "kiln.key")

	// Save private key
	if err := SavePrivateKey(privateKey, keyPath); err != nil {
		WipeData(privateKey)
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Create config with absolute path for env file
	cfg := config.NewConfig()
	cfg.AddRecipient(publicKey)

	cfg.Files["default"] = filepath.Join(tmpDir, ".kiln.env")
	if err := cfg.Save(configPath); err != nil {
		WipeData(privateKey)
		t.Fatalf("Failed to save config: %v", err)
	}

	// Clean up the in-memory private key
	WipeData(privateKey)

	return configPath, keyPath
}

func setupTempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}
