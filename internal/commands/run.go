package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type RunCmd struct {
	File    string   `short:"f" help:"Environment file to use" default:"default"`
	DryRun  bool     `help:"Show environment variables without running command"`
	Timeout string   `help:"Timeout for command execution"`
	WorkDir string   `help:"Working directory for command execution"`
	Shell   bool     `help:"Run command through shell"`
	Expand  bool     `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
	Command []string `arg:"" help:"Command and arguments to run"`
}

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

		return nil
	}

	return c.executeCommand(variables, globals)
}

func (c *RunCmd) executeCommand(variables map[string][]byte, globals *Globals) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if c.Timeout != "" {
		duration, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, duration)
		defer timeoutCancel()

		globals.Logger.Debug().
			Str("timeout", c.Timeout).
			Msg("command timeout configured")
	}

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

	cmd.Env = os.Environ()
	for key, value := range variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, string(value)))
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
		globals.Logger.Debug().
			Str("workdir", c.WorkDir).
			Msg("working directory set")
	}

	globals.Logger.Debug().
		Str("command", strings.Join(c.Command, " ")).
		Int("env_vars", len(variables)).
		Msg("executing command")

	if c.Expand {
		globals.Logger.Debug().Msg("variable expansion applied")
	}

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				globals.Logger.Debug().
					Int("exit_code", status.ExitStatus()).
					Msg("command exited with non-zero status")
				os.Exit(status.ExitStatus())
			}
		}
		globals.Logger.Error().
			Err(err).
			Str("command", strings.Join(c.Command, " ")).
			Msg("command execution failed")
		return fmt.Errorf("command failed: %w", err)
	}

	globals.Logger.Debug().
		Str("command", strings.Join(c.Command, " ")).
		Msg("command executed successfully")

	return nil
}
