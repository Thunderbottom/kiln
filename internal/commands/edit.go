package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/thunderbottom/kiln/internal/core"
)

type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

// Global cleanup tracking for signal handling
var (
	activeCleanups []func()
	cleanupMu      sync.Mutex
	signalSetup    sync.Once
)

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

	// Ensure all values are wiped when we're done
	defer func() {
		for _, value := range vars {
			core.WipeData(value)
		}
	}()

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
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Setup cleanup
	cleanup := func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}

	// Setup signal handling and register cleanup
	c.setupSignalHandling()
	c.registerCleanup(cleanup)
	defer func() {
		c.unregisterCleanup(cleanup)
		cleanup()
	}()

	// Write content to temp file
	if _, err := tempFile.Write(content); err != nil {
		return fmt.Errorf("failed to write content to temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get modification time before editing
	beforeStat, err := os.Stat(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Launch editor
	if err := c.launchEditor(tempFile.Name(), globals); err != nil {
		return err
	}

	// Check if file was modified
	afterStat, err := os.Stat(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info().Msg("no changes detected")
		return nil
	}

	// Read and save changes
	modified, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
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

	// Ensure new values are wiped when done
	defer func() {
		for _, value := range newVars {
			core.WipeData(value)
		}
	}()

	if err := sess.SaveVars(ctx, c.File, newVars); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	globals.Logger.Info().Str("file", c.File).Msg("environment file updated")
	return nil
}

// setupSignalHandling initializes signal handling for graceful cleanup
func (c *EditCmd) setupSignalHandling() {
	signalSetup.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			// Run all registered cleanup functions
			cleanupMu.Lock()
			for _, cleanup := range activeCleanups {
				cleanup()
			}
			cleanupMu.Unlock()

			// Exit with appropriate code
			// 128 + SIGINT(2) = 130
			os.Exit(130)
		}()
	})
}

// registerCleanup adds a cleanup function to be called on signal
func (c *EditCmd) registerCleanup(cleanup func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activeCleanups = append(activeCleanups, cleanup)
}

// unregisterCleanup removes a cleanup function from the signal handler
func (c *EditCmd) unregisterCleanup(targetCleanup func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	newCleanups := make([]func(), 0, len(activeCleanups))
	for _, cleanup := range activeCleanups {
		if fmt.Sprintf("%p", cleanup) != fmt.Sprintf("%p", targetCleanup) {
			newCleanups = append(newCleanups, cleanup)
		}
	}
	activeCleanups = newCleanups
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
