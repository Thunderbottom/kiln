package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

func (c *EditCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	envFilePath := cfg.GetEnvFile(c.File)

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

	plaintext, err := c.loadOrCreateTemplate(envFilePath, ageManager)
	if err != nil {
		return err
	}

	return c.editAndSave(plaintext, envFilePath, ageManager, globals)
}

func (c *EditCmd) loadOrCreateTemplate(envFilePath string, ageManager *crypto.AgeManager) ([]byte, error) {
	if _, err := os.Stat(envFilePath); err == nil {
		encrypted, err := os.ReadFile(envFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read environment file: %w", err)
		}
		return ageManager.Decrypt(encrypted)
	}

	return []byte(`# Environment Variables
# Format: KEY=value
DATABASE_URL=
API_TOKEN=
DEBUG=false
`), nil
}

func (c *EditCmd) editAndSave(plaintext []byte, envFilePath string, ageManager *crypto.AgeManager, globals *Globals) error {
	tempFile, err := utils.CreateSecureTempFile("kiln-edit-*.env")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer utils.SecureDelete(tempFile)

	if err := os.WriteFile(tempFile, plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	beforeStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}

	editorCmd := c.determineEditor()
	if err := c.launchEditor(editorCmd, tempFile); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	afterStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		fmt.Printf("No changes detected\n")
		return nil
	}

	return c.saveChanges(tempFile, envFilePath, ageManager, globals)
}

func (c *EditCmd) determineEditor() string {
	if c.Editor != "" {
		return c.Editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}

func (c *EditCmd) launchEditor(editor, tempFile string) error {
	cmd := exec.Command(editor, tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *EditCmd) saveChanges(tempFile, envFilePath string, ageManager *crypto.AgeManager, globals *Globals) error {
	modifiedContent, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
	}

	if _, err := env.ParseEnvFile(string(modifiedContent)); err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	encrypted, err := ageManager.Encrypt(modifiedContent)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return fmt.Errorf("failed to save environment file: %w", err)
	}

	fmt.Printf("Environment file updated: %s\n", envFilePath)
	return nil
}
