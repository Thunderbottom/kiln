package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
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
	ctx := context.Background()
	privateKey, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	newManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return fmt.Errorf("failed to setup new encryption: %w", err)
	}

	envFilePath := cfg.GetEnvFile(c.File)

	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		if globals.Verbose {
			fmt.Printf("File %s doesn't exist, skipping\n", c.File)
		}
		return nil
	}

	oldRecipients := cfg.Recipients[:len(cfg.Recipients)-len(c.AddRecipient)]
	oldManager, err := crypto.NewAgeManager(oldRecipients)
	if err != nil {
		return fmt.Errorf("failed to setup old encryption: %w", err)
	}

	if err := oldManager.AddIdentity(privateKey); err != nil {
		return fmt.Errorf("failed to add identity: %w", err)
	}

	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	plaintext, err := oldManager.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt with old keys: %w", err)
	}

	newEncrypted, err := newManager.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt with new keys: %w", err)
	}

	return utils.SaveFile(envFilePath, newEncrypted)
}
