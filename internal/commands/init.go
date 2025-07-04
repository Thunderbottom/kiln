package commands

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
)

func Init(ctx context.Context, args []string) error {
	// Check for help flags first
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printInitUsage()
			return nil
		}
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	fromPublicKey := fs.String("from", "", "use existing public key instead of generating new key pair")
	configPath := fs.String("config", "", "custom config file path (default: .kiln.yaml)")
	keyOutput := fs.String("key-output", "", "directory to save private key (default: current directory)")
	verbose := fs.Bool("v", false, "verbose output")

	fs.Usage = printInitUsage

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Determine config file path
	configFile := *configPath
	if configFile == "" {
		configFile = config.DefaultConfigFile
	}

	// Check if project already exists
	if config.Exists(configFile) {
		return fmt.Errorf("kiln project already exists at %s", configFile)
	}

	// Create new configuration
	cfg := config.NewConfig()

	if *fromPublicKey != "" {
		return initWithExistingKey(cfg, configFile, *fromPublicKey, *verbose)
	}

	return initWithNewKeyPair(ctx, cfg, configFile, *keyOutput, *verbose)
}

// printInitUsage prints the usage information for the init command
func printInitUsage() {
	fmt.Print(`Usage: kiln init [flags]

Initialize a new kiln project by generating an age key pair and creating
a configuration file.

Flags:
  -from string        use existing public key instead of generating new key pair
  -config string      custom config file path (default: .kiln.yaml)
  -key-output string  directory to save private key (default: current directory)
  -v                  verbose output

Examples:
  kiln init
  kiln init -from age1xyz...
  kiln init -config custom.yaml
  kiln init -key-output ~/.config/kiln/
`)
}

func initWithExistingKey(cfg *config.Config, configFile, publicKey string, verbose bool) error {
	// Validate the provided public key
	if err := crypto.ValidatePublicKey(publicKey); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	cfg.AddRecipient(publicKey)

	// Save configuration
	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create empty encrypted environment file
	if err := createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil {
		if verbose {
			fmt.Printf("Warning: failed to create empty env file: %v\n", err)
		}
	}

	fmt.Printf("Initialized kiln project with existing public key\n")
	if verbose {
		fmt.Printf("Public key: %s\n", publicKey)
		fmt.Printf("Config file: %s\n", configFile)
	}

	fmt.Printf("\nMake sure you have access to the corresponding private key!\n")
	printUsageInstructions()

	return nil
}

func initWithNewKeyPair(ctx context.Context, cfg *config.Config, configFile, keyOutput string, verbose bool) error {
	fmt.Printf("Generating new age key pair...\n")

	privateKey, publicKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
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
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save private key to file
	privateKeyFile := filepath.Join(keyDir, "kiln.key")
	if err := utils.SavePrivateKey(privateKey, privateKeyFile); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save configuration
	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create empty encrypted environment file
	if err := createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil {
		if verbose {
			fmt.Printf("Warning: failed to create empty env file: %v\n", err)
		}
	}

	fmt.Printf("Generated new age key pair\n")
	if verbose {
		fmt.Printf("Public key: %s\n", publicKey)
		fmt.Printf("Config file: %s\n", configFile)
		fmt.Printf("Private key file: %s\n", privateKeyFile)
	}

	fmt.Printf("\nIMPORTANT: Secure your private key!\n")
	fmt.Printf("Private key saved to: %s (permissions: 0600)\n", privateKeyFile)

	printKeyManagementInstructions(privateKeyFile, publicKey)
	printUsageInstructions()

	return nil
}

func printKeyManagementInstructions(privateKeyFile, publicKey string) {
	fmt.Printf("\nKey Management Instructions:\n\n")
	fmt.Printf("1. Move private key to secure location:\n")
	fmt.Printf("   mkdir -p ~/.config/kiln\n")
	fmt.Printf("   mv %s ~/.config/kiln/\n\n", privateKeyFile)
	fmt.Printf("2. Set environment variable (add to ~/.bashrc or ~/.zshrc):\n")
	fmt.Printf("   export KILN_PRIVATE_KEY_FILE=~/.config/kiln/%s\n\n", filepath.Base(privateKeyFile))
	fmt.Printf("3. Share public key with team members:\n")
	fmt.Printf("   %s\n", publicKey)
}

func printUsageInstructions() {
	fmt.Printf("\nYou can now:\n")
	fmt.Printf("  kiln edit                    # Edit environment variables\n")
	fmt.Printf("  kiln export                  # Export variables for shell\n")
	fmt.Printf("  kiln export --format json    # Export as JSON\n")
	fmt.Printf("  kiln run -- <command>        # Run command with environment\n")
}

func createEmptyEnvFile(envFile string, recipients []string) error {
	if err := utils.ValidateFilePath(envFile); err != nil {
		return fmt.Errorf("invalid environment file path: %w", err)
	}

	ageManager, err := crypto.NewAgeManager(recipients)
	if err != nil {
		return err
	}

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

`

	encrypted, err := ageManager.Encrypt([]byte(template))
	if err != nil {
		return err
	}

	return utils.SaveFile(envFile, encrypted)
}
