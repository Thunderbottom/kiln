package core

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/env"
)

// GetFileInfo returns file information or error
func GetFileInfo(configPath, fileName string) (string, os.FileInfo, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	filePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(filePath)
	return filePath, info, err
}

// MaskVars applies masking to sensitive environment variables
func MaskVars(envVars map[string]string, config *config.Config) map[string]string {
	if len(config.Security.MaskKeys) == 0 {
		return envVars
	}

	processed := make(map[string]string)
	for key, value := range envVars {
		if config.ShouldMaskKey(key) {
			processed[key] = config.MaskValue(value)
		} else {
			processed[key] = value
		}
	}
	return processed
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

// SetVar sets a single environment variable in the specified file
func SetVar(ctx context.Context, configPath, fileName, key, value string) error {
	vars, err := LoadVars(ctx, configPath, fileName, "")
	if err != nil {
		return err
	}
	vars[key] = value
	return SaveVars(ctx, configPath, fileName, vars, "")
}

// GetVar retrieves a single environment variable from the specified file
func GetVar(ctx context.Context, configPath, fileName, key string) (string, error) {
	vars, err := LoadVars(ctx, configPath, fileName, "")
	if err != nil {
		return "", err
	}

	value, exists := vars[key]
	if !exists {
		return "", fmt.Errorf("variable %s not found", key)
	}

	return value, nil
}

// ExportVars loads environment variables and optionally applies expansion
func ExportVars(ctx context.Context, configPath, fileName string, expand, allowCommands bool) (map[string]string, error) {
	vars, err := LoadVars(ctx, configPath, fileName, "")
	if err != nil {
		return nil, err
	}

	if expand {
		vars = env.ExpandVariables(vars, allowCommands)
	}

	return vars, nil
}

// CheckFile validates that a file can be decrypted and parsed
func CheckFile(ctx context.Context, configPath, fileName string) error {
	_, err := LoadVars(ctx, configPath, fileName, "")
	return err
}
