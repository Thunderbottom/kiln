package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/errors"
	"github.com/thunderbottom/kiln/internal/utils"
)

var (
	editFile     string
	editEditor   string
	editValidate bool
)

var editCmd = &cobra.Command{
	Use:   "edit [file]",
	Short: "Edit encrypted environment variables",
	Long:  GetLongDescription("edit", EditDescription),
	RunE:  runEdit,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for name := range cfg.ListEnvFiles() {
			completions = append(completions, name)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&editFile, "file", "",
		"environment file to edit (default: from config)")
	editCmd.Flags().StringVar(&editEditor, "editor", "",
		"editor to use (overrides $EDITOR)")
	editCmd.Flags().BoolVar(&editValidate, "validate", true,
		"validate environment file after editing")

	// Add bash completion
	editCmd.RegisterFlagCompletionFunc("file", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

func runEdit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return errors.Wrap(err, "failed to load configuration")
	}

	// Determine which file to edit
	envFile := editFile
	if envFile == "" {
		if len(args) > 0 {
			envFile = args[0]
		} else {
			envFile = "default"
		}
	}

	envFilePath := cfg.GetEnvFile(envFile)

	if IsVerbose() {
		fmt.Printf("Editing environment file: %s (%s)\n", envFile, envFilePath)
	}

	// Load private key
	privateKey, keyInfo, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load private key")
	}

	if IsVerbose() {
		fmt.Printf("Loaded private key from: %s\n", keyInfo.Source)
	}

	// Setup age manager
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return errors.Wrap(err, "failed to setup encryption")
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return errors.Wrap(err, "failed to add identity")
	}

	// Read and decrypt existing file or create template
	plaintext, existed, err := readOrCreateTemplate(envFilePath, ageManager)
	if err != nil {
		return err
	}

	if !existed && IsVerbose() {
		fmt.Printf("File doesn't exist, created template\n")
	}

	// Create secure temporary file
	tempFile, err := utils.CreateSecureTempFile("kiln-edit-*.env")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary file")
	}
	defer func() {
		utils.SecureDelete(tempFile)
	}()

	// Write plaintext to temp file
	if err := os.WriteFile(tempFile, plaintext, 0600); err != nil {
		return errors.Wrap(err, "failed to write to temporary file")
	}

	// Get file modification time before editing
	beforeStat, err := os.Stat(tempFile)
	if err != nil {
		return errors.Wrap(err, "failed to stat temporary file")
	}

	// Determine and launch editor
	editor := determineEditor()
	if IsVerbose() {
		fmt.Printf("Opening editor: %s\n", editor)
	}

	if err := launchEditor(ctx, editor, tempFile); err != nil {
		return errors.Wrap(err, "editor failed")
	}

	// Check if file was modified
	afterStat, err := os.Stat(tempFile)
	if err != nil {
		return errors.Wrap(err, "failed to stat temporary file after editing")
	}

	if !afterStat.ModTime().After(beforeStat.ModTime()) {
		fmt.Printf("No changes detected, file not updated\n")
		return nil
	}

	// Read modified content
	modifiedContent, err := os.ReadFile(tempFile)
	if err != nil {
		return errors.Wrap(err, "failed to read modified content")
	}

	// Validate content if requested
	if editValidate {
		if err := validateContent(modifiedContent); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			fmt.Printf("Save anyway? [y/N]: ")

			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" && response != "yes" {
				return fmt.Errorf("validation failed, changes not saved")
			}
		} else if IsVerbose() {
			fmt.Printf("✅ Content validation passed\n")
		}
	}

	// Encrypt and save
	encrypted, err := ageManager.Encrypt(modifiedContent)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt content")
	}

	if err := utils.SaveFile(envFilePath, encrypted); err != nil {
		return errors.Wrap(err, "failed to save environment file")
	}

	fmt.Printf("✅ Environment file updated: %s\n", envFilePath)

	// Show summary if verbose
	if IsVerbose() {
		showEditSummary(plaintext, modifiedContent)
	}

	return nil
}

func readOrCreateTemplate(envFilePath string, ageManager *crypto.AgeManager) ([]byte, bool, error) {
	// Try to read existing file
	if _, err := os.Stat(envFilePath); err == nil {
		encrypted, err := os.ReadFile(envFilePath)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to read environment file")
		}

		plaintext, err := ageManager.Decrypt(encrypted)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to decrypt environment file")
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

func determineEditor() string {
	// Check flag first
	if editEditor != "" {
		return editEditor
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
	result := env.ParseEnvFileDetailed(string(content))

	// Report errors
	if len(result.Errors) > 0 {
		fmt.Printf("\nValidation errors found:\n")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", err.String())
		}
		return fmt.Errorf("found %d validation error(s)", len(result.Errors))
	}

	// Report warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
	}

	return nil
}

func showEditSummary(before, after []byte) {
	beforeResult := env.ParseEnvFileDetailed(string(before))
	afterResult := env.ParseEnvFileDetailed(string(after))

	beforeCount := len(beforeResult.Variables)
	afterCount := len(afterResult.Variables)

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
	for key, afterVal := range afterResult.Variables {
		if beforeVal, exists := beforeResult.Variables[key]; exists && beforeVal != afterVal {
			modified++
		}
	}

	if modified > 0 {
		fmt.Printf("  Modified: %d variables\n", modified)
	}
}
