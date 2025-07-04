package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

func Edit(ctx context.Context, args []string) error {
	// Check for help flags first
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printEditUsage()
			return nil
		}
	}

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)

	file := fs.String("file", "default", "environment file to edit")
	editor := fs.String("editor", "", "editor to use (overrides $EDITOR)")
	validate := fs.Bool("validate", true, "validate environment file after editing")
	verbose := fs.Bool("v", false, "verbose output")

	fs.Usage = printEditUsage

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Handle positional argument for file
	if fs.NArg() > 0 {
		*file = fs.Arg(0)
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get environment file path
	envFilePath := cfg.GetEnvFile(*file)

	if *verbose {
		fmt.Printf("Editing environment file: %s (%s)\n", *file, envFilePath)
	}

	// Load private key
	privateKey, keyInfo, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	if *verbose {
		fmt.Printf("Loaded private key from: %s\n", keyInfo.Source)
	}

	// Setup age manager
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return fmt.Errorf("failed to add identity: %w", err)
	}

	// Read and decrypt existing file or create template
	plaintext, existed, err := readOrCreateTemplate(envFilePath, ageManager)
	if err != nil {
		return err
	}

	if !existed && *verbose {
		fmt.Printf("File doesn't exist, created template\n")
	}

	// Create secure temporary file
	tempFile, err := utils.CreateSecureTempFile("kiln-edit-*.env")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		utils.SecureDelete(tempFile)
	}()

	// Write plaintext to temp file
	if err := os.WriteFile(tempFile, plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Get file modification time before editing
	beforeStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}

	// Determine and launch editor
	editorCmd := determineEditor(*editor)
	if *verbose {
		fmt.Printf("Opening editor: %s\n", editorCmd)
	}

	if err := launchEditor(ctx, editorCmd, tempFile); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Check if file was modified
	afterStat, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat temporary file after editing: %w", err)
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		fmt.Printf("No changes detected, file not updated\n")
		return nil
	}

	// Read modified content
	modifiedContent, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read modified content: %w", err)
	}

	// Validate content if requested
	if *validate {
		if err := validateContent(modifiedContent); err != nil {
			fmt.Printf("Validation failed: %v\n", err)
			fmt.Printf("Save anyway? [y/N]: ")

			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" && response != "yes" {
				return fmt.Errorf("validation failed, changes not saved")
			}
		} else if *verbose {
			fmt.Printf("Content validation passed\n")
		}
	}

	// Encrypt and save
	encrypted, err := ageManager.Encrypt(modifiedContent)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return fmt.Errorf("failed to save environment file: %w", err)
	}

	fmt.Printf("Environment file updated: %s\n", envFilePath)

	// Show summary if verbose
	if *verbose {
		showEditSummary(plaintext, modifiedContent)
	}

	return nil
}

// printEditUsage prints the usage information for the edit command
func printEditUsage() {
	fmt.Print(`Usage: kiln edit [flags] [file]

Edit encrypted environment variables in your default editor.

Flags:
  -file string     environment file to edit (default "default")
  -editor string   editor to use (overrides $EDITOR)
  -validate        validate environment file after editing (default true)
  -v               verbose output

The editor respects the following precedence:
  1. -editor flag
  2. KILN_EDITOR environment variable
  3. EDITOR environment variable
  4. vi (fallback)

Examples:
  kiln edit
  kiln edit -file staging
  kiln edit -editor nano
  kiln edit -validate=false
`)
}

func readOrCreateTemplate(envFilePath string, ageManager *crypto.AgeManager) ([]byte, bool, error) {
	// Try to read existing file
	if _, err := os.Stat(envFilePath); err == nil {
		encrypted, err := os.ReadFile(envFilePath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read environment file: %w", err)
		}

		plaintext, err := ageManager.Decrypt(encrypted)
		if err != nil {
			return nil, false, fmt.Errorf("failed to decrypt environment file: %w", err)
		}

		return plaintext, true, nil
	}

	// Create template for new file
	template := `# Environment Variables
# Format: KEY=value
# 
# Examples:
# DATABASE_URL=postgres://user:pass@localhost/db
# API_TOKEN=your_secret_token
# DEBUG=true
# PORT=8080
#
# Tips:
# - Use UPPERCASE for variable names
# - Quote values with spaces: VAR="value with spaces"
# - No spaces around the = sign
# - Lines starting with # are comments

`

	return []byte(template), false, nil
}

func determineEditor(editorFlag string) string {
	// Check flag first
	if editorFlag != "" {
		return editorFlag
	}

	// Check KILN_EDITOR
	if editor := os.Getenv("KILN_EDITOR"); editor != "" {
		return editor
	}

	// Check EDITOR
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Fallback to vi
	return "vi"
}

func launchEditor(ctx context.Context, editor, tempFile string) error {
	// Create context with timeout
	timeout := 30 * time.Minute // Reasonable timeout for editing
	editCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Launch editor
	cmd := exec.CommandContext(editCtx, editor, tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func validateContent(content []byte) error {
	_, err := env.ParseEnvFile(string(content))
	return err
}

func showEditSummary(before, after []byte) {
	beforeVars, _ := env.ParseEnvFile(string(before))
	afterVars, _ := env.ParseEnvFile(string(after))

	beforeCount := len(beforeVars)
	afterCount := len(afterVars)

	fmt.Printf("\nEdit Summary:\n")
	fmt.Printf("  Variables before: %d\n", beforeCount)
	fmt.Printf("  Variables after:  %d\n", afterCount)

	if afterCount > beforeCount {
		fmt.Printf("  Added: %d variables\n", afterCount-beforeCount)
	} else if afterCount < beforeCount {
		fmt.Printf("  Removed: %d variables\n", beforeCount-afterCount)
	}

	// Show modified variables
	modified := 0
	for key, afterVal := range afterVars {
		if beforeVal, exists := beforeVars[key]; exists && beforeVal != afterVal {
			modified++
		}
	}

	if modified > 0 {
		fmt.Printf("  Modified: %d variables\n", modified)
	}
}
