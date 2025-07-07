package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

func TestNewGlobals(t *testing.T) {
	globals, err := NewGlobals("config.toml", "key.txt", false)
	if err != nil {
		t.Fatalf("NewGlobals failed: %v", err)
	}

	if globals.Config != "config.toml" {
		t.Errorf("Config path incorrect: expected config.toml, got %s", globals.Config)
	}

	if globals.Key != "key.txt" {
		t.Errorf("Key path incorrect: expected key.txt, got %s", globals.Key)
	}

	ctx := globals.Context()
	if ctx == nil {
		t.Error("Context is nil")
	}
}

func TestGlobalsSession(t *testing.T) {
	configPath, keyPath := createTestSession(t)

	globals, err := NewGlobals(configPath, keyPath, false)
	if err != nil {
		t.Fatalf("NewGlobals failed: %v", err)
	}

	// First call should create session
	session1, err := globals.Session()
	if err != nil {
		t.Fatalf("First Session() call failed: %v", err)
	}

	// Second call should return same session
	session2, err := globals.Session()
	if err != nil {
		t.Fatalf("Second Session() call failed: %v", err)
	}

	if session1 != session2 {
		t.Error("Session() should return the same instance")
	}
}

func TestInitKeyCmd(t *testing.T) {
	tmpDir := createTempDir(t)
	keyPath := filepath.Join(tmpDir, "test.key")

	cmd := &InitKeyCmd{
		Path:    keyPath,
		Encrypt: false,
		Force:   false,
	}

	globals, _ := NewGlobals("dummy.toml", "dummy.key", false)

	err := cmd.Run(globals)
	if err != nil {
		t.Fatalf("InitKeyCmd failed: %v", err)
	}

	// Verify key file was created
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("Key file not created: %v", err)
	}

	// Verify it's a valid private key
	keyData, _ := os.ReadFile(keyPath)
	if !core.IsPrivateKey(string(keyData)) {
		t.Error("Generated file does not contain valid private key")
	}
}

func TestInitConfigCmd(t *testing.T) {
	tmpDir := createTempDir(t)
	_, publicKey, _ := core.GenerateKeyPair()
	configPath := filepath.Join(tmpDir, "test.toml")

	cmd := &InitConfigCmd{
		Path:       configPath,
		PublicKeys: []string{publicKey},
		Force:      false,
	}

	globals, _ := NewGlobals("dummy.toml", "dummy.key", false)

	err := cmd.Run(globals)
	if err != nil {
		t.Fatalf("InitConfigCmd failed: %v", err)
	}

	// Verify config file was created and is valid
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}

	if len(cfg.Recipients) != 1 || cfg.Recipients[0] != publicKey {
		t.Error("Config does not contain expected recipient")
	}
}

func TestSetGetCommands(t *testing.T) {
	configPath, keyPath := createTestSession(t)
	globals, _ := NewGlobals(configPath, keyPath, false)

	// Test Set command
	setCmd := &SetCmd{
		Name:  "TEST_VAR",
		Value: "test value",
		File:  "default",
	}

	err := setCmd.Run(globals)
	if err != nil {
		t.Fatalf("SetCmd failed: %v", err)
	}

	// Test Get command
	getCmd := &GetCmd{
		Name:   "TEST_VAR",
		File:   "default",
		Format: "value",
	}

	err = getCmd.Run(globals)
	if err != nil {
		t.Fatalf("GetCmd failed: %v", err)
	}
}

func TestExportCmd(t *testing.T) {
	configPath, keyPath := createTestSession(t)
	globals, _ := NewGlobals(configPath, keyPath, false)

	// Set up test data
	setCmd := &SetCmd{Name: "TEST_EXPORT", Value: "export value", File: "default"}
	setCmd.Run(globals)

	// Test export
	cmd := &ExportCmd{File: "default", Format: "shell", Expand: false}

	err := cmd.Run(globals)
	if err != nil {
		t.Fatalf("ExportCmd failed: %v", err)
	}
}

func TestRunCmd(t *testing.T) {
	configPath, keyPath := createTestSession(t)
	globals, _ := NewGlobals(configPath, keyPath, false)

	// Test dry run
	cmd := &RunCmd{
		File:    "default",
		DryRun:  true,
		Command: []string{"echo", "test"},
	}

	err := cmd.Run(globals)
	if err != nil {
		t.Fatalf("RunCmd dry run failed: %v", err)
	}

	// Test no command error
	cmd.Command = []string{}
	cmd.DryRun = false

	err = cmd.Run(globals)
	if err == nil {
		t.Error("RunCmd should fail with no command")
	}
}

// Helper functions
func createTempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-commands-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

func createTestSession(t *testing.T) (configPath, keyPath string) {
	t.Helper()
	tmpDir := createTempDir(t)

	// Generate a single key pair that we'll use consistently
	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Setup paths
	configPath = filepath.Join(tmpDir, "kiln.toml")
	keyPath = filepath.Join(tmpDir, "kiln.key")

	// Save private key to file
	if err := core.SavePrivateKey(privateKey, keyPath); err != nil {
		core.WipeData(privateKey)
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Create config with matching public key AND absolute path for env file
	cfg := config.NewConfig()
	cfg.AddRecipient(publicKey)
	// FIX: Use absolute path for the env file instead of relative path
	cfg.Files["default"] = filepath.Join(tmpDir, ".kiln.env")
	if err := cfg.Save(configPath); err != nil {
		core.WipeData(privateKey)
		t.Fatalf("Failed to save config: %v", err)
	}

	// Clean up the in-memory private key
	core.WipeData(privateKey)

	// Simple verification - just ensure session can be created
	_, err = core.NewSession(configPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return configPath, keyPath
}
