package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
	Key    string `help:"Path to private key file to use for decryption" default:"~/.kiln/kiln.key"`
}

func (c *EditCmd) Run(globals *Globals) error {
	ctx := globals.Context()

	// Load existing vars or get empty map
	vars, err := core.LoadVars(ctx, globals.Config, c.File, c.Key)
	if err != nil {
		return err
	}

	// Format content for editing
	var content []byte
	if len(vars) > 0 {
		content = []byte(env.FormatEnvFile(vars))
	} else {
		content = []byte(`# Environment Variables
# Format: KEY=value
`)
	}

	// Create secure temp file
	tempFile, err := c.createTempFile(content, globals)
	if err != nil {
		return err
	}
	defer func() {
		if data, err := os.ReadFile(tempFile); err == nil {
			utils.WipeData(data)
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
		globals.Logger.Info("no changes detected")
		return nil
	}

	// Read and save changes
	modified, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
	}
	defer utils.WipeData(modified)

	newVars, err := env.ParseEnvFile(string(modified))
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	if err := core.SaveVars(ctx, globals.Config, c.File, newVars, c.Key); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	globals.Logger.Info("environment file updated", "file", c.File)
	return nil
}

// createTempFile creates a secure temporary file with the given content
func (c *EditCmd) createTempFile(content []byte, globals *Globals) (string, error) {
	// Create temp file in secure location
	tempDir := os.TempDir()
	if home, err := os.UserHomeDir(); err == nil {
		kilnTemp := filepath.Join(home, ".kiln", "tmp")
		if err := os.MkdirAll(kilnTemp, 0700); err == nil {
			tempDir = kilnTemp
		}
	}

	// Create temp file with secure permissions
	tempFile, err := os.CreateTemp(tempDir, "kiln-edit-*.env")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := tempFile.Close(); err != nil {
			globals.Logger.Debug("failed to close temp file", "error", err)
		}
	}()

	// Set secure permissions and write content
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

// launchEditor opens the specified editor with the temp file
func (c *EditCmd) launchEditor(tempFile string, globals *Globals) error {
	// Determine editor to use
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

	globals.Logger.Debug("launching editor", "editor", editor)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("editor timed out after 30 minutes")
		}
		return fmt.Errorf("editor failed: %w", err)
	}

	return nil
}
