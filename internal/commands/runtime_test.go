package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/core"
)

func TestRuntimeLifecycle(t *testing.T) {
	tmpDir := createTempDir(t)

	// Setup test files
	configPath := filepath.Join(tmpDir, "kiln.toml")
	keyPath := filepath.Join(tmpDir, "kiln.key")

	// Generate and save key
	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer core.WipeData(privateKey)

	err = core.SaveKeys(privateKey, publicKey, keyPath)
	if err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	// Create minimal config
	configContent := fmt.Sprintf(`[recipients]
test-user = "%s"

[files.default]
filename = "%s"
access = ["*"]
`, publicKey, filepath.Join(tmpDir, ".kiln.env"))

	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Test Runtime creation and lifecycle
	runtime, err := NewRuntime(configPath, keyPath, false)
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer runtime.Cleanup()

	// Test lazy loading
	cfg, err := runtime.Config()
	if err != nil {
		t.Fatalf("Config loading failed: %v", err)
	}

	if len(cfg.Recipients) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(cfg.Recipients))
	}

	identity, err := runtime.Identity()
	if err != nil {
		t.Fatalf("Identity loading failed: %v", err)
	}

	if identity == nil {
		t.Error("Identity should not be nil")
	}

	// Test cleanup
	runtime.Cleanup()
}

func TestCommandValidation(t *testing.T) {
	// Test various command validation scenarios
	setCmd := &SetCmd{
		Name: "invalid-name",
		File: "default",
	}

	err := setCmd.validate()
	if err == nil {
		t.Error("SetCmd should fail validation for invalid variable name")
	}

	getCmd := &GetCmd{
		Name: "",
		File: "default",
	}

	err = getCmd.validate()
	if err == nil {
		t.Error("GetCmd should fail validation for empty variable name")
	}
}

func TestSetCmdValidation(t *testing.T) {
	tests := []struct {
		name    string
		cmd     SetCmd
		wantErr bool
	}{
		{"valid name", SetCmd{Name: "VALID_NAME", File: "default"}, false},
		{"empty name", SetCmd{Name: "", File: "default"}, true},
		{"invalid name", SetCmd{Name: "invalid-name", File: "default"}, true},
		{"invalid file", SetCmd{Name: "VALID", File: "../invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
