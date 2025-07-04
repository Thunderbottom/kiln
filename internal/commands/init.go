package commands

import (
	"fmt"
	"path/filepath"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
)

type InitCmd struct {
	From      string `help:"Use existing public key"`
	KeyOutput string `help:"Directory to save private key" default:"."`
	Force     bool   `help:"Overwrite existing configuration"`
}

func (c *InitCmd) Run(globals *Globals) error {
	if !c.Force && config.Exists(globals.Config) {
		return fmt.Errorf("kiln project already exists at %s", globals.Config)
	}

	cfg := config.NewConfig()
	if c.From != "" {
		return c.initWithExistingKey(cfg, globals)
	}
	return c.initWithNewKeyPair(cfg, globals)
}

func (c *InitCmd) initWithExistingKey(cfg *config.Config, globals *Globals) error {
	if err := crypto.ValidatePublicKey(c.From); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	cfg.AddRecipient(c.From)
	if err := cfg.Save(globals.Config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if err := c.createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil && globals.Verbose {
		fmt.Printf("Warning: failed to create empty env file: %v\n", err)
	}

	fmt.Printf("Initialized kiln project with existing public key\n")
	return nil
}

func (c *InitCmd) initWithNewKeyPair(cfg *config.Config, globals *Globals) error {
	privateKey, publicKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	cfg.AddRecipient(publicKey)

	keyDir := utils.ExpandPath(c.KeyOutput)
	if err := utils.EnsureDirectoryExists(keyDir); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	privateKeyFile := filepath.Join(keyDir, "kiln.key")
	if err := utils.SavePrivateKey(privateKey, privateKeyFile); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if err := cfg.Save(globals.Config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if err := c.createEmptyEnvFile(cfg.GetEnvFile("default"), cfg.Recipients); err != nil && globals.Verbose {
		fmt.Printf("Warning: failed to create empty env file: %v\n", err)
	}

	fmt.Printf("Generated new age key pair\n")
	if globals.Verbose {
		fmt.Printf("Public key: %s\n", publicKey)
		fmt.Printf("Private key file: %s\n", privateKeyFile)
	}
	return nil
}

func (c *InitCmd) createEmptyEnvFile(envFile string, recipients []string) error {
	ageManager, err := crypto.NewAgeManager(recipients)
	if err != nil {
		return err
	}

	template := `# Kiln Environment Variables
# Format: KEY=value
DATABASE_URL=
API_TOKEN=
DEBUG=false
`
	encrypted, err := ageManager.Encrypt([]byte(template))
	if err != nil {
		return err
	}

	return utils.SaveFile(envFile, encrypted)
}
