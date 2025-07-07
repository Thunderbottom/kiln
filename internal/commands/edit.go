package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/thunderbottom/kiln/internal/core"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

func (c *EditCmd) Run(globals *Globals) error {
	cmd := NewCommand(globals)

	// Load existing variables
	vars, cleanup, err := cmd.Session().LoadVars(c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	// Convert to string format for editing
	stringVars := make(map[string]string)
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	// Format content for editing
	var content []byte
	if len(stringVars) > 0 {
		byteVars := make(map[string][]byte)
		for k, v := range stringVars {
			byteVars[k] = []byte(v)
		}
		content = core.FormatEnv(byteVars)
	} else {
		content = []byte(`# Environment Variables
# Format: KEY=value
`)
	}

	// Create secure temp file
	tempFile, err := os.CreateTemp("", "*.env")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tempFileName := tempFile.Name()

	var cleanupOnce sync.Once
	cleaned := false
	cleanupTemp := func() {
		cleanupOnce.Do(func() {
			if !cleaned {
				_ = tempFile.Close()
				_ = os.Remove(tempFileName)
				cleaned = true
			}
		})
	}
	defer cleanupTemp()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to communicate signal handling completion
	signalDone := make(chan struct{})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer close(signalDone)
		select {
		case <-sigChan:
			cmd.Logger().Debug().Msg("received interrupt signal, cleaning up...")
			cleanupTemp()
			cancel()
		case <-ctx.Done():
		}
	}()

	// Ensure signal notifications are stopped when we're done
	defer func() {
		signal.Stop(sigChan)
		cancel()
		<-signalDone
	}()

	// Write content to temp file
	if _, err := tempFile.Write(content); err != nil {
		return fmt.Errorf("write content to temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Get modification time before editing
	beforeStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}

	// Determine editor to use
	editor := c.Editor
	if editor == "" {
		if env := os.Getenv("EDITOR"); env != "" {
			editor = env
		} else {
			return fmt.Errorf("no editor specified: use --editor flag or set EDITOR environment variable")
		}
	}

	// Launch editor with context
	cmd.Logger().Debug().Str("editor", editor).Msg("launching editor")
	execCmd := exec.CommandContext(ctx, editor, tempFileName)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			cmd.Logger().Info().Msg("editor interrupted by signal")
			return fmt.Errorf("editor interrupted")
		}
		return fmt.Errorf("editor failed: %w", err)
	}

	// Check if context was cancelled after editor finished
	if ctx.Err() != nil {
		cmd.Logger().Info().Msg("operation cancelled")
		return fmt.Errorf("operation cancelled")
	}

	// Check if file was modified
	afterStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		cmd.Logger().Info().Msg("no changes detected")
		return nil
	}

	// Read and save changes
	modified, err := os.ReadFile(tempFileName)
	defer core.WipeData(modified)
	if err != nil {
		return fmt.Errorf("read modified content: %w", err)
	}

	envVars, err := core.ParseEnv(modified)
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	if err := cmd.Session().SaveVars(c.File, envVars); err != nil {
		return fmt.Errorf("save changes: %w", err)
	}

	cmd.Logger().Info().Str("file", c.File).Msg("environment file updated")
	return nil
}
