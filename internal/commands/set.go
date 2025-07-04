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

type SetCmd struct {
	Key   string `arg:"" help:"Environment variable key"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	envFilePath := cfg.GetEnvFile(c.File)

	ctx := context.Background()
	privateKey, _, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return fmt.Errorf("failed to add identity: %w", err)
	}

	envVars, err := c.loadExistingVars(envFilePath, ageManager)
	if err != nil {
		return err
	}

	envVars[c.Key] = c.Value
	content := env.FormatEnvFile(envVars)

	encrypted, err := ageManager.Encrypt([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return fmt.Errorf("failed to save environment file: %w", err)
	}

	fmt.Printf("Set %s in %s\n", c.Key, c.File)
	return nil
}

func (c *SetCmd) loadExistingVars(envFilePath string, ageManager *crypto.AgeManager) (map[string]string, error) {
	envVars := make(map[string]string)

	if utils.FileExists(envFilePath) {
		encrypted, err := os.ReadFile(envFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read environment file: %w", err)
		}

		plaintext, err := ageManager.Decrypt(encrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt environment file: %w", err)
		}

		envVars, err = env.ParseEnvFile(string(plaintext))
		if err != nil {
			return nil, fmt.Errorf("failed to parse environment file: %w", err)
		}
	}

	return envVars, nil
}
