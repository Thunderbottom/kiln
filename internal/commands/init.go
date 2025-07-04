package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
)

type InitCmd struct {
	From      string `help:"Use existing public key"`
	KeyOutput string `help:"Directory to save private key" default:"."`
	Force     bool   `help:"Overwrite existing configuration"`
}

func (c *InitCmd) Run(globals *Globals) error {
	if !c.Force && config.Exists(globals.Config) {
		return fmt.Errorf("kiln project already exists at %s", globals.Config)
	}

	cfg := config.NewConfig()
	return c.initEnv(cfg, globals)
}

func (c *InitCmd) initEnv(cfg *config.Config, globals *Globals) error {
	if c.From != "" {
		if err := crypto.ValidatePublicKey(c.From); err != nil {
			return fmt.Errorf("invalid public key: %w", err)
		}

		cfg.AddRecipient(c.From)
		if err := cfg.Save(globals.Config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	} else {
		privateKey, publicKey, err := crypto.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate key pair: %w", err)
		}

		cfg.AddRecipient(publicKey)

		keyDir := utils.ExpandPath(c.KeyOutput)
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			return fmt.Errorf("failed to create key directory: %w", err)
		}

		privateKeyFile := filepath.Join(keyDir, "kiln.key")
		if err := utils.SavePrivateKey(privateKey, privateKeyFile); err != nil {
			return fmt.Errorf("failed to save private key: %w", err)
		}

		if err := cfg.Save(globals.Config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		globals.Logger.Info("generated new age key pair")
		globals.Logger.Debug("generated age keys", "public key", publicKey, "private key", privateKeyFile)
		globals.Logger.Info("to configure key masking, add to kiln.toml:",
			"example", `[security]
mask_keys = ["API_TOKEN", "SECRET_KEY"]
show_chars = 4`)
	}

	file, err := cfg.GetEnvFile("default")
	if err != nil {
		return err
	}

	if err := c.createEmptyEnvFile(file, cfg.Recipients); err != nil {
		globals.Logger.Warn("failed to create empty env file", "error", err)
	}

	return nil
}

func (c *InitCmd) createEmptyEnvFile(envFile string, recipients []string) error {
	ageManager, err := crypto.NewAgeManager(recipients)
	if err != nil {
		return err
	}

	ctx := context.Background()

	template := `# Kiln Environment Variables
# Format: KEY=value
DATABASE_URL=
API_TOKEN=
DEBUG=false
`
	encrypted, err := ageManager.Encrypt(ctx, []byte(template))
	if err != nil {
		return err
	}

	return utils.SaveFile(envFile, encrypted)
}
