package commands

import (
	"fmt"
	"path/filepath"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

type InitCmd struct {
	Key    *InitKeyCmd    `cmd:"" help:"Generate encryption key"`
	Config *InitConfigCmd `cmd:"" help:"Generate configuration file"`
}

type InitKeyCmd struct {
	Path    string `help:"Path for private key" default:"~/.kiln/kiln.key" type:"path"`
	Encrypt bool   `help:"Save key with passphrase protection"`
	Force   bool   `help:"Overwrite existing key (dangerous!)"`
}

type InitConfigCmd struct {
	Path       string   `help:"Path for config file" default:"kiln.toml"`
	PublicKeys []string `help:"Path to public key file(s) or public key strings" required:""`
	Force      bool     `help:"Overwrite existing config"`
}

// Run generates a new encryption key
func (c *InitKeyCmd) Run(globals *Globals) error {
	keyPath, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}

	// Check if key already exists
	if core.FileExists(keyPath) && !c.Force {
		return fmt.Errorf("private key already exists at %s. Overwriting will make existing encrypted files unreadable. Use --force to overwrite (NOT RECOMMENDED)", keyPath)
	}

	// Generate key pair
	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}
	defer core.WipeData(privateKey)

	// Encrypt private key if requested
	keyToSave := privateKey
	if c.Encrypt {
		encryptedKey, err := core.EncryptPrivateKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		keyToSave = encryptedKey
		defer core.WipeData(encryptedKey)
	}

	// Save private key
	if err := core.SavePrivateKey(keyToSave, keyPath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if !c.Encrypt {
		globals.Logger.Warn().Msg("private key is NOT password protected")
	}

	globals.Logger.Info().
		Str("private", keyPath).
		Bool("encrypted", c.Encrypt).
		Msg("key pair generated")
	fmt.Printf("\nage public key: %s\n", publicKey)

	return nil
}

// Run generates a new configuration file
func (c *InitConfigCmd) Run(globals *Globals) error {
	// Check if config already exists
	if config.Exists(c.Path) && !c.Force {
		return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", c.Path)
	}

	// Load all public keys using consolidated function
	var recipients []string
	for _, keyInput := range c.PublicKeys {
		publicKey, err := core.LoadPublicKey(keyInput)
		if err != nil {
			return fmt.Errorf("failed to load key %s: %w", keyInput, err)
		}
		recipients = append(recipients, publicKey)
	}

	// Create and save config
	cfg := config.NewConfig()
	for _, recipient := range recipients {
		cfg.AddRecipient(recipient)
	}

	if err := cfg.Save(c.Path); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	globals.Logger.Info().
		Str("path", c.Path).
		Int("recipients", len(recipients)).
		Msg("configuration created")

	return nil
}
