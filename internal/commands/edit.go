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
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	vars, cleanup, err := session.LoadVars(c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	stringVars := make(map[string]string)
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	var content []byte
	if len(stringVars) > 0 {
		byteVars := make(map[string][]byte)
		for k, v := range stringVars {
			byteVars[k] = []byte(v)
		}
		content = core.FormatEnv(byteVars)
	} else {
		content = []byte("# Environment Variables\n# Format: KEY=value\n")
	}

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
				globals.Logger.Debug().
					Str("temp_file", tempFileName).
					Msg("cleaned up temporary file")
			}
		})
	}
	defer cleanupTemp()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	defer func() {
		signal.Stop(sigChan)
		cancel()
		<-signalDone
	}()

	if _, err := tempFile.Write(content); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("temp_file", tempFileName).
			Msg("failed to write content to temporary file")
		return fmt.Errorf("write content to temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("temp_file", tempFileName).
			Msg("failed to close temporary file")
		return fmt.Errorf("close temp file: %w", err)
	}

	beforeStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}

	editor := c.Editor
	if editor == "" {
		if env := os.Getenv("EDITOR"); env != "" {
			editor = env
			globals.Logger.Debug().
				Str("editor", editor).
				Msg("using editor from EDITOR environment variable")
		} else {
			return fmt.Errorf("no editor specified: use --editor flag or set EDITOR environment variable")
		}
	} else {
		globals.Logger.Debug().
			Str("editor", editor).
			Msg("using editor from command line flag")
	}

	globals.Logger.Info().
		Str("editor", editor).
		Str("file", c.File).
		Msg("launching editor")

	execCmd := exec.CommandContext(ctx, editor, tempFileName)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		if ctx.Err() != nil {
			globals.Logger.Warn().Msg("editor interrupted by signal")
			return fmt.Errorf("editor interrupted")
		}
		globals.Logger.Error().
			Err(err).
			Str("editor", editor).
			Msg("editor execution failed")
		return fmt.Errorf("editor failed: %w", err)
	}

	if ctx.Err() != nil {
		globals.Logger.Warn().Msg("operation cancelled")
		return fmt.Errorf("operation cancelled")
	}

	afterStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		globals.Logger.Info().Msg("no changes detected")
		return nil
	}

	globals.Logger.Debug().Msg("changes detected, processing updated content")

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
