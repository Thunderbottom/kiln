package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

func (c *EditCmd) Run(globals *Globals) error {
	plaintext, err := c.loadOrCreateTemplate(globals)
	if err != nil {
		return err
	}

	return c.editAndSave(plaintext, globals)
}

func (c *EditCmd) loadOrCreateTemplate(globals *Globals) ([]byte, error) {
	// Try to load existing file
	ctx := globals.Context()
	envVars, err := core.LoadOrCreateEnvVars(ctx, globals.Config, c.File)
	if err != nil {
		return nil, err
	}

	// If we got variables, format them
	if len(envVars) > 0 {
		content := env.FormatEnvFile(envVars)
		return []byte(content), nil
	}

	// Otherwise return template
	return []byte(`# Environment Variables
# Format: KEY=value
DATABASE_URL=
API_TOKEN=
DEBUG=false
`), nil
}

func (c *EditCmd) editAndSave(plaintext []byte, globals *Globals) error {
	tempFile, err := os.CreateTemp("", "kiln-edit-*.env")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	if err := os.WriteFile(tempFile.Name(), plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	beforeStat, err := os.Stat(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}

	editorCmd := c.determineEditor()
	if err := c.launchEditor(editorCmd, tempFile.Name()); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	afterStat, err := os.Stat(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to stat temporary file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info("no changes detected")
		return nil
	}

	return c.saveChanges(tempFile.Name(), globals)
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

func (c *EditCmd) saveChanges(tempFile string, globals *Globals) error {
	modifiedContent, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
	}

	defer utils.WipeData(modifiedContent)

	envVars, err := env.ParseEnvFile(string(modifiedContent))
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	ctx := globals.Context()
	if err := core.SaveEnvVars(ctx, globals.Config, c.File, envVars); err != nil {
		return fmt.Errorf("failed to save environment file: %w", err)
	}

	cfg, _ := config.Load(globals.Config)
	envFilePath, err := cfg.GetEnvFile(c.File)
	if err != nil {
		return err
	}

	globals.Logger.Info("environment file updated", "path", envFilePath)

	return nil
}
