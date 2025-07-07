package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/thunderbottom/kiln/internal/config"
)

// Session holds configuration and crypto manager
type Session struct {
	config *config.Config
	crypto *AgeManager
}

// NewSession creates a session with config and private key
func NewSession(configPath, keyPath string) (*Session, error) {
	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Load private key
	keyAbsPath, _ := filepath.Abs(keyPath)
	privateKey, err := LoadPrivateKey(keyAbsPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	defer WipeData(privateKey)

	// Create crypto manager
	crypto, err := NewAgeManager(cfg.Recipients, privateKey)
	if err != nil {
		return nil, fmt.Errorf("create crypto manager: %w", err)
	}

	return &Session{
		config: cfg,
		crypto: crypto,
	}, nil
}

// LoadVars loads and decrypts environment variables from file with automatic cleanup
func (s *Session) LoadVars(fileName string) (map[string][]byte, func(), error) {
	envFilePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return nil, nil, err
	}

	// Return empty map if file doesn't exist
	if !FileExists(envFilePath) {
		return make(map[string][]byte), func() {}, nil
	}

	// Read encrypted file
	encrypted, err := ReadFile(envFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	// Decrypt using session's crypto manager
	plaintext, err := s.crypto.Decrypt(encrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt: %w", err)
	}
	defer WipeData(plaintext)

	// Parse environment file directly to []byte values
	vars, err := ParseEnvData(plaintext)
	if err != nil {
		return nil, nil, err
	}

	// Create cleanup function
	cleanup := func() {
		for _, value := range vars {
			WipeData(value)
		}
	}

	return vars, cleanup, nil
}

// SaveVars encrypts and saves environment variables to file
func (s *Session) SaveVars(fileName string, vars map[string][]byte) error {
	envFilePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	// Format content directly from []byte values
	content := FormatEnvData(vars)

	// Encrypt using session's crypto manager
	encrypted, err := s.crypto.Encrypt(content)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	// Save to file
	return WriteFile(envFilePath, encrypted)
}

// SetVar sets a single environment variable
func (s *Session) SetVar(fileName, key string, value []byte) error {
	vars, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return err
	}
	defer cleanup()

	// Create new vars map with the additional/updated value
	newVars := make(map[string][]byte)
	for k, v := range vars {
		newVars[k] = make([]byte, len(v))
		copy(newVars[k], v)
	}
	newVars[key] = make([]byte, len(value))
	copy(newVars[key], value)

	return s.SaveVars(fileName, newVars)
}

// GetVar gets a single environment variable with automatic cleanup
func (s *Session) GetVar(fileName, key string) ([]byte, func(), error) {
	vars, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return nil, nil, err
	}

	value, exists := vars[key]
	if !exists {
		cleanup()
		return nil, nil, fmt.Errorf("variable %s not found", key)
	}

	// Make a copy for return and clean up the original map
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	cleanup()

	// Return cleanup for the copy
	valueCleanup := func() {
		WipeData(valueCopy)
	}

	return valueCopy, valueCleanup, nil
}

// ExportVars loads variables with optional expansion and automatic cleanup
func (s *Session) ExportVars(fileName string, expand bool) (map[string][]byte, func(), error) {
	vars, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return nil, nil, err
	}

	if !expand {
		return vars, cleanup, nil
	}

	// Apply variable expansion - need string conversion for os.Expand
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

	// Create new cleanup that handles both original and result
	newCleanup := func() {
		cleanup() // Clean original vars
		for _, value := range result {
			WipeData(value)
		}
	}

	return result, newCleanup, nil
}

// CheckFile validates that a file can be decrypted
func (s *Session) CheckFile(fileName string) error {
	_, cleanup, err := s.LoadVars(fileName)
	if cleanup != nil {
		defer cleanup()
	}
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
