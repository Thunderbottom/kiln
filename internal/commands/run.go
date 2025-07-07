package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	Expand  bool     `help:"Enable variable expansion (${VAR} syntax)" default:"false"`
	Command []string `arg:"" help:"Command and arguments to run"`
}

func (c *RunCmd) Run(globals *Globals) error {
	if len(c.Command) == 0 {
		return fmt.Errorf("no command specified")
	}

	sess, err := globals.Session()
	if err != nil {
		return err
	}

	ctx := globals.Context()
	envVars, err := sess.ExportVars(ctx, c.File, c.Expand)
	if err != nil {
		return err
	}

	if c.DryRun {
		globals.Logger.Info().
			Str("cmd", strings.Join(c.Command, " ")).
			Int("variables", len(envVars)).
			Msg("dry run mode enabled")

		for key, value := range envVars {
			displayValue := value
			if len(displayValue) > 50 {
				displayValue = displayValue[:47] + "..."
			}
			globals.Logger.Info().Str("key", key).Str("value", displayValue)
		}

		return nil
	}

	return c.executeCommand(envVars, globals)
}

func (c *RunCmd) executeCommand(envVars map[string]string, globals *Globals) error {
	ctx := context.Background()
	if c.Timeout != "" {
		duration, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	var cmd *exec.Cmd
	if c.Shell {
		cmdStr := strings.Join(c.Command, " ")
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, c.Command[0], c.Command[1:]...)
	}

	cmd.Env = os.Environ()
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}

	globals.Logger.Debug().Str("cmd", strings.Join(c.Command, " ")).Msg("executing command")
	if c.Expand {
		globals.Logger.Debug().Msg("variable expansion applied")
	}

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}
