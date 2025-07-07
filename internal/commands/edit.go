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

// EditCmd represents the edit command for modifying encrypted environment variables.
type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use"`
}

// Run executes the edit command, opening an editor to modify environment variables.
func (c *EditCmd) Run(globals *Globals) error {
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	content, err := c.prepareContent(session)
	if err != nil {
		return err
	}

	tempFile, cleanupTemp, err := c.createTempFile(content, globals)
	if err != nil {
		return err
	}
	defer cleanupTemp()

	ctx, cancelCtx := c.setupSignalHandling(cleanupTemp, globals)
	defer cancelCtx()

	beforeStat, err := c.getFileStats(tempFile.Name())
	if err != nil {
		return err
	}

	if err := c.launchEditor(ctx, tempFile.Name(), globals); err != nil {
		return err
	}

	return c.processChanges(session, tempFile.Name(), beforeStat, globals)
}

func (c *EditCmd) prepareContent(session *core.Session) ([]byte, error) {
	vars, cleanup, err := session.LoadVars(c.File)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if len(vars) > 0 {
		return core.FormatEnv(vars), nil
	}

	return []byte("# Environment Variables\n# Format: KEY=value\n"), nil
}

func (c *EditCmd) createTempFile(content []byte, globals *Globals) (*os.File, func(), error) {
	tempFile, err := os.CreateTemp("", "*.env")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp file: %w", err)
	}

	tempFileName := tempFile.Name()

	var cleanupOnce sync.Once

	cleaned := false

	cleanupTemp := func() {
		cleanupOnce.Do(func() {
			if !cleaned {
				_ = tempFile.Close()

				if removeErr := os.Remove(tempFileName); removeErr != nil {
					globals.Logger.Debug().
						Err(removeErr).
						Str("temp_file", tempFileName).
						Msg("failed to remove temporary file")
				}

				cleaned = true

				globals.Logger.Debug().
					Str("temp_file", tempFileName).
					Msg("cleaned up temporary file")
			}
		})
	}

	if err := c.writeAndCloseTempFile(tempFile, content, globals); err != nil {
		cleanupTemp()

		return nil, nil, err
	}

	return tempFile, cleanupTemp, nil
}

func (c *EditCmd) writeAndCloseTempFile(tempFile *os.File, content []byte, globals *Globals) error {
	tempFileName := tempFile.Name()

	if _, writeErr := tempFile.Write(content); writeErr != nil {
		globals.Logger.Error().
			Err(writeErr).
			Str("temp_file", tempFileName).
			Msg("failed to write content to temporary file")

		return fmt.Errorf("write content to temp file: %w", writeErr)
	}

	if closeErr := tempFile.Close(); closeErr != nil {
		globals.Logger.Error().
			Err(closeErr).
			Str("temp_file", tempFileName).
			Msg("failed to close temporary file")

		return fmt.Errorf("close temp file: %w", closeErr)
	}

	return nil
}

func (c *EditCmd) setupSignalHandling(cleanupTemp func(), globals *Globals) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	signalDone := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer close(signalDone)
		select {
		case <-sigChan:
			globals.Logger.Debug().Msg("received interrupt signal, cleaning up...")
			cleanupTemp()
			cancel()
		case <-ctx.Done():
		}
	}()

	cleanup := func() {
		signal.Stop(sigChan)
		cancel()
		<-signalDone
	}

	return ctx, cleanup
}

func (c *EditCmd) getFileStats(filename string) (os.FileInfo, error) {
	beforeStat, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("stat temp file: %w", err)
	}

	return beforeStat, nil
}

func (c *EditCmd) launchEditor(ctx context.Context, tempFileName string, globals *Globals) error {
	editor := c.determineEditor(globals)
	if editor == "" {
		return fmt.Errorf("no editor specified: use --editor flag or set EDITOR environment variable")
	}

	globals.Logger.Info().
		Str("editor", editor).
		Str("file", c.File).
		Msg("launching editor")

	return c.executeEditor(ctx, editor, tempFileName, globals)
}

func (c *EditCmd) determineEditor(globals *Globals) string {
	if c.Editor != "" {
		globals.Logger.Debug().
			Str("editor", c.Editor).
			Msg("using editor from command line flag")

		return c.Editor
	}

	if env := os.Getenv("EDITOR"); env != "" {
		globals.Logger.Debug().
			Str("editor", env).
			Msg("using editor from EDITOR environment variable")

		return env
	}

	return ""
}

func (c *EditCmd) executeEditor(ctx context.Context, editor, tempFileName string, globals *Globals) error {
	execCmd := exec.CommandContext(ctx, editor, tempFileName)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if execErr := execCmd.Run(); execErr != nil {
		if ctx.Err() != nil {
			globals.Logger.Warn().Msg("editor interrupted by signal")

			return fmt.Errorf("editor interrupted")
		}

		globals.Logger.Error().
			Err(execErr).
			Str("editor", editor).
			Msg("editor execution failed")

		return fmt.Errorf("editor failed: %w", execErr)
	}

	if ctx.Err() != nil {
		globals.Logger.Warn().Msg("operation cancelled")

		return fmt.Errorf("operation cancelled")
	}

	return nil
}

func (c *EditCmd) processChanges(session *core.Session, tempFileName string, beforeStat os.FileInfo, globals *Globals) error {
	afterStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info().Msg("no changes detected")

		return nil
	}

	globals.Logger.Debug().Msg("changes detected, processing updated content")

	return c.saveChanges(session, tempFileName, globals)
}

func (c *EditCmd) saveChanges(session *core.Session, tempFileName string, globals *Globals) error {
	modified, err := os.ReadFile(tempFileName)
	defer core.WipeData(modified)

	if err != nil {
		globals.Logger.Error().
			Err(err).
			Str("temp_file", tempFileName).
			Msg("failed to read modified content")

		return fmt.Errorf("read modified content: %w", err)
	}

	updatedVariables, err := core.ParseEnv(modified)
	if err != nil {
		globals.Logger.Error().
			Err(err).
			Msg("invalid environment file format")

		return fmt.Errorf("invalid environment file format: %w", err)
	}

	if err := session.SaveVars(c.File, updatedVariables); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("file", c.File).
			Msg("failed to save changes")

		return fmt.Errorf("save changes: %w", err)
	}

	globals.Logger.Info().
		Str("file", c.File).
		Int("variables", len(updatedVariables)).
		Msg("environment file updated successfully")

	return nil
}
