package core

import (
	"context"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

// LoadVars loads environment variables from an encrypted file.
// If the file doesn't exist, it returns an empty map instead of an error.
// If keyPath is empty, uses default key loading logic.
// If keyPath is specified, loads the private key from that specific path.
func LoadVars(ctx context.Context, configPath, fileName, keyPath string) (map[string]string, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	envFilePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return nil, err
	}

	if !utils.FileExists(envFilePath) {
		// Return empty map if file doesn't exist - this is not an error
		return make(map[string]string), nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read encrypted file
	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	// Load private key (custom path or default)
	var privateKey string
	if keyPath != "" {
		// Load from specific path
		privateKey, err = utils.LoadPrivateKeyFromFile(ctx, utils.ExpandPath(keyPath))
		if err != nil {
			return nil, err
		}
	} else {
		// Use default key loading
		privateKey, err = utils.LoadPrivateKey(ctx)
		if err != nil {
			return nil, err
		}
	}
	defer utils.WipeString(privateKey)

	// Setup crypto
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return nil, err
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return nil, err
	}

	// Decrypt and parse
	plaintext, err := ageManager.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}

	defer utils.WipeData(plaintext)

	return env.ParseEnvFile(string(plaintext))
}
