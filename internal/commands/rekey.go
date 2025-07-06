package commands

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
)

type RekeyCmd struct {
	File         string   `short:"f" help:"Environment file to rekey"`
	AddRecipient []string `help:"Add new recipient public keys"`
	Force        bool     `help:"Force rekey without confirmation"`
	Key          string   `help:"Path to private key file to use for decryption" default:"~/.kiln/kiln.key" type:"path"`
}

func (c *RekeyCmd) Run(globals *Globals) error {
	if c.File == "" {
		return fmt.Errorf("--file flag is required")
	}

	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate new recipients from CLI
	for _, recipient := range c.AddRecipient {
		if err := crypto.ValidatePublicKey(recipient); err != nil {
			return fmt.Errorf("invalid recipient key %s: %w", recipient, err)
		}
		cfg.AddRecipient(recipient)
	}

	// Check if we can currently decrypt the file
	ctx := globals.Context()
	envFilePath, err := cfg.GetEnvFile(c.File)
	if err != nil {
		return err
	}

	if !utils.FileExists(envFilePath) {
		globals.Logger.Debug("file does not exist, skipping", "filename", c.File)
		return nil
	}

	// Try to load the file with current key
	_, loadErr := core.LoadVars(ctx, globals.Config, c.File, c.Key)
	if loadErr != nil {
		return fmt.Errorf("cannot decrypt file with current key - ensure you have access: %w", loadErr)
	}

	// Re-encrypt with all recipients in config
	envVars, err := core.LoadVars(ctx, globals.Config, c.File, c.Key)
	if err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Save with updated recipient list
	if err := core.SaveVars(ctx, globals.Config, c.File, envVars, c.Key); err != nil {
		return fmt.Errorf("failed to save with updated recipients: %w", err)
	}

	globals.Logger.Info("successfully rekeyed file", "file", c.File, "total_recipients", len(cfg.Recipients))
	return nil
}
