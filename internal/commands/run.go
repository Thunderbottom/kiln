package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

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

	envVars, err := loadEnvVars(globals, c.File)
	if err != nil {
		return err
	}

	// Apply variable expansion if enabled
	if c.Expand {
		if globals.Verbose {
			fmt.Printf("Applying variable expansion\n")
		}

		envVars = env.ExpandVariables(envVars)
	}

	if c.DryRun {
		return c.showDryRun(envVars)
	}

	return c.executeCommand(envVars, globals)
}

func (c *RunCmd) showDryRun(envVars map[string]string) error {
	fmt.Printf("Dry run mode - would execute: %s\n", strings.Join(c.Command, " "))

	if c.Expand {
		fmt.Printf("Variable expansion: enabled\n")
		if c.AllowCommands {
			fmt.Printf("Command substitution: enabled\n")
		}
	}

	fmt.Printf("Environment variables (%d):\n", len(envVars))

	for key, value := range envVars {
		displayValue := value
		if len(displayValue) > 50 {
			displayValue = displayValue[:47] + "..."
		}
		fmt.Printf("  %s=%s\n", key, displayValue)
	}

	return nil
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

	if globals.Verbose {
		fmt.Printf("Executing: %s\n", strings.Join(c.Command, " "))
		if c.Expand {
			fmt.Printf("Variable expansion applied to %d variables\n", len(envVars))
		}
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
