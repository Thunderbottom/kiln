package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/thunderbottom/kiln/internal/config"
)

// Session holds a configured crypto manager for a command execution
type Session struct {
	config     *config.Config
	ageManager *ageManager
}

// NewSession loads the private key once and creates a reusable session
func NewSession(configPath, keyPath string) (*Session, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load private key once per session
	keyAbsPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, err
	}

	privateKey, err := loadPrivateKey(keyAbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}
	defer WipeData(privateKey)

	// Setup crypto manager once per session
	ageManager, err := newAgeManager(cfg.Recipients)
	if err != nil {
		return nil, err
	}

	if err := ageManager.addIdentity(privateKey); err != nil {
		return nil, err
	}

	return &Session{
		config:     cfg,
		ageManager: ageManager,
	}, nil
}

// LoadVars loads and decrypts environment variables from file
func (s *Session) LoadVars(ctx context.Context, fileName string) (map[string][]byte, error) {
	envFilePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return nil, err
	}

	// Return empty map if file doesn't exist
	if !FileExists(envFilePath) {
		return make(map[string][]byte), nil
	}

	// Read encrypted file
	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decrypt using session's cached crypto manager
	plaintext, err := s.ageManager.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	defer WipeData(plaintext)

	// Parse environment file and convert to []byte values
	stringVars, err := ParseEnvFile(string(plaintext))
	if err != nil {
		return nil, err
	}

	// Convert string values to []byte
	vars := make(map[string][]byte)
	for key, value := range stringVars {
		vars[key] = []byte(value)
	}

	return vars, nil
}

// SaveVars encrypts and saves environment variables to file
func (s *Session) SaveVars(ctx context.Context, fileName string, vars map[string][]byte) error {
	envFilePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	// Convert []byte values to string for formatting
	stringVars := make(map[string]string)
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	// Format content
	content := FormatEnvFile(stringVars)

	// Encrypt using session's cached crypto manager
	encrypted, err := s.ageManager.encrypt([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Save to file
	return saveFile(envFilePath, encrypted)
}

// SetVar sets a single environment variable
func (s *Session) SetVar(ctx context.Context, fileName, key string, value []byte) error {
	vars, err := s.LoadVars(ctx, fileName)
	if err != nil {
		return err
	}
	vars[key] = value
	return s.SaveVars(ctx, fileName, vars)
}

// GetVar gets a single environment variable
func (s *Session) GetVar(ctx context.Context, fileName, key string) ([]byte, error) {
	vars, err := s.LoadVars(ctx, fileName)
	if err != nil {
		return nil, err
	}

	value, exists := vars[key]
	if !exists {
		return nil, fmt.Errorf("variable %s not found", key)
	}
	return value, nil
}

// ExportVars loads variables with optional expansion
func (s *Session) ExportVars(ctx context.Context, fileName string, expand bool) (map[string][]byte, error) {
	vars, err := s.LoadVars(ctx, fileName)
	if err != nil {
		return nil, err
	}

	if !expand {
		return vars, nil
	}

	// Apply variable expansion
	stringVars := make(map[string]string)
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	expanded := make(map[string]string)
	for key, value := range stringVars {
		expanded[key] = os.Expand(value, func(expandKey string) string {
			if val, exists := stringVars[expandKey]; exists {
				return val
			}
			return os.Getenv(expandKey)
		})
	}

	// Convert back to []byte
	result := make(map[string][]byte)
	for key, value := range expanded {
		result[key] = []byte(value)
	}

	return result, nil
}

// CheckFile validates that a file can be decrypted
func (s *Session) CheckFile(ctx context.Context, fileName string) error {
	_, err := s.LoadVars(ctx, fileName)
	return err
}

// GetFileInfo returns file information
func (s *Session) GetFileInfo(fileName string) (string, os.FileInfo, error) {
	filePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(filePath)
	return filePath, info, err
}

// SortedKeys returns sorted keys from environment variables map (exported for export command)
func SortedKeys(envVars map[string][]byte) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Config returns the session's configuration
func (s *Session) Config() *config.Config {
	return s.config
}
