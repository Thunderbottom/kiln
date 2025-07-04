package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// EditCmd represents the edit command for modifying encrypted environment variables.
type EditCmd struct {
	File   string `short:"f" help:"Environment file to edit" default:"default"`
	Editor string `help:"Editor to use for editing the file, defaults to the EDITOR environment variable" placeholder:"EDITOR"`
}

func (c *EditCmd) validate() error {
	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	return nil
}

// Run executes the edit command, opening an editor to modify environment variables.
func (c *EditCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "edit").Str("file", c.File).Msg("validation started")

	if err := c.validate(); err != nil {
		return err
	}

	identity, err := rt.Identity()
	if err != nil {
		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	content, err := c.prepareContent(identity, cfg)
	if err != nil {
		return err
	}

	tempFile, cleanupTemp, err := c.createTempFile(content)
	if err != nil {
		return err
	}
	defer cleanupTemp()

	beforeStat, err := c.getFileStats(tempFile.Name())
	if err != nil {
		return err
	}

	editor, err := c.determineEditor()
	if err != nil {
		return err
	}

	rt.Logger.Debug().Str("editor", editor).Msg("launching editor")

	context, cancel := c.setupSignalHandling(cleanupTemp)
	defer cancel()

	if err := c.executeEditor(context, editor, tempFile.Name()); err != nil {
		return err
	}

	return c.processChanges(identity, cfg, tempFile.Name(), beforeStat, rt)
}

func (c *EditCmd) prepareContent(identity *core.Identity, cfg *config.Config) ([]byte, error) {
	vars, cleanup, err := core.GetAllEnvVars(identity, cfg, c.File)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if len(vars) > 0 {
		return core.FormatEnv(vars), nil
	}

	return []byte("# Environment Variables\n# Format: KEY=value\n"), nil
}

func (c *EditCmd) createTempFile(content []byte) (*os.File, func(), error) {
	var tmpDir string

	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/dev/shm"); err == nil {
			tmpDir = "/dev/shm"
		}
	}

	tempFile, err := os.CreateTemp(tmpDir, "kiln-edit-*.env")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp file: %w", err)
	}

	tempFileName := tempFile.Name()

	var cleanupOnce sync.Once

	cleanupTemp := func() {
		cleanupOnce.Do(func() {
			_ = tempFile.Close()

			if removeErr := os.Remove(tempFileName); removeErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to remove temp file %s: %v\n", tempFileName, removeErr)
			}
		})
	}

	if err := c.writeAndCloseTempFile(tempFile, content); err != nil {
		cleanupTemp()

		return nil, nil, err
	}

	return tempFile, cleanupTemp, nil
}

func (c *EditCmd) writeAndCloseTempFile(tempFile *os.File, content []byte) error {
	if _, writeErr := tempFile.Write(content); writeErr != nil {
		return fmt.Errorf("write content to temp file: %w", writeErr)
	}

	if closeErr := tempFile.Close(); closeErr != nil {
		return fmt.Errorf("close temp file: %w", closeErr)
	}

	return nil
}

func (c *EditCmd) setupSignalHandling(cleanupTemp func()) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	signalDone := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer close(signalDone)
		select {
		case <-sigChan:
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

func (c *EditCmd) determineEditor() (string, error) {
	editor := c.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	if editor == "" {
		return "", kerrors.ConfigError("no editor specified", "set EDITOR environment variable or use --editor flag")
	}

	if err := core.IsValidEditor(editor); err != nil {
		return "", kerrors.ConfigError(err.Error(), "check editor installation and PATH")
	}

	return editor, nil
}

func (c *EditCmd) executeEditor(ctx context.Context, editor, tempFileName string) error {
	execCmd := exec.CommandContext(ctx, editor, tempFileName)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if execErr := execCmd.Run(); execErr != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("editor interrupted")
		}

		return fmt.Errorf("editor failed: %w", execErr)
	}

	if ctx.Err() != nil {
		return fmt.Errorf("operation cancelled")
	}

	return nil
}

func (c *EditCmd) processChanges(identity *core.Identity, cfg *config.Config, tempFileName string, beforeStat os.FileInfo, rt *Runtime) error {
	afterStat, err := os.Stat(tempFileName)
	if err != nil {
		return fmt.Errorf("stat temp file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		rt.Logger.Info().Msg("No changes detected")

		return nil
	}

	return c.saveChanges(identity, cfg, tempFileName, rt)
}

func (c *EditCmd) saveChanges(identity *core.Identity, cfg *config.Config, tempFileName string, rt *Runtime) error {
	modified, err := os.ReadFile(tempFileName)
	defer core.WipeData(modified)

	if err != nil {
		return fmt.Errorf("read modified content: %w", err)
	}

	updatedVariables, err := core.ParseEnv(modified)
	if err != nil {
		return fmt.Errorf("invalid environment file format: %w", err)
	}

	for varName := range updatedVariables {
		if !core.IsValidVarName(varName) {
			return kerrors.ValidationError("variable name",
				fmt.Sprintf("'%s' must start with letter or underscore, followed by letters, numbers, or underscores", varName))
		}
	}

	if err := core.SaveAllEnvVars(identity, cfg, c.File, updatedVariables); err != nil {
		return fmt.Errorf("save changes: %w", err)
	}

	rt.Logger.Info().Str("file", c.File).Int("variables", len(updatedVariables)).Msg("environment file updated successfully")

	return nil
}
