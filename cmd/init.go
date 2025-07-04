package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/errors"
	"github.com/thunderbottom/kiln/internal/utils"
)

var (
	fromPublicKey  string
	initConfigPath string
	keyOutput      string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new kiln project",
	Long:  GetLongDescription("init", InitDescription),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&fromPublicKey, "from", "",
		"use existing public key instead of generating new key pair")
	initCmd.Flags().StringVar(&initConfigPath, "config", "",
		"custom config file path (default: .kiln.yaml)")
	initCmd.Flags().StringVar(&keyOutput, "key-output", "",
		"directory to save private key (default: current directory)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine config file path
	configFile := initConfigPath
	if configFile == "" {
		configFile = config.DefaultConfigFile
	}

	// Check if project already exists
	if config.Exists(configFile) {
		return fmt.Errorf("kiln project already exists at %s", configFile)
	}

	// Create new configuration
	cfg := config.NewConfig()

	if fromPublicKey != "" {
		return initExistingKey(cfg, configFile, fromPublicKey)
	}

	return initNewKeyPair(cmd.Context(), cfg, configFile)
}

func initExistingKey(cfg *config.Config, configFile, publicKey string) error {
	// Validate the provided public key
	if err := crypto.ValidatePublicKey(publicKey); err != nil {
		return errors.Wrap(err, "invalid public key")
	}

	cfg.AddRecipient(publicKey)

	// Save configuration
	if err := cfg.Save(configFile); err != nil {
		return errors.Wrap(err, "failed to save configuration")
	}

	// Create empty encrypted environment file
	if err := createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil {
		if IsVerbose() {
			fmt.Printf("Warning: failed to create empty env file: %v\n", err)
		}
	}

	// Success message
	fmt.Printf("✅ Initialized kiln project with existing public key\n")
	if IsVerbose() {
		fmt.Printf("   Public key: %s\n", publicKey)
		fmt.Printf("   Config file: %s\n", configFile)
	}

	fmt.Printf("\n💡 Make sure you have access to the corresponding private key!\n")
	fmt.Printf("\nNext steps:\n")
	printUsageInstructions()

	return nil
}

func initNewKeyPair(ctx context.Context, cfg *config.Config, configFile string) error {
	// Generate new key pair
	fmt.Printf("🔐 Generating new Age key pair...\n")

	privateKey, publicKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return errors.Wrap(err, "failed to generate key pair")
	}

	cfg.AddRecipient(publicKey)

	// Determine where to save the private key
	keyDir := keyOutput
	if keyDir == "" {
		keyDir = "."
	}

	// Expand path and ensure directory exists
	keyDir = utils.ExpandPath(keyDir)
	if err := utils.EnsureDirectoryExists(keyDir); err != nil {
		return errors.Wrap(err, "failed to create key directory")
	}

	// Save private key to file
	privateKeyFile := filepath.Join(keyDir, "kiln.key")
	if err := utils.SavePrivateKey(privateKey, privateKeyFile); err != nil {
		return errors.Wrap(err, "failed to save private key")
	}

	// Save configuration
	if err := cfg.Save(configFile); err != nil {
		return errors.Wrap(err, "failed to save configuration")
	}

	// Create empty encrypted environment file
	if err := createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil {
		if IsVerbose() {
			fmt.Printf("Warning: failed to create empty env file: %v\n", err)
		}
	}

	// Success message
	fmt.Printf("✅ Generated new Age key pair\n")
	if IsVerbose() {
		fmt.Printf("   Public key: %s\n", publicKey)
		fmt.Printf("   Config file: %s\n", configFile)
		fmt.Printf("   Private key file: %s\n", privateKeyFile)
	}

	fmt.Printf("\n🔐 IMPORTANT: Secure your private key!\n")
	fmt.Printf("Private key saved to: %s (permissions: 0600)\n", privateKeyFile)

	printKeyManagementInstructions(privateKeyFile, publicKey)
	printUsageInstructions()

	return nil
}

func printKeyManagementInstructions(privateKeyFile, publicKey string) {
	fmt.Printf("\n📋 Key Management Instructions:\n\n")

	fmt.Printf("1. Move private key to secure location:\n")
	fmt.Printf("   mkdir -p ~/.config/kiln\n")
	fmt.Printf("   mv %s ~/.config/kiln/\n\n", privateKeyFile)

	fmt.Printf("2. Set environment variable (add to ~/.bashrc or ~/.zshrc):\n")
	fmt.Printf("   export KILN_PRIVATE_KEY_FILE=~/.config/kiln/%s\n\n", filepath.Base(privateKeyFile))

	fmt.Printf("3. Share public key with team members:\n")
	fmt.Printf("   %s\n\n", publicKey)

	fmt.Printf("Alternative storage options:\n")
	fmt.Printf("  • Password manager (recommended for personal use)\n")
	fmt.Printf("  • Hardware security key (age-plugin-yubikey)\n")
	fmt.Printf("  • Cloud secret management (AWS Secrets Manager, etc.)\n")
	fmt.Printf("  • Environment variable: export KILN_PRIVATE_KEY=\"$(cat ~/.config/kiln/%s)\"\n",
		filepath.Base(privateKeyFile))
}

func printUsageInstructions() {
	fmt.Printf("\n🚀 You can now:\n")
	fmt.Printf("  • kiln edit                    # Edit environment variables\n")
	fmt.Printf("  • kiln export                  # Export variables for shell\n")
	fmt.Printf("  • kiln export --format json    # Export as JSON\n")
	fmt.Printf("  • kiln run -- <command>        # Run command with environment\n")
	fmt.Printf("  • kiln version                 # Show version information\n")
	fmt.Printf("\n💡 For help: kiln --help\n")
}

func createEmptyEnvFile(envFile string, recipients []string) error {
	// Validate the file path
	if err := utils.ValidateFilePath(envFile); err != nil {
		return errors.Wrap(err, "invalid environment file path")
	}

	// Create age manager
	ageManager, err := crypto.NewAgeManager(recipients)
	if err != nil {
		return err
	}

	// Create helpful template content
	template := `# Kiln Environment Variables
# 
# Add your environment variables below using the format:
# VARIABLE_NAME=value
#
# Examples:
# DATABASE_URL=postgres://user:pass@localhost/db
# API_TOKEN=your_secret_token_here
# DEBUG=true
# PORT=8080
#
# Notes:
# - Lines starting with # are comments
# - Values with spaces should be quoted: VAR="value with spaces"
# - No spaces around the = sign
# - Variable names should be UPPERCASE with underscores

`

	// Encrypt the template
	encrypted, err := ageManager.Encrypt([]byte(template))
	if err != nil {
		return err
	}

	// Write encrypted file
	return utils.SaveFile(envFile, encrypted)
}
