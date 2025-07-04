package core

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

// Common error messages
const (
	ErrConfigLoad   = "failed to load configuration"
	ErrSetupCrypto  = "failed to setup encryption"
	ErrLoadKey      = "failed to load private key"
	ErrAddIdentity  = "failed to add identity"
	ErrFileNotFound = "environment file not found"
	ErrReadFile     = "failed to read environment file"
	ErrDecrypt      = "failed to decrypt environment file"
	ErrEncrypt      = "failed to encrypt content"
	ErrSaveFile     = "failed to save environment file"
)

// LoadEnvVars loads and decrypts environment variables from file
func LoadEnvVars(ctx context.Context, configPath, fileName string) (map[string]string, error) {
	cfg, ageManager, err := SetupEncryption(configPath)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	envFilePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return nil, err
	}
	if err := CheckFileExists(envFilePath); err != nil {
		return nil, err
	}

	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrReadFile, err)
	}

	return DecryptAndParse(ctx, ageManager, encrypted)
}

// LoadOrCreateEnvVars loads existing vars or returns empty map if file doesn't exist
func LoadOrCreateEnvVars(ctx context.Context, configPath, fileName string) (map[string]string, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrConfigLoad, err)
	}

	envFilePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return nil, err
	}
	if !utils.FileExists(envFilePath) {
		return make(map[string]string), nil
	}

	return LoadEnvVars(ctx, configPath, fileName)
}

// SaveEnvVars encrypts and saves environment variables to file
func SaveEnvVars(ctx context.Context, configPath, fileName string, envVars map[string]string) error {
	cfg, ageManager, err := SetupEncryption(configPath)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	content := env.FormatEnvFile(envVars)
	encrypted, err := ageManager.Encrypt(ctx, []byte(content))
	if err != nil {
		return fmt.Errorf("%s: %w", ErrEncrypt, err)
	}

	envFilePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return fmt.Errorf("%s: %w", ErrSaveFile, err)
	}

	return nil
}

// SetupEncryption loads config and sets up encryption manager with private key
func SetupEncryption(configPath string) (*config.Config, *crypto.AgeManager, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ErrConfigLoad, err)
	}

	ctx := context.Background()
	privateKey, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ErrLoadKey, err)
	}

	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ErrSetupCrypto, err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ErrAddIdentity, err)
	}

	return cfg, ageManager, nil
}

// CheckFileExists checks if file exists and returns appropriate error
func CheckFileExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("%s: %s", ErrFileNotFound, filePath)
	}
	return nil
}

// DecryptAndParse decrypts data and parses environment variables
func DecryptAndParse(ctx context.Context, ageManager *crypto.AgeManager, encrypted []byte) (map[string]string, error) {
	plaintext, err := ageManager.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrDecrypt, err)
	}

	defer utils.WipeData(plaintext)

	return env.ParseEnvFile(string(plaintext))
}

// GetFileInfo returns file information or error
func GetFileInfo(configPath, fileName string) (string, os.FileInfo, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", ErrConfigLoad, err)
	}

	filePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(filePath)
	return filePath, info, err
}

// ProcessEnvVars applies common processing to environment variables
func ProcessEnvVars(envVars map[string]string, maskSensitive bool) map[string]string {
	if !maskSensitive {
		return envVars
	}

	processed := make(map[string]string)
	for key, value := range envVars {
		if IsSensitiveKey(key) {
			value = MaskValue(value)
		}
		processed[key] = value
	}
	return processed
}

// IsSensitiveKey checks if a key contains sensitive information
func IsSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	sensitivePatterns := []string{"password", "secret", "token", "key", "auth", "api"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}

// MaskValue masks sensitive values
func MaskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

// SortedKeys returns sorted keys from environment variables map
func SortedKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ValidateEnvFile validates that a file can be decrypted and parsed
func ValidateEnvFile(ctx context.Context, configPath, fileName string) error {
	_, err := LoadEnvVars(ctx, configPath, fileName)
	return err
}
