package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

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
	// Create temporary directory
	tempDir, err := c.createTempDir()
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		// Clean up temporary directory and all contents
		os.RemoveAll(tempDir)
	}()

	// Create temporary file in the directory
	tempFile := filepath.Join(tempDir, "kiln-edit.env")
	if err := c.writeToTempFile(tempFile, plaintext); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	defer func() {
		// Wipe sensitive data from memory if file still exists
		if data, err := os.ReadFile(tempFile); err == nil {
			utils.WipeData(data)
		}
	}()

	beforeStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}

	editorCmd := c.determineEditor()
	if err := c.launchEditor(editorCmd, tempFile, globals); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	afterStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info("no changes detected")
		return nil
	}

	return c.saveChanges(tempFile, globals)
}

// createTempDir creates a temporary directory with secure permissions
func (c *EditCmd) createTempDir() (string, error) {
	// Get a base directory for temporary files
	baseDir, err := c.getTempBase()
	if err != nil {
		return "", err
	}

	// Create a unique directory name
	dirName := fmt.Sprintf("kiln-edit-%d-%d", os.Getpid(), time.Now().UnixNano())
	tempDir := filepath.Join(baseDir, dirName)

	// Create directory with restricted permissions (owner only: 0700)
	if err := os.Mkdir(tempDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Verify the directory has the correct permissions on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := c.verifyDirPermissions(tempDir); err != nil {
			os.Remove(tempDir)
			return "", err
		}
	}

	return tempDir, nil
}

// getTempBase returns a base directory for temporary files
func (c *EditCmd) getTempBase() (string, error) {
	// Try user's home directory first for better security
	if home, err := os.UserHomeDir(); err == nil {
		kilnTempDir := filepath.Join(home, ".kiln", "tmp")
		if err := os.MkdirAll(kilnTempDir, 0700); err == nil {
			return kilnTempDir, nil
		}
	}

	// Fall back to system temp, but create our own subdirectory
	systemTemp := os.TempDir()

	// Create a kiln-specific directory in system temp with secure permissions
	// Use current user ID to prevent conflicts between users
	var userSpecific string
	if runtime.GOOS != "windows" {
		userSpecific = fmt.Sprintf("kiln-%d", os.Getuid())
	} else {
		userSpecific = fmt.Sprintf("kiln-%d", os.Getpid())
	}

	kilnTempDir := filepath.Join(systemTemp, userSpecific)
	if err := os.MkdirAll(kilnTempDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp base directory: %w", err)
	}

	return kilnTempDir, nil
}

// verifyDirPermissions verifies that a directory has the expected permissions
func (c *EditCmd) verifyDirPermissions(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	// Verify permissions are 0700 (owner read/write/execute only)
	perm := info.Mode().Perm()
	expected := os.FileMode(0700)
	if perm != expected {
		return fmt.Errorf("directory permissions %v do not match expected %v", perm, expected)
	}

	return nil
}

// writeToTempFile writes data to a temporary file with secure permissions
func (c *EditCmd) writeToTempFile(filename string, data []byte) error {
	// Create file with secure permissions (owner read/write only: 0600)
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer file.Close()

	// Write data
	if _, err := file.Write(data); err != nil {
		os.Remove(filename) // Clean up on write failure
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk before editor opens
	if err := file.Sync(); err != nil {
		os.Remove(filename)
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	return nil
}

func (c *EditCmd) determineEditor() string {
	if c.Editor != "" {
		return c.Editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Use platform-appropriate defaults
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// launchEditor launches the editor with proper signal handling and timeout
func (c *EditCmd) launchEditor(editor, tempFile string, globals *Globals) error {
	// Create context with timeout to prevent hanging indefinitely
	ctx, cancel := context.WithTimeout(globals.Context(), 30*time.Minute)
	defer cancel()

	// Create command with context for timeout support
	cmd := exec.CommandContext(ctx, editor, tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set process group for proper signal handling on Unix-like systems
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	globals.Logger.Debug("launching editor", "editor", editor, "file", tempFile)

	if err := cmd.Run(); err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("editor operation timed out after 30 minutes")
		}

		// Check for exit errors and handle them gracefully
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				return fmt.Errorf("editor exited with code %d", exitError.ExitCode())
			}
		}
		return err
	}

	return nil
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
