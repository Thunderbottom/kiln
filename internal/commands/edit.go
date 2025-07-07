package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
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
		content = []byte(core.FormatEnvFile(stringVars))
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

	// Simple cleanup with defer and signal handling
	tempFileName := tempFile.Name()
	cleaned := false
	cleanupTemp := func() {
		if !cleaned {
			_ = tempFile.Close()
			_ = os.Remove(tempFileName)
			cleaned = true
		}
	}
	defer cleanupTemp()

	// Setup simple signal handling for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cleanupTemp()
		os.Exit(130) // 128 + SIGINT(2)
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

	// Launch editor
	cmd.Logger().Debug().Str("editor", editor).Msg("launching editor")
	execCmd := exec.Command(editor, tempFileName)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
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
	if err != nil {
		return fmt.Errorf("read modified content: %w", err)
	}
	defer core.WipeData(modified)

	newStringVars, err := core.ParseEnvFile(string(modified))
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	newVars := make(map[string][]byte)
	for key, value := range newStringVars {
		newVars[key] = []byte(value)
	}

	if err := cmd.Session().SaveVars(c.File, newVars); err != nil {
		return fmt.Errorf("save changes: %w", err)
	}

	cmd.Logger().Info().Str("file", c.File).Msg("environment file updated")
	return nil
}
