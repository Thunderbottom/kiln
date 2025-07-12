// Package kiln provides essential functions as a library for reading
// and writing encrypted environment variables.
package kiln

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

type (
	// Identity wraps age.Identity with concrete type safety and enhanced functionality
	Identity = core.Identity
	// Config represents the kiln configuration
	Config = config.Config
)

// LoadConfig loads and validates a kiln configuration file.
// Returns error if file doesn't exist, is malformed, or contains invalid configuration.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	if !core.FileExists(configPath) {
		return nil, fmt.Errorf("configuration file '%s' not found", configPath)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load configuration from '%s': %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in '%s': %w", configPath, err)
	}

	return cfg, nil
}

// NewIdentityFromKey loads an identity from a private key file.
// Supports both age and SSH private keys. Returns error if key is invalid or inaccessible.
func NewIdentityFromKey(keyPath string) (*Identity, error) {
	if keyPath == "" {
		return nil, fmt.Errorf("key path cannot be empty")
	}

	if !core.FileExists(keyPath) {
		return nil, fmt.Errorf("private key file '%s' not found", keyPath)
	}

	identity, err := core.NewIdentityFromKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load identity from '%s': %w", keyPath, err)
	}

	return identity, nil
}

// GetAllEnvironmentVars retrieves all environment variables from an encrypted file.
// Returns variables as map[string][]byte and a cleanup function that must be called
// to securely wipe sensitive data from memory.
//
// The cleanup function should always be called, typically with defer:
//
//	vars, cleanup, err := GetAllEnvironmentVars(identity, cfg, "production")
//	if err != nil { return err }
//	defer cleanup()
func GetAllEnvironmentVars(identity *Identity, cfg *Config, file string) (map[string][]byte, func(), error) {
	if err := validateInputs(identity, cfg, file); err != nil {
		return nil, nil, err
	}

	if !isValidFileName(file) {
		return nil, nil, fmt.Errorf("invalid file name '%s': cannot contain '..' or '/' characters", file)
	}

	variables, cleanup, err := core.GetAllEnvVars(identity, cfg, file)
	if err != nil {
		return nil, nil, fmt.Errorf("get environment variables from '%s': %w", file, err)
	}

	return variables, cleanup, nil
}

// GetEnvironmentVar retrieves a single environment variable from an encrypted file.
// Returns the variable value as []byte and a cleanup function that must be called
// to securely wipe sensitive data from memory.
//
// Returns error if variable doesn't exist in the specified file.
func GetEnvironmentVar(identity *Identity, cfg *Config, file, key string) ([]byte, func(), error) {
	if err := validateInputs(identity, cfg, file); err != nil {
		return nil, nil, err
	}

	if !isValidFileName(file) {
		return nil, nil, fmt.Errorf("invalid file name '%s': cannot contain '..' or '/' characters", file)
	}

	if !isValidVarName(key) {
		return nil, nil, fmt.Errorf("invalid variable name '%s': must start with letter or underscore, followed by letters, numbers, or underscores", key)
	}

	value, cleanup, err := core.GetEnvVar(identity, cfg, file, key)
	if err != nil {
		return nil, nil, fmt.Errorf("get variable '%s' from '%s': %w", key, file, err)
	}

	return value, cleanup, nil
}

// SetEnvironmentVar sets a single environment variable in an encrypted file.
// Creates the file if it doesn't exist. Re-encrypts the entire file with the new variable.
//
// Variable names must follow the POSIX shell pattern: start with letter or underscore,
// followed by letters, numbers, or underscores.
func SetEnvironmentVar(identity *Identity, cfg *Config, file, key string, value []byte) error {
	if err := validateInputs(identity, cfg, file); err != nil {
		return err
	}

	if !isValidFileName(file) {
		return fmt.Errorf("invalid file name '%s': cannot contain '..' or '/' characters", file)
	}

	if !isValidVarName(key) {
		return fmt.Errorf("invalid variable name '%s': must start with letter or underscore, followed by letters, numbers, or underscores", key)
	}

	if len(value) == 0 {
		return fmt.Errorf("variable value cannot be empty")
	}

	// Prevent extremely large values that could cause DoS
	const maxValueSize = 1024 * 1024 // 1MB
	if len(value) > maxValueSize {
		return fmt.Errorf("variable value too large: %d bytes (maximum %d bytes)", len(value), maxValueSize)
	}

	if err := core.SetEnvVar(identity, cfg, file, key, value); err != nil {
		return fmt.Errorf("set variable '%s' in '%s': %w", key, file, err)
	}

	return nil
}

// SetMultipleEnvironmentVars sets multiple environment variables in an encrypted file.
// More efficient than calling SetEnvironmentVar multiple times as it only re-encrypts once.
// Creates the file if it doesn't exist.
//
// All variable names must be valid. If any validation fails, no variables are set.
func SetMultipleEnvironmentVars(identity *Identity, cfg *Config, file string, vars map[string][]byte) error {
	if err := validateInputs(identity, cfg, file); err != nil {
		return err
	}

	if !isValidFileName(file) {
		return fmt.Errorf("invalid file name '%s': cannot contain '..' or '/' characters", file)
	}

	if len(vars) == 0 {
		return fmt.Errorf("no variables provided")
	}

	// Validate all inputs before making any changes
	const maxValueSize = 1024 * 1024 // 1MB

	totalSize := 0

	for key, value := range vars {
		if !isValidVarName(key) {
			return fmt.Errorf("invalid variable name '%s': must start with letter or underscore, followed by letters, numbers, or underscores", key)
		}

		if len(value) == 0 {
			return fmt.Errorf("variable '%s' value cannot be empty", key)
		}

		if len(value) > maxValueSize {
			return fmt.Errorf("variable '%s' value too large: %d bytes (maximum %d bytes)", key, len(value), maxValueSize)
		}

		totalSize += len(value)
	}

	// Prevent extremely large total payloads
	const maxTotalSize = 10 * 1024 * 1024 // 10MB
	if totalSize > maxTotalSize {
		return fmt.Errorf("total variables size too large: %d bytes (maximum %d bytes)", totalSize, maxTotalSize)
	}

	if err := core.SaveAllEnvVars(identity, cfg, file, vars); err != nil {
		return fmt.Errorf("set multiple variables in '%s': %w", file, err)
	}

	return nil
}

// DiscoverPrivateKey attempts to find a compatible private key in standard locations.
// Returns the path to the first usable private key found.
// Useful for applications that want to auto-discover keys like the CLI tool does.
func DiscoverPrivateKey() (string, error) {
	candidates := core.GetPrivateKeyCandidates()

	for _, candidate := range candidates {
		if core.FileExists(candidate) {
			// Verify it's actually a valid key by trying to load it
			if _, err := core.NewIdentityFromKey(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("no compatible private key found in standard locations")
}

// validateInputs performs common validation for identity, configuration, and file parameters.
func validateInputs(identity *Identity, cfg *Config, file string) error {
	if identity == nil {
		return fmt.Errorf("identity cannot be nil")
	}

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if file == "" {
		return fmt.Errorf("file name cannot be empty")
	}

	return nil
}

// isValidFileName checks if a file name is safe (prevents directory traversal attacks).
func isValidFileName(name string) bool {
	return core.IsValidFileName(name)
}

// isValidVarName checks if a variable name follows the required pattern.
func isValidVarName(name string) bool {
	return core.IsValidVarName(name)
}
