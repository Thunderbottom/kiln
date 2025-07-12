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

	if cfg.Files["default"].Filename != DefaultEnvFile {
		t.Errorf("Default file incorrect: expected %q, got %q", DefaultEnvFile, cfg.Files["default"].Filename)
	}
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := createTempDir(t)

	cfg := NewConfig()
	cfg.AddRecipient("alice", "age1234...")

	defaultPath := ".kiln.env"
	prodPath := ".kiln.prod.env"

	cfg.Files["default"] = FileConfig{
		Filename: defaultPath,
		Access:   []string{"*"},
	}
	cfg.Files["production"] = FileConfig{
		Filename: prodPath,
		Access:   []string{"alice"},
	}

	configPath := filepath.Join(tmpDir, "kiln.toml")

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedFiles := map[string]FileConfig{
		"default": {
			Filename: filepath.Join(tmpDir, defaultPath),
			Access:   []string{"*"},
		},
		"production": {
			Filename: filepath.Join(tmpDir, prodPath),
			Access:   []string{"alice"},
		},
	}

	if !reflect.DeepEqual(loadedCfg.Files, expectedFiles) {
		t.Errorf("Files mismatch: expected %v, got %v", expectedFiles, loadedCfg.Files)
	}
}

func TestConfigValidate(t *testing.T) {
	// Valid config
	cfg := NewConfig()
	cfg.AddRecipient("alice", "age1234567890")

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// No recipients
	cfg.Recipients = map[string]string{}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected validation error for no recipients")
	}
}

func TestConfigAddRemoveRecipient(t *testing.T) {
	cfg := NewConfig()

	// Add recipients
	cfg.AddRecipient("alice", "age1111111111")
	cfg.AddRecipient("bob", "age2222222222")

	if len(cfg.Recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(cfg.Recipients))
	}

	// Add duplicate (should overwrite)
	cfg.AddRecipient("alice", "age1111111111")

	if len(cfg.Recipients) != 2 {
		t.Errorf("Expected 2 recipients after duplicate, got %d", len(cfg.Recipients))
	}

	// Remove recipient
	removed := cfg.RemoveRecipient("alice")
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
	cfg.Files["production"] = FileConfig{
		Filename: prodEnv,
		Access:   []string{"*"},
	}

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
		t.Errorf("Expected %q, got %q", prodEnv, file)
	}

	// Non-existent file
	_, err = cfg.GetEnvFile("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestResolveFileAccess(t *testing.T) {
	cfg := NewConfig()
	cfg.AddRecipient("alice", "age1111111111")
	cfg.AddRecipient("bob", "age2222222222")
	cfg.Groups["developers"] = []string{"alice", "bob"}

	cfg.Files["team"] = FileConfig{
		Filename: "team.env",
		Access:   []string{"developers"},
	}

	recipients, err := cfg.ResolveFileAccess("team")
	if err != nil {
		t.Fatalf("ResolveFileAccess failed: %v", err)
	}

	expectedCount := 2 // alice and bob from developers group
	if len(recipients) != expectedCount {
		t.Errorf("Expected %d recipients, got %d", expectedCount, len(recipients))
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
