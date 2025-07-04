package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

// Globals contains global configuration shared across all commands
type Globals struct {
	Config  string
	Verbose bool
}

func loadEnvVars(globals *Globals, file string) (map[string]string, error) {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	envFilePath := cfg.GetEnvFile(file)
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("environment file not found: %s", envFilePath)
	}

	ctx := context.Background()
	privateKey, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return nil, fmt.Errorf("failed to add identity: %w", err)
	}

	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt environment file: %w", err)
	}

	defer utils.WipeData(plaintext)

	return env.ParseEnvFile(string(plaintext))
}
