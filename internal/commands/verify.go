package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type VerifyCmd struct {
	File string `short:"f" help:"Verify specific file"`
}

func (c *VerifyCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ctx := context.Background()
	privateKey, _, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return fmt.Errorf("failed to add identity: %w", err)
	}

	filesToVerify := []string{"default"}
	if c.File != "" {
		filesToVerify = []string{c.File}
	} else {
		filesToVerify = make([]string, 0, len(cfg.Files))
		for name := range cfg.Files {
			filesToVerify = append(filesToVerify, name)
		}
	}

	successful := 0
	for _, fileName := range filesToVerify {
		if err := c.verifyFile(cfg, fileName, ageManager, globals); err != nil {
			fmt.Printf("  %s: Error - %v\n", fileName, err)
		} else {
			fmt.Printf("  %s: OK\n", fileName)
			successful++
		}
	}

	fmt.Printf("Verification: %d/%d files verified successfully\n", successful, len(filesToVerify))

	if successful < len(filesToVerify) {
		return fmt.Errorf("verification failed for %d file(s)", len(filesToVerify)-successful)
	}

	return nil
}

func (c *VerifyCmd) verifyFile(cfg *config.Config, fileName string, ageManager *crypto.AgeManager, globals *Globals) error {
	filePath := cfg.GetEnvFile(fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}

	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	_, err = env.ParseEnvFile(string(plaintext))
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	return nil
}
