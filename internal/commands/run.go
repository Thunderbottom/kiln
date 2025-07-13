package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// RunCmd represents the run command for executing programs with encrypted environment variables.
type RunCmd struct {
	File    string        `short:"f" help:"Environment file to use" default:"default"`
	DryRun  bool          `help:"Show environment variables without running command"`
	Timeout time.Duration `help:"Timeout for command execution" placeholder:"[10s]"`
	WorkDir string        `help:"Working directory for command execution" placeholder:"[path]"`
	Shell   bool          `help:"Run command through shell"`
	Command []string      `arg:"" help:"Command and arguments to run"`
}

// ExitError represents a command exit with a specific code.
type ExitError struct {
	Code int
}

// Error returns a exit-code formatted error.
func (e *ExitError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

func (c *RunCmd) validate() error {
	if len(c.Command) == 0 {
		return kerrors.ValidationError("command", "no command specified")
	}

	if err := core.IsValidCommand(c.Command); err != nil {
		return kerrors.SecurityError(err.Error(), "use simpler command arguments")
	}

	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	if c.Timeout > 0 && !core.IsValidTimeout(c.Timeout) {
		return kerrors.ValidationError("timeout", "must be between 1 second and 24 hours")
	}

	if c.WorkDir != "" {
		if err := core.IsValidWorkingDirectory(c.WorkDir); err != nil {
			return kerrors.ValidationError("working directory", err.Error())
		}
	}

	return nil
}

// Run executes the run command, loading environment variables and executing the specified command.
func (c *RunCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "run").Strs("args", c.Command).Str("file", c.File).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

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

	variables, cleanup, err := core.GetAllEnvVars(identity, cfg, c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	rt.Logger.Debug().Int("count", len(variables)).Msg("loaded environment variables")

	if c.DryRun {
		c.showDryRun(variables, rt)

		return nil
	}

	return c.executeCommand(variables, rt)
}

func (c *RunCmd) showDryRun(variables map[string][]byte, rt *Runtime) {
	rt.Logger.Info().Str("command", strings.Join(c.Command, " ")).Msg("Would execute")
	rt.Logger.Info().Str("file", c.File).Msg("Environment file")
	rt.Logger.Info().Int("count", len(variables)).Msg("Variables")

	keys := core.SortedKeys(variables)
	for _, key := range keys {
		value := string(variables[key])
		fmt.Printf("  %s=%s\n", key, value)
	}
}

// executeCommand runs the specified command with injected environment variables.
func (c *RunCmd) executeCommand(variables map[string][]byte, rt *Runtime) error {
	ctx, cancel := c.createContext(rt)
	defer cancel()

	cmd := c.buildCommand(ctx, rt)
	c.setupEnvironment(cmd, variables)
	c.configureCommand(cmd, rt)

	err := cmd.Run()
	if err != nil {
		return c.handleCommandError(err, rt)
	}

	return nil
}

// createContext creates a command execution context with signal handling and optional timeout.
func (c *RunCmd) createContext(rt *Runtime) (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	if c.Timeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, c.Timeout)
		rt.Logger.Debug().Dur("timeout", c.Timeout).Msg("command timeout configured")

		cancelAll := func() {
			timeoutCancel()
			cancel()
		}

		return timeoutCtx, cancelAll
	}

	return ctx, cancel
}

// buildCommand creates an exec.Cmd for either shell or direct execution.
func (c *RunCmd) buildCommand(ctxWithCancel context.Context, rt *Runtime) *exec.Cmd {
	var cmd *exec.Cmd

	if c.Shell {
		commandString := strings.Join(c.Command, " ")
		cmd = exec.CommandContext(ctxWithCancel, "/bin/sh", "-c", commandString)
		rt.Logger.Debug().Str("shell_command", commandString).Msg("executing through shell")
	} else {
		executable := c.Command[0]
		if strings.HasPrefix(executable, "./") || strings.HasPrefix(executable, "../") {
			if absPath, err := filepath.Abs(executable); err == nil {
				executable = absPath
				rt.Logger.Debug().Str("original", c.Command[0]).Str("resolved", executable).Msg("resolved relative path")
			}
		}

		cmd = exec.CommandContext(ctxWithCancel, executable, c.Command[1:]...)
		rt.Logger.Debug().Str("executable", executable).Strs("args", c.Command[1:]).Msg("executing directly")
	}

	return cmd
}

func (c *RunCmd) setupEnvironment(cmd *exec.Cmd, variables map[string][]byte) {
	cmd.Env = os.Environ()
	for key, value := range variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, string(value)))
	}
}

func (c *RunCmd) configureCommand(cmd *exec.Cmd, rt *Runtime) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
		rt.Logger.Debug().Str("workdir", c.WorkDir).Msg("working directory set")
	}
}

func (c *RunCmd) handleCommandError(err error, rt *Runtime) error {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			rt.Logger.Debug().Int("exit_code", status.ExitStatus()).Msg("command exited with non-zero status")

			return &ExitError{Code: status.ExitStatus()}
		}
	}

	return fmt.Errorf("command failed: %w", err)
}
