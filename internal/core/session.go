package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/thunderbottom/kiln/internal/config"
)

// Session holds configuration and crypto manager for kiln operations
type Session struct {
	config *config.Config
	crypto *AgeManager
}

// NewSession creates a session with config and private key
func NewSession(configPath, keyPath string) (*Session, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	keyAbsPath, _ := filepath.Abs(keyPath)
	privateKey, err := LoadPrivateKey(keyAbsPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	defer WipeData(privateKey)

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
	filePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return nil, nil, err
	}

	if !FileExists(filePath) {
		return make(map[string][]byte), func() {}, nil
	}

	encryptedData, err := ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	plaintext, err := s.crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt: %w", err)
	}
	defer WipeData(plaintext)

	variables, err := ParseEnv(plaintext)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		for _, value := range variables {
			WipeData(value)
		}
	}

	return variables, cleanup, nil
}

// SaveVars encrypts and saves environment variables to file
func (s *Session) SaveVars(fileName string, variables map[string][]byte) error {
	filePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	content := FormatEnv(variables)

	encryptedData, err := s.crypto.Encrypt(content)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	return WriteFile(filePath, encryptedData)
}

// SetVar sets a single environment variable
func (s *Session) SetVar(fileName, key string, value []byte) error {
	variables, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return err
	}
	defer cleanup()

	updatedVariables := make(map[string][]byte)
	for k, v := range variables {
		updatedVariables[k] = make([]byte, len(v))
		copy(updatedVariables[k], v)
	}
	updatedVariables[key] = make([]byte, len(value))
	copy(updatedVariables[key], value)

	return s.SaveVars(fileName, updatedVariables)
}

// GetVar gets a single environment variable with automatic cleanup
func (s *Session) GetVar(fileName, key string) ([]byte, func(), error) {
	variables, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return nil, nil, err
	}

	value, exists := variables[key]
	if !exists {
		cleanup()
		return nil, nil, fmt.Errorf("variable %s not found", key)
	}

	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	cleanup()

	valueCleanup := func() {
		WipeData(valueCopy)
	}

	return valueCopy, valueCleanup, nil
}

// ExportVars loads variables with optional expansion and automatic cleanup
func (s *Session) ExportVars(fileName string, expand bool) (map[string][]byte, func(), error) {
	variables, cleanup, err := s.LoadVars(fileName)
	if err != nil {
		return nil, nil, err
	}

	if !expand {
		return variables, cleanup, nil
	}

	stringVariables := make(map[string]string)
	for key, value := range variables {
		stringVariables[key] = string(value)
	}

	expandedVariables := make(map[string]string)
	for key, value := range stringVariables {
		expandedVariables[key] = os.Expand(value, func(expandKey string) string {
			if val, exists := stringVariables[expandKey]; exists {
				return val
			}
			return os.Getenv(expandKey)
		})
	}

	result := make(map[string][]byte)
	for key, value := range expandedVariables {
		result[key] = []byte(value)
	}

	newCleanup := func() {
		cleanup()
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

// GetFileInfo returns file path and information
func (s *Session) GetFileInfo(fileName string) (string, os.FileInfo, error) {
	filePath, err := s.config.GetEnvFile(fileName)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(filePath)
	return filePath, info, err
}

// Config returns the session's configuration
func (s *Session) Config() *config.Config {
	return s.config
}

// SortedKeys returns sorted keys from environment variables map
func SortedKeys(variables map[string][]byte) []string {
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
