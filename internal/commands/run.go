package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

func Run(ctx context.Context, args []string) error {
	// Check for help flags first, before requiring -- separator
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printRunUsage()
			return nil
		}
	}

	// Find the -- separator to split flags from command
	dashIndex := findDashSeparator(args)
	if dashIndex == -1 {
		return fmt.Errorf("missing '--' separator before command\n\nUsage: kiln run [flags] -- <command> [args...]")
	}

	flagArgs := args[:dashIndex]
	cmdArgs := args[dashIndex+1:]

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified after '--'")
	}

	// Parse flags
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	file := fs.String("file", "default", "environment file to use")
	dryRun := fs.Bool("dry-run", false, "show environment variables without running command")
	timeout := fs.String("timeout", "", "timeout for command execution (e.g., 30s, 5m, 1h)")
	workDir := fs.String("workdir", "", "working directory for command execution")
	shell := fs.Bool("shell", false, "run command through shell (enables variable expansion)")
	verbose := fs.Bool("v", false, "verbose output")

	// Set custom usage function
	fs.Usage = printRunUsage

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get environment file path
	envFilePath := cfg.GetEnvFile(*file)
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFilePath)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Loading environment from: %s (%s)\n", *file, envFilePath)
	}

	// Load and decrypt environment variables
	envVars, err := loadAndDecryptEnv(ctx, cfg, envFilePath, *verbose)
	if err != nil {
		return err
	}

	if *dryRun {
		return showDryRun(cfg, envVars, cmdArgs, *shell)
	}

	// Execute command with environment
	return executeCommand(ctx, envVars, cmdArgs, *timeout, *workDir, *shell, *verbose)
}

// printRunUsage prints the usage information for the run command
func printRunUsage() {
	fmt.Print(`Usage: kiln run [flags] -- <command> [args...]

Run command with encrypted environment variables loaded.

Flags:
  -file string      environment file to use (default "default")
  -dry-run          show environment variables without running command
  -timeout string   timeout for command execution (e.g., 30s, 5m, 1h)
  -workdir string   working directory for command execution
  -shell            run command through shell (enables variable expansion)
  -v                verbose output

Examples:
  kiln run -- env
  kiln run -- echo "$SECRET_KEY"                    # May not expand
  kiln run -shell -- echo "$SECRET_KEY"             # Will expand variables
  kiln run -file staging -- terraform plan
  kiln run -dry-run -- env
  kiln run -timeout 5m -- long-running-command
`)
}

func findDashSeparator(args []string) int {
	for i, arg := range args {
		if arg == "--" {
			return i
		}
	}
	return -1
}

func loadAndDecryptEnv(ctx context.Context, cfg *config.Config, envFilePath string, verbose bool) (map[string]string, error) {
	// Load private key
	privateKey, keyInfo, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Using private key from: %s\n", keyInfo.Source)
	}

	// Setup age manager
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return nil, fmt.Errorf("failed to add identity: %w", err)
	}

	// Read and decrypt file
	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt environment file: %w", err)
	}

	// Parse environment variables
	envVars, err := env.ParseEnvFile(string(plaintext))
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment file: %w", err)
	}

	return envVars, nil
}

func showDryRun(cfg *config.Config, envVars map[string]string, cmdArgs []string, shell bool) error {
	fmt.Printf("Dry run mode - showing what would be executed\n\n")

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
			if cfg.IsSensitiveKey(key) {
				displayValue = maskSensitiveValue(key, value)
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
	if shell {
		fmt.Printf("  Shell: %s\n", strings.Join(cmdArgs, " "))
	} else {
		fmt.Printf("  Direct: %s\n", strings.Join(cmdArgs, " "))
	}

	return nil
}

func executeCommand(ctx context.Context, envVars map[string]string, cmdArgs []string, timeoutStr, workDir string, shell, verbose bool) error {
	// Handle timeout if specified
	if timeoutStr != "" {
		timeout, err := parseTimeout(timeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var cmd *exec.Cmd
	var cmdDescription string

	if shell {
		// Run through shell for variable expansion
		var shellCmd, shellFlag string
		if isWindows() {
			shellCmd = "cmd"
			shellFlag = "/C"
		} else {
			shellCmd = "/bin/sh"
			shellFlag = "-c"
		}

		// Join all arguments into a single command string
		commandString := strings.Join(cmdArgs, " ")
		cmd = exec.CommandContext(ctx, shellCmd, shellFlag, commandString)
		cmdDescription = fmt.Sprintf("%s %s '%s'", shellCmd, shellFlag, commandString)
	} else {
		// Direct execution
		binary := cmdArgs[0]
		args := cmdArgs[1:]

		// Look for the binary in PATH
		binaryPath, err := exec.LookPath(binary)
		if err != nil {
			return fmt.Errorf("command not found: %s", binary)
		}

		cmd = exec.CommandContext(ctx, binaryPath, args...)
		cmdDescription = strings.Join(cmdArgs, " ")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Executing: %s\n", cmdDescription)
		fmt.Fprintf(os.Stderr, "With %d environment variables\n", len(envVars))
		if workDir != "" {
			fmt.Fprintf(os.Stderr, "Working directory: %s\n", workDir)
		}
		if shell {
			fmt.Fprintf(os.Stderr, "Using shell execution for variable expansion\n")
		}
	}

	// CRITICAL FIX: Properly inherit current environment and add kiln variables
	cmd.Env = os.Environ() // Start with current environment
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory if specified
	if workDir != "" {
		if err := utils.ValidateFilePath(workDir); err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}
		cmd.Dir = workDir
	}

	// Execute command and preserve exit codes
	err := cmd.Run()
	if err != nil {
		// Extract and preserve the original exit code
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

func maskSensitiveValue(key, value string) string {
	if len(value) == 0 {
		return ""
	}

	if len(value) <= 8 {
		return "****"
	}

	// Show first 2 and last 2 characters
	return value[:2] + "****" + value[len(value)-2:]
}

// isWindows returns true if running on Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}
