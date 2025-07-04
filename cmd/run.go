package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/errors"
	"github.com/thunderbottom/kiln/internal/utils"
)

var (
	runFile    string
	runDryRun  bool
	runTimeout string
	runWorkDir string
)

var runCmd = &cobra.Command{
	Use:   "run [flags] -- <command> [args...]",
	Short: "Run command with encrypted environment variables",
	Long:  GetLongDescription("run", RunDescription),
	RunE:  runRun,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("requires a command to run\n\nUsage: kiln run -- <command> [args...]")
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete common commands
			commands := []string{
				"env", "terraform", "kubectl", "docker", "npm", "yarn", "go", "python", "node",
			}
			return commands, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveDefault
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runFile, "file", "",
		"environment file to use (default: from config)")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false,
		"show environment variables without running command")
	runCmd.Flags().StringVar(&runTimeout, "timeout", "",
		"timeout for command execution (e.g., 30s, 5m, 1h)")
	runCmd.Flags().StringVar(&runWorkDir, "workdir", "",
		"working directory for command execution")

	// Register flag completions
	runCmd.RegisterFlagCompletionFunc("file", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for name := range cfg.ListEnvFiles() {
			completions = append(completions, name)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return errors.Wrap(err, "failed to load configuration")
	}

	// Determine which file to use
	envFile := runFile
	if envFile == "" {
		envFile = "default"
	}

	envFilePath := cfg.GetEnvFile(envFile)

	// Check if file exists
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFilePath)
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Loading environment from: %s (%s)\n", envFile, envFilePath)
	}

	// Load private key
	privateKey, keyInfo, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load private key")
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Using private key from: %s\n", keyInfo.Source)
	}

	// Setup age manager
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return errors.Wrap(err, "failed to setup encryption")
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return errors.Wrap(err, "failed to add identity")
	}

	// Read and decrypt file
	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read environment file")
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt environment file")
	}

	// Parse environment variables
	envVars, err := env.ParseEnvFile(string(plaintext))
	if err != nil {
		return errors.Wrap(err, "failed to parse environment file")
	}

	if len(envVars) == 0 {
		if IsVerbose() {
			fmt.Fprintf(os.Stderr, "No environment variables found\n")
		}
	}

	// Handle dry run
	if runDryRun {
		return handleDryRun(envVars, args)
	}

	// Prepare command execution
	return executeCommand(ctx, envVars, args)
}

func handleDryRun(envVars map[string]string, args []string) error {
	fmt.Printf("🔍 Dry run mode - showing what would be executed\n\n")

	if len(envVars) > 0 {
		fmt.Printf("Environment variables that would be injected:\n\n")

		// Sort keys for consistent output
		keys := make([]string, 0, len(envVars))
		for key := range envVars {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		maxKeyLen := 0
		for _, key := range keys {
			if len(key) > maxKeyLen {
				maxKeyLen = len(key)
			}
		}

		for _, key := range keys {
			value := envVars[key]
			displayValue := value

			// Mask sensitive-looking values for display
			if env.IsSensitiveKey(key) {
				displayValue = env.MaskSensitiveValue(key, value)
			}

			// Truncate very long values
			if len(displayValue) > 60 {
				displayValue = displayValue[:57] + "..."
			}

			fmt.Printf("  %-*s = %s\n", maxKeyLen, key, displayValue)
		}

		fmt.Printf("\nTotal: %d variables\n", len(envVars))
	} else {
		fmt.Printf("No environment variables to inject\n")
	}

	fmt.Printf("\nCommand that would be executed:\n")
	fmt.Printf("  %s\n", strings.Join(args, " "))

	if runWorkDir != "" {
		fmt.Printf("  Working directory: %s\n", runWorkDir)
	}

	if runTimeout != "" {
		fmt.Printf("  Timeout: %s\n", runTimeout)
	}

	return nil
}

func executeCommand(ctx context.Context, envVars map[string]string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	binary := args[0]
	cmdArgs := args[1:]

	// Look for the binary in PATH
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("command not found: %s", binary)
	}

	// Prepare environment
	cmdEnv := os.Environ()
	for key, value := range envVars {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Handle timeout if specified
	if runTimeout != "" {
		timeout, err := parseTimeout(runTimeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Executing: %s\n", strings.Join(args, " "))
		fmt.Fprintf(os.Stderr, "With %d environment variables\n", len(envVars))
		if runWorkDir != "" {
			fmt.Fprintf(os.Stderr, "Working directory: %s\n", runWorkDir)
		}
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, binaryPath, cmdArgs...)
	cmd.Env = cmdEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory if specified
	if runWorkDir != "" {
		if err := utils.ValidateFilePath(runWorkDir); err != nil {
			return fmt.Errorf("invalid working directory: %v", err)
		}
		cmd.Dir = runWorkDir
	}

	// Execute command
	err = cmd.Run()
	if err != nil {
		// Extract exit code if possible
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

func parseTimeout(timeoutStr string) (time.Duration, error) {
	// Handle common timeout formats
	if strings.HasSuffix(timeoutStr, "s") ||
		strings.HasSuffix(timeoutStr, "m") ||
		strings.HasSuffix(timeoutStr, "h") {
		return time.ParseDuration(timeoutStr)
	}

	// If no unit specified, assume seconds
	return time.ParseDuration(timeoutStr + "s")
}
