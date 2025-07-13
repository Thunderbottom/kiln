package commands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// RekeyCmd represents the rekey command for rotating encryption keys.
type RekeyCmd struct {
	File         string   `short:"f" help:"Environment file to rekey" required:"true"`
	AddRecipient []string `help:"Add new named recipients in format 'name=key'" placeholder:"name=age-pub-key"`
	Force        bool     `help:"Force rekey without confirmation"`
}

func (c *RekeyCmd) validate() error {
	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	if len(c.AddRecipient) == 0 {
		return kerrors.ValidationError("recipients", "no recipients specified (use --add-recipient name=key)")
	}

	for _, recipient := range c.AddRecipient {
		if err := c.validateRecipient(recipient); err != nil {
			return kerrors.ValidationError("recipient", fmt.Sprintf("'%s': %s", recipient, err.Error()))
		}
	}

	return nil
}

func (c *RekeyCmd) validateRecipient(recipient string) error {
	parts := strings.SplitN(recipient, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format (use 'name=public-key')")
	}

	name := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])

	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if core.IsPrivateKey(key) {
		return fmt.Errorf("private key provided instead of public key")
	}

	if err := core.ValidatePublicKey(key); err != nil {
		return fmt.Errorf("invalid public key format")
	}

	return nil
}

// Run executes the rekey command, re-encrypting files with updated recipients.
func (c *RekeyCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "rekey").Str("file", c.File).Int("new_recipients", len(c.AddRecipient)).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	// Check for duplicate recipients
	if err := c.checkDuplicateRecipients(cfg); err != nil {
		return err
	}

	rt.Logger.Debug().Str("file", c.File).Int("new_recipients", len(c.AddRecipient)).Msg("rekeying file")

	if len(c.AddRecipient) > 0 {
		rt.Logger.Info().Str("file", c.File).Int("new_recipients", len(c.AddRecipient)).Msg("Rekeying with new recipients")
	} else {
		rt.Logger.Info().Str("file", c.File).Msg("Rekeying")
	}

	if err := c.addRecipientsToConfig(cfg); err != nil {
		return err
	}

	c.updateFileAccess(cfg)

	if err := c.rekeyFile(rt, cfg); err != nil {
		return err
	}

	return nil
}

// addRecipientsToConfig adds new recipients to the configuration
func (c *RekeyCmd) addRecipientsToConfig(cfg *config.Config) error {
	for _, recipient := range c.AddRecipient {
		parts := strings.SplitN(recipient, "=", 2)
		name := strings.TrimSpace(parts[0])
		publicKey := strings.TrimSpace(parts[1])

		cfg.AddRecipient(name, publicKey)
	}

	return nil
}

// checkDuplicateRecipients verifies new recipients don't conflict with existing ones.
func (c *RekeyCmd) checkDuplicateRecipients(cfg *config.Config) error {
	for _, recipient := range c.AddRecipient {
		parts := strings.SplitN(recipient, "=", 2)
		name := strings.TrimSpace(parts[0])
		newKey := strings.TrimSpace(parts[1])

		if existingKey, exists := cfg.Recipients[name]; exists {
			if existingKey != newKey {
				return kerrors.ConfigError(
					fmt.Sprintf("recipient '%s' already exists with different key", name),
					"use different name or remove existing recipient first")
			}
		}
	}

	return nil
}

// updateFileAccess adds new recipients to the file's access control list
func (c *RekeyCmd) updateFileAccess(cfg *config.Config) {
	fileConfig, exists := cfg.Files[c.File]
	if !exists {
		return
	}

	for _, recipient := range c.AddRecipient {
		parts := strings.SplitN(recipient, "=", 2)
		name := strings.TrimSpace(parts[0])

		if c.hasFileAccess(cfg, fileConfig, name) {
			continue
		}

		fileConfig.Access = append(fileConfig.Access, name)
		cfg.Files[c.File] = fileConfig
	}
}

// hasFileAccess checks if a recipient already has access to the file
func (c *RekeyCmd) hasFileAccess(cfg *config.Config, fileConfig config.FileConfig, name string) bool {
	if slices.Contains(fileConfig.Access, name) || slices.Contains(fileConfig.Access, "*") {
		return true
	}

	for _, accessor := range fileConfig.Access {
		if groupMembers, isGroup := cfg.Groups[accessor]; isGroup {
			return slices.Contains(groupMembers, name)
		}
	}

	return false
}

// rekeyFile re-encrypts the environment file with updated recipients
func (c *RekeyCmd) rekeyFile(rt *Runtime, cfg *config.Config) error {
	filePath, err := cfg.GetEnvFile(c.File)
	if err != nil {
		return err
	}

	if !core.FileExists(filePath) {
		rt.Logger.Info().Str("file", c.File).Msg("rekeyed (file will be created with new recipients when variables are added)")

		return nil
	}

	identity, err := rt.Identity()
	if err != nil {
		return err
	}

	envVars, cleanup, loadErr := core.GetAllEnvVars(identity, cfg, c.File)
	if loadErr != nil {
		return loadErr
	}
	defer cleanup()

	if saveErr := cfg.Save(rt.ConfigPath()); saveErr != nil {
		return saveErr
	}

	if err := core.SaveAllEnvVars(identity, cfg, c.File, envVars); err != nil {
		return err
	}

	rt.Logger.Info().Str("file", c.File).Int("added", len(c.AddRecipient)).Int("total", len(cfg.Recipients)).Msg("rekeyed with new recipients")

	return nil
}
