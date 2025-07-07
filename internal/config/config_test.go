package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	prodEnv = ".kiln.prod.env"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}

	if len(cfg.Recipients) != 0 {
		t.Errorf("Expected empty recipients, got %d", len(cfg.Recipients))
	}

	if len(cfg.Files) != 1 {
		t.Errorf("Expected 1 default file, got %d", len(cfg.Files))
	}

	if cfg.Files["default"] != DefaultEnvFile {
		t.Errorf("Default file incorrect: expected %q, got %q", DefaultEnvFile, cfg.Files["default"])
	}
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := createTempDir(t)
	configPath := filepath.Join(tmpDir, "test.toml")

	// Create test config
	cfg := NewConfig()
	cfg.AddRecipient("age1234567890abcdef")
	cfg.Files["production"] = prodEnv

	// Save config
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify
	if !reflect.DeepEqual(loaded.Recipients, cfg.Recipients) {
		t.Errorf("Recipients mismatch: expected %v, got %v", cfg.Recipients, loaded.Recipients)
	}

	if !reflect.DeepEqual(loaded.Files, cfg.Files) {
		t.Errorf("Files mismatch: expected %v, got %v", cfg.Files, loaded.Files)
	}
}

func TestConfigValidate(t *testing.T) {
	// Valid config
	cfg := NewConfig()
	cfg.AddRecipient("age1234567890")

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// No recipients
	cfg.Recipients = []string{}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation error for no recipients")
	}
}

func TestConfigAddRemoveRecipient(t *testing.T) {
	cfg := NewConfig()

	// Add recipients
	cfg.AddRecipient("age1111111111")
	cfg.AddRecipient("age2222222222")

	if len(cfg.Recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(cfg.Recipients))
	}

	// Add duplicate (should not be added)
	cfg.AddRecipient("age1111111111")

	if len(cfg.Recipients) != 2 {
		t.Errorf("Expected 2 recipients after duplicate, got %d", len(cfg.Recipients))
	}

	// Remove recipient
	removed := cfg.RemoveRecipient("age1111111111")
	if !removed {
		t.Error("RemoveRecipient should return true")
	}

	if len(cfg.Recipients) != 1 {
		t.Errorf("Expected 1 recipient after removal, got %d", len(cfg.Recipients))
	}

	// Remove non-existent
	removed = cfg.RemoveRecipient("nonexistent")
	if removed {
		t.Error("RemoveRecipient should return false for non-existent")
	}
}

func TestGetEnvFile(t *testing.T) {
	cfg := NewConfig()
	cfg.Files["production"] = prodEnv

	// Default file
	file, err := cfg.GetEnvFile("")
	if err != nil {
		t.Fatalf("GetEnvFile failed: %v", err)
	}

	if file != DefaultEnvFile {
		t.Errorf("Expected %q, got %q", DefaultEnvFile, file)
	}

	// Specific file
	file, err = cfg.GetEnvFile("production")
	if err != nil {
		t.Fatalf("GetEnvFile failed: %v", err)
	}

	if file != prodEnv {
		t.Errorf("Expected .kiln.prod.env, got %q", file)
	}

	// Non-existent file
	_, err = cfg.GetEnvFile("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Helper functions
func createTempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}
