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

func (c *InitKeyCmd) Run(globals *Globals) error {
	keyPath, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}

	globals.Logger.Debug().
		Str("key_path", keyPath).
		Bool("encrypt", c.Encrypt).
		Bool("force", c.Force).
		Msg("generating new key pair")

	if core.FileExists(keyPath) && !c.Force {
		globals.Logger.Error().
			Str("key_path", keyPath).
			Msg("private key already exists")
		return fmt.Errorf("private key already exists. Overwriting will make existing encrypted files unreadable. Use --force to overwrite (NOT RECOMMENDED)")
	}

	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		globals.Logger.Error().
			Err(err).
			Msg("failed to generate key pair")
		return fmt.Errorf("generate key pair: %w", err)
	}
	defer core.WipeData(privateKey)

	globals.Logger.Debug().Msg("key pair generated successfully")

	keyData := privateKey
	if c.Encrypt {
		globals.Logger.Debug().Msg("encrypting private key with passphrase")
		encryptedKey, err := core.EncryptPrivateKey(privateKey)
		if err != nil {
			globals.Logger.Error().
				Err(err).
				Msg("failed to encrypt private key")
			return fmt.Errorf("encrypt private key: %w", err)
		}
		keyData = encryptedKey
		defer core.WipeData(encryptedKey)
		globals.Logger.Debug().Msg("private key encrypted successfully")
	}

	if err := core.SavePrivateKey(keyData, keyPath); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("key_path", keyPath).
			Msg("failed to save private key")
		return fmt.Errorf("save private key: %w", err)
	}

	if !c.Encrypt {
		globals.Logger.Warn().Msg("private key is NOT password protected")
	}

	globals.Logger.Info().
		Str("private_key", keyPath).
		Bool("encrypted", c.Encrypt).
		Msg("key pair generated successfully")

	fmt.Printf("\nage public key: %s\n", publicKey)

	return nil
}

func (c *InitConfigCmd) Run(globals *Globals) error {
	globals.Logger.Debug().
		Str("config_path", c.Path).
		Int("public_key_count", len(c.PublicKeys)).
		Bool("force", c.Force).
		Msg("generating configuration file")

	if config.Exists(c.Path) && !c.Force {
		globals.Logger.Error().
			Str("config_path", c.Path).
			Msg("configuration already exists")
		return fmt.Errorf("configuration already exists. Use --force to overwrite")
	}

	var recipients []string
	for i, keyInput := range c.PublicKeys {
		globals.Logger.Debug().
			Int("index", i).
			Str("input", keyInput).
			Msg("loading public key")

		publicKey, err := core.LoadPublicKey(keyInput)
		if err != nil {
			globals.Logger.Error().
				Err(err).
				Str("input", keyInput).
				Msg("failed to load public key")
			return fmt.Errorf("load key %s: %w", keyInput, err)
		}
		recipients = append(recipients, publicKey)

		globals.Logger.Debug().
			Str("public_key", publicKey).
			Msg("public key loaded successfully")
	}

	cfg := config.NewConfig()
	for _, recipient := range recipients {
		cfg.AddRecipient(recipient)
	}

	if err := cfg.Save(c.Path); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("config_path", c.Path).
			Msg("failed to save configuration")
		return fmt.Errorf("save configuration: %w", err)
	}

	globals.Logger.Info().
		Str("config_path", c.Path).
		Int("recipients", len(recipients)).
		Msg("configuration created successfully")

	return nil
}
