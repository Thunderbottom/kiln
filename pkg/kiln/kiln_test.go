package kiln_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/pkg/kiln"
)

// TestLoadConfig validates configuration loading with various scenarios including path resolution
func TestLoadConfig(t *testing.T) {
	tmpDir := createTestDir(t)

	tests := []struct {
		name        string
		setup       func() string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			setup: func() string {
				configPath := filepath.Join(tmpDir, "valid.toml")
				configContent := `[recipients]
test = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

[files.default]
filename = ".kiln.env"
access = ["*"]
`
				writeFile(t, configPath, configContent)

				return configPath
			},
			expectError: false,
		},
		{
			name: "config with relative paths",
			setup: func() string {
				// Create subdirectory structure for relative path testing
				envDir := filepath.Join(tmpDir, "environments")
				os.MkdirAll(envDir, 0o755)

				configPath := filepath.Join(tmpDir, "relative.toml")
				configContent := `[recipients]
test = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

[files.default]
filename = "./environments/dev.env"
access = ["*"]

[files.production]
filename = "environments/prod.env"
access = ["*"]
`
				writeFile(t, configPath, configContent)

				return configPath
			},
			expectError: false,
		},
		{
			name: "config with absolute paths",
			setup: func() string {
				configPath := filepath.Join(tmpDir, "absolute.toml")
				absEnvPath := filepath.Join(tmpDir, "absolute.env")
				configContent := fmt.Sprintf(`[recipients]
test = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

[files.default]
filename = "%s"
access = ["*"]
`, absEnvPath)
				writeFile(t, configPath, configContent)

				return configPath
			},
			expectError: false,
		},
		{
			name: "empty path",
			setup: func() string {
				return ""
			},
			expectError: true,
			errorMsg:    "config path cannot be empty",
		},
		{
			name: "nonexistent file",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.toml")
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "invalid toml",
			setup: func() string {
				configPath := filepath.Join(tmpDir, "invalid.toml")
				writeFile(t, configPath, "invalid toml content [[[")

				return configPath
			},
			expectError: true,
			errorMsg:    "load configuration",
		},
		{
			name: "no recipients",
			setup: func() string {
				configPath := filepath.Join(tmpDir, "empty.toml")
				writeFile(t, configPath, `[files.default]
filename = ".kiln.env"
access = ["*"]
`)

				return configPath
			},
			expectError: true,
			errorMsg:    "no recipients in configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup()

			cfg, err := kiln.LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if cfg == nil {
				t.Error("Config should not be nil")
			}
		})
	}
}

// TestConfigPathResolution specifically tests the path resolution fix
func TestConfigPathResolution(t *testing.T) {
	// Create a nested directory structure to test relative path resolution
	baseDir := createTestDir(t)
	projectDir := filepath.Join(baseDir, "project")
	configDir := filepath.Join(projectDir, "config")
	envDir := filepath.Join(projectDir, "environments")

	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(envDir, 0o755)

	// Create config file with relative paths
	configPath := filepath.Join(configDir, "kiln.toml")
	configContent := `[recipients]
test = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

[files.local]
filename = "../environments/local.env"
access = ["*"]

[files.relative]
filename = "./relative.env"
access = ["*"]
`
	writeFile(t, configPath, configContent)

	// Load config from a different working directory to test path resolution
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to base directory (not config directory)
	os.Chdir(baseDir)

	cfg, err := kiln.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify that relative paths were resolved correctly
	localFile, err := cfg.GetEnvFile("local")
	if err != nil {
		t.Fatalf("GetEnvFile failed: %v", err)
	}

	expectedLocalPath := filepath.Join(envDir, "local.env")
	if localFile != expectedLocalPath {
		t.Errorf("Local file path resolution failed: expected %s, got %s", expectedLocalPath, localFile)
	}

	relativeFile, err := cfg.GetEnvFile("relative")
	if err != nil {
		t.Fatalf("GetEnvFile failed: %v", err)
	}

	expectedRelativePath := filepath.Join(configDir, "relative.env")
	if relativeFile != expectedRelativePath {
		t.Errorf("Relative file path resolution failed: expected %s, got %s", expectedRelativePath, relativeFile)
	}
}

