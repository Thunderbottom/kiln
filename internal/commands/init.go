package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// InitCmd represents the init command for initializing kiln projects.
type InitCmd struct {
	Key    *InitKeyCmd    `cmd:"" help:"Generate encryption key"`
	Config *InitConfigCmd `cmd:"" help:"Generate configuration file"`
}

// InitKeyCmd represents the key generation subcommand of init.
type InitKeyCmd struct {
	Path    string `help:"Path for private key" default:"~/.kiln/kiln.key" type:"path"`
	Encrypt bool   `help:"Save key with passphrase protection"`
	Force   bool   `help:"Overwrite existing key (dangerous!)"`
}

// InitConfigCmd represents the config generation subcommand of init.
type InitConfigCmd struct {
	Path       string            `help:"Path for config file" default:"kiln.toml"`
	Recipients map[string]string `help:"Named recipients in format 'name=key'" type:"agepubkey"`
	Force      bool              `help:"Overwrite existing config"`
}

func (c *InitKeyCmd) validate() error {
	if c.Path != "" && !core.IsValidFilePath(c.Path) {
		return kerrors.ValidationError("key path", "invalid file path")
	}

	return nil
}

// Run executes the init key command, generating a new encryption key pair.
func (c *InitKeyCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "init-key").Str("path", c.Path).Bool("encrypt", c.Encrypt).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	keyPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("resolve key path: %w", err)
	}

	if core.FileExists(keyPath) && !c.Force {
		return fmt.Errorf("key already exists at '%s' (use --force to override)", keyPath)
	}

	rt.Logger.Debug().Str("path", keyPath).Bool("encrypt", c.Encrypt).Msg("generating key pair")

	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}
	defer core.WipeData(privateKey)

	keyData := privateKey

	if c.Encrypt {
		encryptedKey, err := core.EncryptPrivateKey(privateKey)
		if err != nil {
			return fmt.Errorf("encrypt private key: %w", err)
		}

		keyData = encryptedKey
		defer core.WipeData(encryptedKey)
	}

	if err := core.SaveKeys(keyData, publicKey, keyPath); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	if !c.Encrypt {
		fmt.Fprintf(os.Stderr, "warning: private key is not password protected\n")
	}

	rt.Logger.Info().Str("path", keyPath).Msg("Private key generated")
	rt.Logger.Info().Str("public_key", publicKey).Str("path", keyPath+".pub").Msg("Public key stored")

	return nil
}

func (c *InitConfigCmd) validate() error {
	if c.Path != "" && !core.IsValidFilePath(c.Path) {
		return kerrors.ValidationError("config path", "invalid file path")
	}

	for name := range c.Recipients {
		if name == "" {
			return kerrors.ValidationError("recipient name", "name cannot be empty")
		}
	}

	return nil
}

// Run executes the init config command, creating a new configuration file.
func (c *InitConfigCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "init-config").Str("path", c.Path).Int("recipients", len(c.Recipients)).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	if config.Exists(c.Path) && !c.Force {
		return fmt.Errorf("config already exists at '%s' (use --force to override)", c.Path)
	}

	rt.Logger.Debug().Str("path", c.Path).Int("recipients", len(c.Recipients)).Msg("creating configuration")

	cfg := config.NewConfig()
	for name, publicKey := range c.Recipients {
		cfg.AddRecipient(name, publicKey)
	}

	if err := cfg.Save(c.Path); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}

	rt.Logger.Info().Str("path", c.Path).Msg("Configuration initialized")

	return nil
}
