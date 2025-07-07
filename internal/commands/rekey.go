package commands

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/core"
)

// RekeyCmd represents the rekey command for rotating encryption keys.
type RekeyCmd struct {
	File         string   `short:"f" help:"Environment file to rekey" placeholder:"default"`
	AddRecipient []string `help:"Add new recipient public keys" placeholder:"[age-pub-key]"`
	Force        bool     `help:"Force rekey without confirmation"`
}

// Run executes the rekey command, re-encrypting files with updated recipients.
func (c *RekeyCmd) Run(globals *Globals) error {
	if c.File == "" {
		return fmt.Errorf("--file flag is required")
	}

	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	cfg := session.Config()

	for _, recipient := range c.AddRecipient {
		if validateErr := core.ValidatePublicKey(recipient); validateErr != nil {
			return fmt.Errorf("invalid recipient key %s: %w", recipient, validateErr)
		}

		cfg.AddRecipient(recipient)
	}

	envFilePath, err := cfg.GetEnvFile(c.File)
	if err != nil {
		return err
	}

	if !core.FileExists(envFilePath) {
		globals.Logger.Debug().Str("file", c.File).Msg("file does not exist, skipping")

		return nil
	}

	envVars, cleanup, loadErr := session.LoadVars(c.File)
	if loadErr != nil {
		return fmt.Errorf("cannot decrypt file with current key - ensure you have access: %w", loadErr)
	}
	defer cleanup()

	if saveErr := cfg.Save(globals.Config); saveErr != nil {
		return fmt.Errorf("save updated config: %w", saveErr)
	}

	newSession, err := core.NewSession(globals.Config, globals.Key)
	if err != nil {
		return fmt.Errorf("create new session with updated recipients: %w", err)
	}

	if err := newSession.SaveVars(c.File, envVars); err != nil {
		return fmt.Errorf("save with updated recipients: %w", err)
	}

	globals.Logger.Info().
		Str("file", c.File).
		Int("recipients", len(cfg.Recipients)).
		Msg("successfully rekeyed file")

	return nil
}
