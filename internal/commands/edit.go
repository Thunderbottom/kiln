package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/thunderbottom/kiln/internal/core"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

func (c *EditCmd) Run(globals *Globals) error {
	sess, err := globals.Session()
	if err != nil {
		return err
	}

	ctx := globals.Context()

	vars, err := sess.LoadVars(ctx, c.File)
	if err != nil {
		return err
	}

	// Format content for editing
	var content []byte
	if len(vars) > 0 {
		content = []byte(core.FormatEnvFile(vars))
	} else {
		content = []byte(`# Environment Variables
# Format: KEY=value
`)
	}

	// Create secure temp file
	tempFile, err := c.createTempFile(content)
	if err != nil {
		return err
	}
	defer func() {
		if data, err := os.ReadFile(tempFile); err == nil {
			core.WipeData(data)
		}
		_ = os.Remove(tempFile)
	}()

	// Get modification time before editing
	beforeStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Launch editor
	if err := c.launchEditor(tempFile, globals); err != nil {
		return err
	}

	// Check if file was modified
	afterStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info().Msg("no changes detected")
		return nil
	}

	// Read and save changes
	modified, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
	}
	defer core.WipeData(modified)

	newVars, err := core.ParseEnvFile(string(modified))
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	if err := sess.SaveVars(ctx, c.File, newVars); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	globals.Logger.Info().Str("file", c.File).Msg("environment file updated")
	return nil
}

func (c *EditCmd) createTempFile(content []byte) (string, error) {
	tempDir := os.TempDir()
	if home, err := os.UserHomeDir(); err == nil {
		kilnTemp := filepath.Join(home, ".kiln", "tmp")
		if err := os.MkdirAll(kilnTemp, 0700); err == nil {
			tempDir = kilnTemp
		}
	}

	tempFile, err := os.CreateTemp(tempDir, "kiln-edit-*.env")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	if err := tempFile.Chmod(0600); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to set permissions: %w", err)
	}

	if _, err := tempFile.Write(content); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	return tempFile.Name(), nil
}

func (c *EditCmd) launchEditor(tempFile string, globals *Globals) error {
	var editor string
	if c.Editor != "" {
		editor = c.Editor
	} else if env := os.Getenv("EDITOR"); env != "" {
		editor = env
	} else {
		return fmt.Errorf("no editor specified: use --editor flag or set EDITOR environment variable")
	}

	ctx, cancel := context.WithTimeout(globals.Context(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, editor, tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	globals.Logger.Debug().Str("editor", editor).Msg("launching editor")

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("editor timed out after 30 minutes")
		}
		return fmt.Errorf("editor failed: %w", err)
	}

	return nil
}