// TestNewIdentityFromKey validates identity loading
func TestNewIdentityFromKey(t *testing.T) {
	tmpDir := createTestDir(t)

	// Create a valid key
	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	defer core.WipeData(privateKey)

	validKeyPath := filepath.Join(tmpDir, "valid.key")
	if err := core.SaveKeys(privateKey, publicKey, validKeyPath); err != nil {
		t.Fatalf("Failed to save keys: %v", err)
	}

	tests := []struct {
		name        string
		keyPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid key",
			keyPath:     validKeyPath,
			expectError: false,
		},
		{
			name:        "empty path",
			keyPath:     "",
			expectError: true,
			errorMsg:    "key path cannot be empty",
		},
		{
			name:        "nonexistent file",
			keyPath:     filepath.Join(tmpDir, "nonexistent.key"),
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "invalid key",
			keyPath: func() string {
				invalidKeyPath := filepath.Join(tmpDir, "invalid.key")
				writeFile(t, invalidKeyPath, "invalid key content")

				return invalidKeyPath
			}(),
			expectError: true,
			errorMsg:    "load identity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := kiln.NewIdentityFromKey(tt.keyPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if identity == nil {
				t.Error("Identity should not be nil")
			}

			// Clean up identity
			if identity != nil {
				identity.Cleanup()
			}
		})
	}
}

// TestEnvironmentVariableOperations tests the core read/write functionality
func TestEnvironmentVariableOperations(t *testing.T) {
	tmpDir := createTestDir(t)

	cfg, identity := setupTestEnvironment(t, tmpDir)
	defer identity.Cleanup()

	// Test setting a single variable
	testKey := "TEST_VAR"
	testValue := []byte("test_value")

	err := kiln.SetEnvironmentVar(identity, cfg, "default", testKey, testValue)
	if err != nil {
		t.Fatalf("SetEnvironmentVar failed: %v", err)
	}

	// Test getting the variable back
	retrievedValue, cleanup, err := kiln.GetEnvironmentVar(identity, cfg, "default", testKey)
	if err != nil {
		t.Fatalf("GetEnvironmentVar failed: %v", err)
	}
	defer cleanup()

	if !bytes.Equal(retrievedValue, testValue) {
		t.Errorf("Retrieved value doesn't match: expected %s, got %s", testValue, retrievedValue)
	}

	// Test getting all variables
	allVars, allCleanup, err := kiln.GetAllEnvironmentVars(identity, cfg, "default")
	if err != nil {
		t.Fatalf("GetAllEnvironmentVars failed: %v", err)
	}
	defer allCleanup()

	if len(allVars) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(allVars))
	}

	if !bytes.Equal(allVars[testKey], testValue) {
		t.Errorf("All vars: value doesn't match for key %s", testKey)
	}
}

