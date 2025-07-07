package commands

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/core"
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

	cmd := NewCommand(globals)
	cfg := cmd.Config()

	// Validate new recipients from CLI
	for _, recipient := range c.AddRecipient {
		if err := core.ValidatePublicKey(recipient); err != nil {
			return fmt.Errorf("invalid recipient key %s: %w", recipient, err)
		}
		cfg.AddRecipient(recipient)
	}

	// Check if we can currently decrypt the file
	envFilePath, err := cfg.GetEnvFile(c.File)
	if err != nil {
		return err
	}

	if !core.FileExists(envFilePath) {
		cmd.Logger().Debug().Str("file", c.File).Msg("file does not exist, skipping")
		return nil
	}

	// Try to load the file with current session
	envVars, cleanup, err := cmd.Session().LoadVars(c.File)
	if err != nil {
		return fmt.Errorf("cannot decrypt file with current key - ensure you have access: %w", err)
	}
	defer cleanup()

	// Create new session with updated recipients
	if err := cfg.Save(globals.Config); err != nil {
		return fmt.Errorf("save updated config: %w", err)
	}

	// Create new session with updated config
	newSess, err := core.NewSession(globals.Config, globals.Key)
	if err != nil {
		return fmt.Errorf("create new session with updated recipients: %w", err)
	}

	// Save with updated recipient list
	if err := newSess.SaveVars(c.File, envVars); err != nil {
		return fmt.Errorf("save with updated recipients: %w", err)
	}

	cmd.Logger().Info().
		Str("file", c.File).
		Int("recipients", len(cfg.Recipients)).
		Msg("successfully rekeyed file")

	return nil
}
