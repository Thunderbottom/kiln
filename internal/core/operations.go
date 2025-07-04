// Package core provides encryption, decryption, and environment variable operations for kiln.
package core

import (
	"fmt"

	"filippo.io/age"

	"github.com/thunderbottom/kiln/internal/config"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// GetAllEnvVars decrypts, gets, and returns environment variables for a given file and identity.
func GetAllEnvVars(identity *Identity, cfg *config.Config, fileName string) (map[string][]byte, func(), error) {
	filePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return nil, nil, kerrors.ConfigError(fmt.Sprintf("file '%s' not configured", fileName), "check kiln.toml file definitions")
	}

	if !FileExists(filePath) {
		return make(map[string][]byte), func() {}, nil
	}

	recipientKeys, err := cfg.ResolveFileAccess(fileName)
	if err != nil {
		return nil, nil, kerrors.SecurityError(fmt.Sprintf("access denied for '%s'", fileName), "check file permissions in kiln.toml")
	}

	recipients, err := ParseRecipients(recipientKeys)
	if err != nil {
		return nil, nil, kerrors.ConfigError(fmt.Sprintf("invalid recipients for '%s'", fileName), "verify public keys in configuration")
	}

	crypto := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})

	encryptedData, err := ReadFile(filePath)
	if err != nil {
		return nil, nil, kerrors.FileAccessError("read", fileName, err)
	}

	plaintext, err := crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, nil, kerrors.SecurityError(fmt.Sprintf("cannot decrypt '%s'", fileName), "ensure your key has access to this file")
	}

	variables, err := ParseEnv(plaintext)
	if err != nil {
		WipeData(plaintext)

		return nil, nil, kerrors.ValidationError("environment format", fmt.Sprintf("file '%s' contains invalid format", fileName))
	}

	cleanup := func() {
		WipeData(plaintext)

		for _, value := range variables {
			WipeData(value)
		}
	}

	return variables, cleanup, nil
}

// SaveAllEnvVars encrypts and saves environment variables to the specified file.
func SaveAllEnvVars(identity *Identity, cfg *config.Config, fileName string, variables map[string][]byte) error {
	filePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return fmt.Errorf("file '%s' not configured", fileName)
	}

	recipientKeys, err := cfg.ResolveFileAccess(fileName)
	if err != nil {
		return fmt.Errorf("access error for '%s': %w", fileName, err)
	}

	recipients, err := ParseRecipients(recipientKeys)
	if err != nil {
		return fmt.Errorf("invalid recipients for '%s': %w", fileName, err)
	}

	crypto := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})

	content := FormatEnv(variables)
	defer WipeData(content)

	encryptedData, err := crypto.Encrypt(content)
	if err != nil {
		return fmt.Errorf("cannot encrypt '%s': %w", fileName, err)
	}

	return WriteFile(filePath, encryptedData)
}

// GetEnvVar retrieves a single environment variable from the specified file.
func GetEnvVar(identity *Identity, cfg *config.Config, fileName, key string) ([]byte, func(), error) {
	variables, cleanup, err := GetAllEnvVars(identity, cfg, fileName)
	if err != nil {
		return nil, nil, err
	}

	value, exists := variables[key]
	if !exists {
		cleanup()

		return nil, nil, fmt.Errorf("variable '%s' not found in '%s'", key, fileName)
	}

	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	cleanup()

	return valueCopy, func() { WipeData(valueCopy) }, nil
}

// SetEnvVar sets a single environment variable in the specified file.
func SetEnvVar(identity *Identity, cfg *config.Config, fileName, key string, value []byte) error {
	variables, cleanup, err := GetAllEnvVars(identity, cfg, fileName)
	if err != nil {
		return err
	}
	defer cleanup()

	newValue := make([]byte, len(value))
	copy(newValue, value)
	variables[key] = newValue

	return SaveAllEnvVars(identity, cfg, fileName, variables)
}

// CheckEnvFile validates that a file can be decrypted
func CheckEnvFile(identity *Identity, cfg *config.Config, fileName string) error {
	_, cleanup, err := GetAllEnvVars(identity, cfg, fileName)
	if cleanup != nil {
		defer cleanup()
	}

	return err
}
