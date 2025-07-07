package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// RunCmd represents the run command for executing programs with encrypted environment variables.
type RunCmd struct {
	File    string        `short:"f" help:"Environment file to use" default:"default"`
	DryRun  bool          `help:"Show environment variables without running command"`
	Timeout time.Duration `help:"Timeout for command execution" placeholder:"[10s]"`
	WorkDir string        `help:"Working directory for command execution" placeholder:"[path]"`
	Shell   bool          `help:"Run command through shell"`
	Expand  bool          `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
	Command []string      `arg:"" help:"Command and arguments to run"`
}

// ExitError represents a command exit with a specific code.
type ExitError struct {
	Code int
}

// Error returns a custom error message with an exit code.
func (e *ExitError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

// Run executes the run command, loading environment variables and executing the specified command.
func (c *RunCmd) Run(globals *Globals) error {
	if len(c.Command) == 0 {
		return fmt.Errorf("no command specified")
	}

	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	globals.Logger.Debug().
		Str("file", c.File).
		Str("command", strings.Join(c.Command, " ")).
		Bool("expand", c.Expand).
		Bool("dry_run", c.DryRun).
		Msg("preparing to run command with environment")

	variables, cleanup, err := session.ExportVars(c.File, c.Expand)
	if err != nil {
		globals.Logger.Error().
			Err(err).
			Str("file", c.File).
			Msg("failed to load environment variables")

		return err
	}
	defer cleanup()

	globals.Logger.Debug().
		Int("variable_count", len(variables)).
		Msg("environment variables loaded")

	if c.DryRun {
		c.showDryRun(variables, globals)

		return nil
	}

	exitErr := c.executeCommand(variables, globals)
	if exitErr != nil {
		var exitError *ExitError
		if errors.As(exitErr, &exitError) {
			// Let deferred functions run, then exit
			defer func() {
				os.Exit(exitError.Code)
			}()

			return nil
		}

		return exitErr
	}

	return nil
}

func (c *RunCmd) showDryRun(variables map[string][]byte, globals *Globals) {
	globals.Logger.Info().
		Str("command", strings.Join(c.Command, " ")).
		Int("variables", len(variables)).
		Msg("dry run mode enabled")

	for key, value := range variables {
		displayValue := string(value)
		if len(displayValue) > 50 {
			displayValue = displayValue[:47] + "..."
		}

		globals.Logger.Info().Str("variable", key).Str("value", displayValue).Msg("environment variable")
	}
}

func (c *RunCmd) executeCommand(variables map[string][]byte, globals *Globals) error {
	ctx := c.createContext(globals)
	cmd := c.buildCommand(ctx, globals)
	c.setupEnvironment(cmd, variables)
	c.configureCommand(cmd, globals)

	return c.runCommand(cmd, globals)
}

// createContext sets up a cancellable context with optional timeout and signal handling for command execution.
func (c *RunCmd) createContext(globals *Globals) context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	_ = cancel // Will be called when context is done

	if c.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, c.Timeout)
		_ = timeoutCancel

		globals.Logger.Debug().
			Dur("timeout", c.Timeout).
			Msg("command timeout configured")
	}

	return ctx
}

// buildCommand creates an exec.Cmd for the specified command, optionally wrapping it in a shell.
func (c *RunCmd) buildCommand(ctx context.Context, globals *Globals) *exec.Cmd {
	var cmd *exec.Cmd

	if c.Shell {
		commandString := strings.Join(c.Command, " ")
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", commandString)
		globals.Logger.Debug().
			Str("shell_command", commandString).
			Msg("executing command through shell")
	} else {
		cmd = exec.CommandContext(ctx, c.Command[0], c.Command[1:]...)
		globals.Logger.Debug().
			Str("executable", c.Command[0]).
			Strs("args", c.Command[1:]).
			Msg("executing command directly")
	}

	return cmd
}

// setupEnvironment configures the command's environment variables by combining system env with decrypted variables.
func (c *RunCmd) setupEnvironment(cmd *exec.Cmd, variables map[string][]byte) {
	cmd.Env = os.Environ()
	for key, value := range variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, string(value)))
	}
}

func (c *RunCmd) configureCommand(cmd *exec.Cmd, globals *Globals) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
		globals.Logger.Debug().
			Str("workdir", c.WorkDir).
			Msg("working directory set")
	}
}

func (c *RunCmd) runCommand(cmd *exec.Cmd, globals *Globals) error {
	globals.Logger.Debug().
		Str("command", strings.Join(c.Command, " ")).
		Msg("executing command")

	if c.Expand {
		globals.Logger.Debug().Msg("variable expansion applied")
	}

	err := cmd.Run()
	if err != nil {
		return c.handleCommandError(err, globals)
	}

	globals.Logger.Debug().
		Str("command", strings.Join(c.Command, " ")).
		Msg("command executed successfully")

	return nil
}

// handleCommandError processes command execution errors and converts exit codes to appropriate error types.
func (c *RunCmd) handleCommandError(err error, globals *Globals) error {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			globals.Logger.Debug().
				Int("exit_code", status.ExitStatus()).
				Msg("command exited with non-zero status")

			// NOTE: calling os.Exit() directly causes defer
			// to fail, so we return the error instead
			return &ExitError{Code: status.ExitStatus()}
		}
	}

	globals.Logger.Error().
		Err(err).
		Str("command", strings.Join(c.Command, " ")).
		Msg("command execution failed")

	return fmt.Errorf("command failed: %w", err)
}
