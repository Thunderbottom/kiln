package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/errors"
)

var (
	fromPublicKey  string
	initConfigPath string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new kiln project",
	Long: `Initialize a new kiln project by generating an Age key pair and creating
a configuration file. The private key should be stored securely and the public
key can be shared with team members.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&fromPublicKey, "from", "", "use existing public key instead of generating new key pair")
	initCmd.Flags().StringVar(&initConfigPath, "config", "", "custom config file path (default: .kiln.yaml)")
}

func runInit(cmd *cobra.Command, args []string) error {
	configFile := initConfigPath
	if configFile == "" {
		configFile = config.DefaultConfigFile
	}

	if config.Exists(configFile) {
		return fmt.Errorf("kiln project already exists at %s", configFile)
	}

	cfg := config.NewConfig()

	if fromPublicKey != "" {
		if err := crypto.ValidatePublicKey(fromPublicKey); err != nil {
			return errors.Wrap(err, "invalid public key")
		}

		cfg.AddRecipient(fromPublicKey)

		if IsVerbose() {
			fmt.Printf("✓ Initialized kiln project with existing public key\n")
			fmt.Printf("  Public key: %s\n", fromPublicKey)
			fmt.Printf("  Config file: %s\n", configFile)
		} else {
			fmt.Printf("✓ Initialized kiln project\n")
		}
		fmt.Printf("\nMake sure you have access to the corresponding private key to decrypt files.\n")
	} else {
		privateKey, publicKey, err := crypto.GenerateKeyPair()
		if err != nil {
			return errors.Wrap(err, "failed to generate key pair")
		}

		cfg.AddRecipient(publicKey)

		if IsVerbose() {
			fmt.Printf("✓ Generated new Age key pair\n")
			fmt.Printf("  Public key: %s\n", publicKey)
			fmt.Printf("  Config file: %s\n", configFile)
		} else {
			fmt.Printf("✓ Generated new Age key pair\n")
		}

		fmt.Printf("\n🔐 IMPORTANT: Store your private key securely!\n")
		fmt.Printf("Private key: %s\n\n", privateKey)

		fmt.Printf("Recommendations for storing your private key:\n")
		fmt.Printf("  • Save to a password manager\n")
		fmt.Printf("  • Store in ~/.config/kiln/key.txt with 600 permissions\n")
		fmt.Printf("  • Add to your shell profile as KILN_PRIVATE_KEY environment variable\n")
		fmt.Printf("  • Use age-plugin-yubikey for hardware security keys\n\n")
		fmt.Printf("Share the public key with team members to grant them access.\n")
	}

	if err := cfg.Save(configFile); err != nil {
		return errors.Wrap(err, "failed to save configuration")
	}

	envFile := cfg.GetEnvFile("default")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		if err := os.WriteFile(envFile, []byte{}, 0600); err != nil {
			if IsVerbose() {
				fmt.Printf("Warning: failed to create empty env file %s: %v\n", envFile, err)
			}
		} else {
			if IsVerbose() {
				fmt.Printf("✓ Created empty environment file: %s\n", envFile)
			}
		}
	}

	return nil
}
