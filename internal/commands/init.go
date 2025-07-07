package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"filippo.io/age"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
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
	Path       string   `help:"Path for config file" default:"kiln.toml"`
	PublicKeys []string `help:"Path to public key file(s) or public key strings" required:""`
	Force      bool     `help:"Overwrite existing config"`
}

// Run generates a new encryption key
func (c *InitKeyCmd) Run(globals *Globals) error {
	keyPath, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}

	// Check if key already exists
	if core.FileExists(keyPath) && !c.Force {
		return fmt.Errorf("private key already exists at %s. Overwriting will make existing encrypted files unreadable. Use --force to overwrite (NOT RECOMMENDED)", keyPath)
	}

	// Generate key pair
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	privateKey := []byte(identity.String())
	publicKey := identity.Recipient().String()

	if c.Encrypt {
		encryptedKey, err := encryptPrivateKey(string(privateKey))
		if err != nil {
			core.WipeData(privateKey) // Wipe original
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		privateKey = []byte(encryptedKey)
		core.WipeData(privateKey) // Wipe original unencrypted key
	}

	// Save private key
	if err := core.SavePrivateKey(privateKey, keyPath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}
	core.WipeData(privateKey)

	if !c.Encrypt {
		globals.Logger.Warn().Msg("private key is NOT password protected")
	}

	globals.Logger.Info().
		Str("private", keyPath).
		Bool("encrypted", c.Encrypt).
		Msg("key pair generated")
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
	for _, keyInput := range c.PublicKeys {
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

	globals.Logger.Info().
		Str("path", c.Path).
		Int("recipients", len(recipients)).
		Msg("configuration created")

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
	if core.IsValidPublicKey(input) {
		return input, nil
	}

	// Try as file path
	if !core.FileExists(input) {
		return "", fmt.Errorf("input %s is neither a valid public key nor a readable file", input)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", input, err)
	}
	defer core.WipeData(data)

	content := strings.TrimSpace(string(data))

	// Could be a public key
	if core.IsValidPublicKey(content) {
		return content, nil
	}

	// Could be a plain private key - extract public part
	if core.IsPrivateKey(content) {
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
		decryptedKey, err := decryptPrivateKeyFromFile(content)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt private key from %s: %w", input, err)
		}
		defer core.WipeString(decryptedKey)

		// Extract public key from decrypted private key
		identity, err := age.ParseX25519Identity(decryptedKey)
		if err != nil {
			return "", fmt.Errorf("invalid decrypted private key format in %s: %w", input, err)
		}
		return identity.Recipient().String(), nil
	}

	return "", fmt.Errorf("file %s does not contain a valid age key", input)
}

// decryptPrivateKeyFromFile decrypts a passphrase-protected private key from file content
func decryptPrivateKeyFromFile(encryptedKey string) (string, error) {
	fmt.Print("Enter passphrase: ")
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	defer core.WipeData(passphrase)

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return "", fmt.Errorf("failed to create scrypt identity: %w", err)
	}

	reader := strings.NewReader(encryptedKey)
	r, err := age.Decrypt(reader, identity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted private key: %w", err)
	}
	defer core.WipeData(decrypted)

	return string(decrypted), nil
}
