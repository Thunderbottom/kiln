package core

import (
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
)

func TestGetAllEnvVars(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath, cfg := setupTestConfig(t, tmpDir)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Test with non-existent file (should return empty map)
	vars, cleanup, err := GetAllEnvVars(identity, cfg, "default")
	if err != nil {
		t.Fatalf("GetAllEnvVars failed for non-existent file: %v", err)
	}

	cleanup()

	if len(vars) != 0 {
		t.Errorf("Expected empty map for non-existent file, got %d variables", len(vars))
	}

	// Test with actual variables
	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost/test"),
		"API_KEY":      []byte("secret-123"),
	}

	err = SaveAllEnvVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	vars, cleanup, err = GetAllEnvVars(identity, cfg, "default")
	if err != nil {
		t.Fatalf("GetAllEnvVars failed: %v", err)
	}
	defer cleanup()

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

func TestSaveAllEnvVars(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath, cfg := setupTestConfig(t, tmpDir)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	testVars := map[string][]byte{
		"TEST_VAR": []byte("test_value"),
	}

	err = SaveAllEnvVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	// Verify file was created
	filePath, _ := cfg.GetEnvFile("default")
	if !FileExists(filePath) {
		t.Error("Environment file was not created")
	}
}

func TestGetEnvVar(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath, cfg := setupTestConfig(t, tmpDir)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Setup test data
	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost/test"),
		"API_KEY":      []byte("secret-123"),
	}

	err = SaveAllEnvVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	// Test getting existing variable
	value, cleanup, err := GetEnvVar(identity, cfg, "default", "API_KEY")
	if err != nil {
		t.Fatalf("GetEnvVar failed: %v", err)
	}
	defer cleanup()

	if string(value) != "secret-123" {
		t.Errorf("GetEnvVar: expected 'secret-123', got %q", string(value))
	}

	// Test getting non-existent variable
	_, _, err = GetEnvVar(identity, cfg, "default", "NONEXISTENT")
	if err == nil {
		t.Error("GetEnvVar should fail for non-existent variable")
	}
}

func TestSetEnvVar(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath, cfg := setupTestConfig(t, tmpDir)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Set a variable
	err = SetEnvVar(identity, cfg, "default", "NEW_VAR", []byte("new_value"))
	if err != nil {
		t.Fatalf("SetEnvVar failed: %v", err)
	}

	// Verify it was set
	value, cleanup, err := GetEnvVar(identity, cfg, "default", "NEW_VAR")
	if err != nil {
		t.Fatalf("GetEnvVar failed after SetEnvVar: %v", err)
	}
	defer cleanup()

	if string(value) != "new_value" {
		t.Errorf("SetEnvVar: expected 'new_value', got %q", string(value))
	}

	// Update existing variable
	err = SetEnvVar(identity, cfg, "default", "NEW_VAR", []byte("updated_value"))
	if err != nil {
		t.Fatalf("SetEnvVar update failed: %v", err)
	}

	// Verify update
	updatedValue, updatedCleanup, err := GetEnvVar(identity, cfg, "default", "NEW_VAR")
	if err != nil {
		t.Fatalf("GetEnvVar failed after update: %v", err)
	}
	defer updatedCleanup()

	if string(updatedValue) != "updated_value" {
		t.Errorf("SetEnvVar update: expected 'updated_value', got %q", string(updatedValue))
	}
}

func TestCheckEnvFile(t *testing.T) {
	tmpDir := createTestDir(t)
	keyPath, cfg := setupTestConfig(t, tmpDir)

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	// Check non-existent file (should not error)
	err = CheckEnvFile(identity, cfg, "default")
	if err != nil {
		t.Errorf("CheckEnvFile failed for non-existent file: %v", err)
	}

	// Create file and check
	testVars := map[string][]byte{
		"TEST": []byte("value"),
	}

	err = SaveAllEnvVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	err = CheckEnvFile(identity, cfg, "default")
	if err != nil {
		t.Errorf("CheckEnvFile failed for valid file: %v", err)
	}

	// Test with unconfigured file
	err = CheckEnvFile(identity, cfg, "unconfigured")
	if err == nil {
		t.Error("CheckEnvFile should fail for unconfigured file")
	}
}

// Helper function to setup test configuration
func setupTestConfig(t *testing.T, tmpDir string) (keyPath string, cfg *config.Config) {
	t.Helper()

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	defer WipeData(privateKey)

	// Save key to file
	keyPath = filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	cfg = config.NewConfig()
	cfg.AddRecipient("test-user", publicKey)
	cfg.Files["default"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, ".kiln.env"),
		Access:   []string{"*"},
	}

	return keyPath, cfg
}
