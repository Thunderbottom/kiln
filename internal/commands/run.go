package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/env"
)

type RunCmd struct {
	File          string   `short:"f" help:"Environment file to use" default:"default"`
	DryRun        bool     `help:"Show environment variables without running command"`
	Timeout       string   `help:"Timeout for command execution"`
	WorkDir       string   `help:"Working directory for command execution"`
	Shell         bool     `help:"Run command through shell"`
	Expand        bool     `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
	AllowCommands bool     `help:"Allow command substitution ($$(command) syntax)"`
	Command       []string `arg:"" help:"Command and arguments to run"`
}

func (c *RunCmd) Run(globals *Globals) error {
	if len(c.Command) == 0 {
		return fmt.Errorf("no command specified")
	}

	ctx := globals.Context()
	envVars, err := core.LoadEnvVars(ctx, globals.Config, c.File)
	if err != nil {
		return err
	}

	// Apply variable expansion if enabled
	if c.Expand {
		globals.Logger.Debug("applying variable expansion")
		if c.AllowCommands {
			globals.Logger.Debug("command substitution enabled")
		}
		envVars = env.ExpandVariables(envVars, c.AllowCommands)
	}

	if c.DryRun {
		globals.Logger.Info("dry run mode enabled", "cmd", strings.Join(c.Command, " "),
			"variables", len(envVars))

		for key, value := range envVars {
			displayValue := value
			if len(displayValue) > 50 {
				displayValue = displayValue[:47] + "..."
			}
			globals.Logger.Info("var", "key", key, "value", displayValue)
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

	// Set up process group to properly kill child processes on Unix-like systems
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
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

	globals.Logger.Debug("executing command", "cmd", strings.Join(c.Command, " "))
	if c.Expand {
		globals.Logger.Debug("variable expansion applied", "count", len(envVars))
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command failed to start: %w", err)
	}

	// Create a channel to receive the Wait() result
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	// Wait for either completion or timeout
	select {
	case err := <-waitCh:
		// Process completed normally
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			}
			return fmt.Errorf("command failed: %w", err)
		}
		return nil

	case <-ctx.Done():
		// Context was cancelled (timeout)
		if runtime.GOOS != "windows" {
			// Kill the entire process group
			if cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		} else {
			// On Windows, just kill the main process
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}

		// Wait for the goroutine to finish
		<-waitCh

		return fmt.Errorf("command timed out")
	}
}
