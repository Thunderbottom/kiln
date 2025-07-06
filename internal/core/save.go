package core

import (
	"context"
	"fmt"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

// SaveVars encrypts and saves environment variables to file
// If keyPath is empty, uses default key loading logic.
func SaveVars(ctx context.Context, configPath, fileName string, envVars map[string]string, keyPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Load private key (custom path or default)
	var privateKey string
	if keyPath != "" {
		// Load from specific path
		privateKey, err = utils.LoadPrivateKeyFromFile(ctx, utils.ExpandPath(keyPath))
		if err != nil {
			return fmt.Errorf("failed to load private key from %s: %w", keyPath, err)
		}
	} else {
		// Use default key loading
		privateKey, err = utils.LoadPrivateKey(ctx)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}
	}
	defer utils.WipeString(privateKey)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Setup crypto
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return err
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return err
	}

	// Encrypt content
	content := env.FormatEnvFile(envVars)
	encrypted, err := ageManager.Encrypt(ctx, []byte(content))
	if err != nil {
		return err
	}

	// Write to file
	envFilePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return err
	}

	return nil
}
