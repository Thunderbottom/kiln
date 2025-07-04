package commands

import (
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/crypto"
)

type RekeyCmd struct {
	File         string   `short:"f" help:"Environment file to rekey"`
	AddRecipient []string `help:"Add new recipient public keys"`
	Force        bool     `help:"Force rekey without confirmation"`
}

func (c *RekeyCmd) Run(globals *Globals) error {
	if c.File == "" {
		return fmt.Errorf("--file flag is required")
	}

	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	originalCount := len(cfg.Recipients)

	// Validate and add new recipients
	for _, recipient := range c.AddRecipient {
		if err := crypto.ValidatePublicKey(recipient); err != nil {
			return fmt.Errorf("invalid recipient key %s: %w", recipient, err)
		}
		cfg.AddRecipient(recipient)
	}

	if len(cfg.Recipients) == originalCount {
		return fmt.Errorf("no new recipients to add")
	}

	return c.rekeyFile(cfg, globals)
}

func (c *RekeyCmd) rekeyFile(cfg *config.Config, globals *Globals) error {
	envFilePath := cfg.GetEnvFile(c.File)

	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		if globals.Verbose {
			fmt.Printf("File %s doesn't exist, skipping\n", c.File)
		}
		return nil
	}

	// Load existing environment variables using old recipients
	oldRecipients := cfg.Recipients[:len(cfg.Recipients)-len(c.AddRecipient)]
	oldCfg := *cfg
	oldCfg.Recipients = oldRecipients

	// Save old config temporarily to load with old recipients
	tempConfig := globals.Config + ".tmp"
	if err := oldCfg.Save(tempConfig); err != nil {
		return fmt.Errorf("failed to save temporary config: %w", err)
	}
	defer os.Remove(tempConfig)

	// Load environment variables with old configuration
	envVars, err := core.LoadEnvVars(tempConfig, c.File)
	if err != nil {
		return fmt.Errorf("failed to load with old keys: %w", err)
	}

	// Save configuration with new recipients
	if err := cfg.Save(globals.Config); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	// Save environment variables with new recipients
	if err := core.SaveEnvVars(globals.Config, c.File, envVars); err != nil {
		return fmt.Errorf("failed to save with new keys: %w", err)
	}

	fmt.Printf("Successfully rekeyed %s with %d new recipients\n", c.File, len(c.AddRecipient))
	return nil
}