// TestSetMultipleEnvironmentVars tests bulk variable operations
func TestSetMultipleEnvironmentVars(t *testing.T) {
	tmpDir := createTestDir(t)

	cfg, identity := setupTestEnvironment(t, tmpDir)
	defer identity.Cleanup()

	// Test setting multiple variables
	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost:5432/test"),
		"API_KEY":      []byte("secret-api-key"),
		"DEBUG":        []byte("true"),
	}

	err := kiln.SetMultipleEnvironmentVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SetMultipleEnvironmentVars failed: %v", err)
	}

	// Verify all variables were set
	allVars, cleanup, err := kiln.GetAllEnvironmentVars(identity, cfg, "default")
	if err != nil {
		t.Fatalf("GetAllEnvironmentVars failed: %v", err)
	}
	defer cleanup()

	if len(allVars) != len(testVars) {
		t.Errorf("Expected %d variables, got %d", len(testVars), len(allVars))
	}

	for key, expectedValue := range testVars {
		if actualValue, exists := allVars[key]; !exists {
			t.Errorf("Variable %s not found", key)
		} else if !bytes.Equal(actualValue, expectedValue) {
			t.Errorf("Variable %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}

// TestValidationErrors tests input validation
func TestValidationErrors(t *testing.T) {
	tmpDir := createTestDir(t)

	cfg, identity := setupTestEnvironment(t, tmpDir)
	defer identity.Cleanup()

	tests := []struct {
		name      string
		operation func() error
		errorMsg  string
	}{
		{
			name: "nil identity",
			operation: func() error {
				_, _, err := kiln.GetAllEnvironmentVars(nil, cfg, "default")
				if err != nil {
					return fmt.Errorf("GetAllEnvironmentVars with nil identity: %w", err)
				}

				return nil
			},
			errorMsg: "identity cannot be nil",
		},
		{
			name: "nil config",
			operation: func() error {
				_, _, err := kiln.GetAllEnvironmentVars(identity, nil, "default")
				if err != nil {
					return fmt.Errorf("GetAllEnvironmentVars with nil config: %w", err)
				}

				return nil
			},
			errorMsg: "config cannot be nil",
		},
		{
			name: "empty file name",
			operation: func() error {
				_, _, err := kiln.GetAllEnvironmentVars(identity, cfg, "")
				if err != nil {
					return fmt.Errorf("GetAllEnvironmentVars with empty file: %w", err)
				}

				return nil
			},
			errorMsg: "file name cannot be empty",
		},
		{
			name: "invalid file name",
			operation: func() error {
				_, _, err := kiln.GetAllEnvironmentVars(identity, cfg, "../invalid")
				if err != nil {
					return fmt.Errorf("GetAllEnvironmentVars with invalid file: %w", err)
				}

				return nil
			},
			errorMsg: "invalid file name",
		},
		{
			name: "invalid variable name",
			operation: func() error {
				err := kiln.SetEnvironmentVar(identity, cfg, "default", "invalid-name", []byte("value"))
				if err != nil {
					return fmt.Errorf("SetEnvironmentVar with invalid name: %w", err)
				}

				return nil
			},
			errorMsg: "invalid variable name",
		},
		{
			name: "empty variable value",
			operation: func() error {
				err := kiln.SetEnvironmentVar(identity, cfg, "default", "VALID_NAME", []byte(""))
				if err != nil {
					return fmt.Errorf("SetEnvironmentVar with empty value: %w", err)
				}

				return nil
			},
			errorMsg: "variable value cannot be empty",
		},
		{
			name: "variable value too large",
			operation: func() error {
				largeValue := make([]byte, 2*1024*1024) // 2MB
				err := kiln.SetEnvironmentVar(identity, cfg, "default", "LARGE_VAR", largeValue)
				if err != nil {
					return fmt.Errorf("SetEnvironmentVar with large value: %w", err)
				}

				return nil
			},
			errorMsg: "variable value too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			if err == nil {
				t.Errorf("Expected error but got none")
			} else if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestDiscoverPrivateKey tests key auto-discovery
func TestDiscoverPrivateKey(t *testing.T) {
	// This test is environment-dependent, so we'll test both success and failure cases
	// Test when no keys are available (should fail)
	// Note: This might pass if the test environment has actual keys
	keyPath, err := kiln.DiscoverPrivateKey()
	if err != nil {
		// Expected case - no keys found
		if !strings.Contains(err.Error(), "no compatible private key found") {
			t.Errorf("Expected 'no compatible private key found' error, got: %v", err)
		}
	} else {
		// Unexpected success - validate the returned key path
		if keyPath == "" {
			t.Error("Key path should not be empty when discovery succeeds")
		}

		if !core.FileExists(keyPath) {
			t.Errorf("Discovered key path doesn't exist: %s", keyPath)
		}
	}
}

// TestNonexistentVariableError tests error handling for missing variables
func TestNonexistentVariableError(t *testing.T) {
	tmpDir := createTestDir(t)

	cfg, identity := setupTestEnvironment(t, tmpDir)
	defer identity.Cleanup()

	// Try to get a variable that doesn't exist
	_, _, err := kiln.GetEnvironmentVar(identity, cfg, "default", "NONEXISTENT_VAR")
	if err == nil {
		t.Error("Expected error for nonexistent variable")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %v", err)
	}
}

// Helper functions

func createTestDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-lib-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	return tmpDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func setupTestEnvironment(t *testing.T, tmpDir string) (*kiln.Config, *kiln.Identity) {
	t.Helper()

	// Generate key pair
	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	defer core.WipeData(privateKey)

	// Save keys
	keyPath := filepath.Join(tmpDir, "test.key")
	if err := core.SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("Failed to save keys: %v", err)
	}

	// Create config
	cfg := config.NewConfig()
	cfg.Recipients["test-user"] = publicKey
	cfg.Files["default"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, ".kiln.env"),
		Access:   []string{"*"},
	}

	// Save config
	configPath := filepath.Join(tmpDir, "kiln.toml")
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config and identity using library functions
	loadedCfg, err := kiln.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	identity, err := kiln.NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("Failed to load identity: %v", err)
	}

	return loadedCfg, identity
}
