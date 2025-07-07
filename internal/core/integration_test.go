package core

import (
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
)

func TestFullWorkflowIntegration(t *testing.T) {
	tmpDir := createTestDir(t)

	// Generate key pair and save to files
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	// Setup configuration
	cfg := config.NewConfig()
	cfg.AddRecipient("test-user", publicKey)
	cfg.Files["default"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, ".kiln.env"),
		Access:   []string{"*"},
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Test workflow: Set -> Get -> Update -> Get -> Check
	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost/test"),
		"API_KEY":      []byte("secret-123"),
		"PORT":         []byte("8080"),
	}

	// Save variables
	if err := SaveAllEnvVars(identity, cfg, "default", testVars); err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	// Retrieve all variables
	retrieved, cleanup, err := GetAllEnvVars(identity, cfg, "default")
	if err != nil {
		t.Fatalf("GetAllEnvVars failed: %v", err)
	}
	defer cleanup()

	// Verify all variables match
	if len(retrieved) != len(testVars) {
		t.Errorf("Variable count mismatch: expected %d, got %d", len(testVars), len(retrieved))
	}

	for key, expectedValue := range testVars {
		actualValue, exists := retrieved[key]
		if !exists {
			t.Errorf("Missing variable %s", key)

			continue
		}

		if string(actualValue) != string(expectedValue) {
			t.Errorf("Variable %s: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Test single variable operations
	singleValue, singleCleanup, err := GetEnvVar(identity, cfg, "default", "API_KEY")
	if err != nil {
		t.Fatalf("GetEnvVar failed: %v", err)
	}
	defer singleCleanup()

	if string(singleValue) != "secret-123" {
		t.Errorf("Single variable retrieval: expected 'secret-123', got %q", singleValue)
	}

	// Test variable update
	if err := SetEnvVar(identity, cfg, "default", "API_KEY", []byte("updated-secret")); err != nil {
		t.Fatalf("SetEnvVar failed: %v", err)
	}

	// Verify update
	updatedValue, updatedCleanup, err := GetEnvVar(identity, cfg, "default", "API_KEY")
	if err != nil {
		t.Fatalf("GetEnvVar after update failed: %v", err)
	}
	defer updatedCleanup()

	if string(updatedValue) != "updated-secret" {
		t.Errorf("Updated variable: expected 'updated-secret', got %q", updatedValue)
	}

	// Test file verification
	if err := CheckEnvFile(identity, cfg, "default"); err != nil {
		t.Errorf("CheckEnvFile failed: %v", err)
	}
}

func TestMultipleFileHandling(t *testing.T) {
	tmpDir := createTestDir(t)

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	// Setup configuration with multiple files
	cfg := config.NewConfig()
	cfg.AddRecipient("user1", publicKey)
	cfg.Files["dev"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, "dev.env"),
		Access:   []string{"*"},
	}
	cfg.Files["prod"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, "prod.env"),
		Access:   []string{"*"},
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Set different variables in different files
	devVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost/dev"),
		"DEBUG":        []byte("true"),
	}

	prodVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://prod-server/db"),
		"DEBUG":        []byte("false"),
	}

	if err := SaveAllEnvVars(identity, cfg, "dev", devVars); err != nil {
		t.Fatalf("SaveAllEnvVars dev failed: %v", err)
	}

	if err := SaveAllEnvVars(identity, cfg, "prod", prodVars); err != nil {
		t.Fatalf("SaveAllEnvVars prod failed: %v", err)
	}

	// Verify dev file
	devRetrieved, devCleanup, err := GetAllEnvVars(identity, cfg, "dev")
	if err != nil {
		t.Fatalf("GetAllEnvVars dev failed: %v", err)
	}
	defer devCleanup()

	if string(devRetrieved["DEBUG"]) != "true" {
		t.Errorf("Dev DEBUG: expected 'true', got %q", devRetrieved["DEBUG"])
	}

	// Verify prod file
	prodRetrieved, prodCleanup, err := GetAllEnvVars(identity, cfg, "prod")
	if err != nil {
		t.Fatalf("GetAllEnvVars prod failed: %v", err)
	}
	defer prodCleanup()

	if string(prodRetrieved["DEBUG"]) != "false" {
		t.Errorf("Prod DEBUG: expected 'false', got %q", prodRetrieved["DEBUG"])
	}
}

func TestErrorScenarios(t *testing.T) {
	tmpDir := createTestDir(t)

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	cfg := config.NewConfig()
	cfg.AddRecipient("test-user", publicKey)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Test accessing unconfigured file
	_, _, err = GetAllEnvVars(identity, cfg, "unconfigured")
	if err == nil {
		t.Error("GetAllEnvVars should error for unconfigured file")
	}

	// Test getting non-existent variable from non-existent file
	_, _, err = GetEnvVar(identity, cfg, "unconfigured", "SOME_VAR")
	if err == nil {
		t.Error("GetEnvVar should error for unconfigured file")
	}
}
