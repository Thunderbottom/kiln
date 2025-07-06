package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"filippo.io/age"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/utils"
	"golang.org/x/term"
)

type InitCmd struct {
	Key    *InitKeyCmd    `cmd:"" help:"Generate encryption key"`
	Config *InitConfigCmd `cmd:"" help:"Generate configuration file"`
}

type InitKeyCmd struct {
	Path    string `help:"Path for private key" default:"~/.kiln/kiln.key" type:"path"`
	Encrypt bool   `help:"Save key with passphrase protection"`
	Force   bool   `help:"Overwrite existing key (dangerous!)"`
}

type InitConfigCmd struct {
	Path  string   `help:"Path for config file" default:"kiln.toml"`
	Key   []string `help:"Path to public key file(s) or public key strings" required:""`
	Force bool     `help:"Overwrite existing config"`
}

// Run generates a new encryption key
func (c *InitKeyCmd) Run(globals *Globals) error {
	keyPath, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}

	// Check if key already exists
	if utils.FileExists(keyPath) && !c.Force {
		return fmt.Errorf("private key already exists at %s. Overwriting will make existing encrypted files unreadable. Use --force to overwrite (NOT RECOMMENDED)", keyPath)
	}

	// Generate key pair
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	privateKey := identity.String()
	publicKey := identity.Recipient().String()

	// Handle encryption for private key
	if c.Encrypt {
		encryptedKey, err := encryptPrivateKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		privateKey = encryptedKey
		utils.WipeString(identity.String())
	}

	// Save private key
	if err := utils.SavePrivateKey(privateKey, keyPath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if !c.Encrypt {
		globals.Logger.Warn(`private key is NOT password protected
Consider regenerating the key with password protection or use external encryption`)
	}

	globals.Logger.Info("key pair generated", "private", keyPath, "encrypted", c.Encrypt)
	fmt.Printf("\nage public key: %s\n", publicKey)

	return nil
}

// Run generates a new configuration file
func (c *InitConfigCmd) Run(globals *Globals) error {
	// Check if config already exists
	if config.Exists(c.Path) && !c.Force {
		return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", c.Path)
	}

	// Load all public keys
	var recipients []string
	for _, keyInput := range c.Key {
		publicKey, err := loadPublicKey(keyInput)
		if err != nil {
			return fmt.Errorf("failed to load key %s: %w", keyInput, err)
		}
		recipients = append(recipients, publicKey)
	}

	// Create and save config
	cfg := config.NewConfig()
	for _, recipient := range recipients {
		cfg.AddRecipient(recipient)
	}

	if err := cfg.Save(c.Path); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	globals.Logger.Info("configuration created", "path", c.Path, "recipients", len(recipients))
	return nil
}

// encryptPrivateKey encrypts a private key using age's native passphrase protection
func encryptPrivateKey(privateKey string) (string, error) {
	// Let age handle passphrase prompting and generation
	fmt.Print("Enter passphrase (leave empty to autogenerate): ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return "", err
	}

	if _, err := w.Write([]byte(privateKey)); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// loadPublicKey loads a public key from either a string or file path
func loadPublicKey(input string) (string, error) {
	// Try as direct public key string first
	if crypto.IsValidPublicKey(input) {
		return input, nil
	}

	// Try as file path
	if !utils.FileExists(input) {
		return "", fmt.Errorf("input %s is neither a valid public key nor a readable file", input)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", input, err)
	}
	defer utils.WipeData(data)

	content := strings.TrimSpace(string(data))

	// Could be a public key
	if crypto.IsValidPublicKey(content) {
		return content, nil
	}

	// Could be a plain private key - extract public part
	if crypto.IsPrivateKey(content) {
		identity, err := age.ParseX25519Identity(content)
		if err != nil {
			return "", fmt.Errorf("invalid private key format in %s: %w", input, err)
		}
		return identity.Recipient().String(), nil
	}

	// Could be a passphrase-encrypted identity file
	if strings.Contains(content, "age-encryption.org/v1") {
		fmt.Printf("Private key at %s is passphrase-protected.\n", input)

		// Decrypt the private key
		decryptedKey, err := utils.DecryptPrivateKey(content)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt private key from %s: %w", input, err)
		}
		defer utils.WipeString(decryptedKey)

		// Extract public key from decrypted private key
		identity, err := age.ParseX25519Identity(decryptedKey)
		if err != nil {
			return "", fmt.Errorf("invalid decrypted private key format in %s: %w", input, err)
		}
		return identity.Recipient().String(), nil
	}

	return "", fmt.Errorf("file %s does not contain a valid age key", input)
}
