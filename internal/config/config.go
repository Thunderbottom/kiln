package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/thunderbottom/kiln/internal/crypto"
)

const (
	DefaultConfigFile = "kiln.toml"
	DefaultEnvFile    = ".kiln.env"
)

// Config represents the kiln configuration
type Config struct {
	Recipients []string          `toml:"recipients"`
	Files      map[string]string `toml:"files"`
	Security   SecurityConfig    `toml:"security,omitempty"`
}

type SecurityConfig struct {
	MaskKeys  []string `toml:"mask_keys,omitempty"`
	ShowChars int      `toml:"show_chars,omitempty"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Recipients: []string{},
		Files: map[string]string{
			"default": DefaultEnvFile,
		},
		Security: SecurityConfig{
			MaskKeys:  []string{},
			ShowChars: 4,
		},
	}
}

// Load reads and validates a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients in config")
	}

	return &config, nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Recipients) == 0 {
		return fmt.Errorf("no recipients configured - run 'kiln init' or add recipients")
	}

	for i, recipient := range c.Recipients {
		if err := crypto.ValidatePublicKey(recipient); err != nil {
			return fmt.Errorf("recipient %d: %w", i+1, err)
		}
	}

	for name, path := range c.Files {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("file path for '%s' is empty", name)
		}
	}

	return nil
}

// AddRecipient adds a recipient if not already present
func (c *Config) AddRecipient(recipient string) {
	if slices.Contains(c.Recipients, recipient) {
		return
	}
	c.Recipients = append(c.Recipients, recipient)
}

// RemoveRecipient removes a recipient
func (c *Config) RemoveRecipient(recipient string) bool {
	for i, r := range c.Recipients {
		if r == recipient {
			c.Recipients = append(c.Recipients[:i], c.Recipients[i+1:]...)
			return true
		}
	}
	return false
}

// GetEnvFile returns the path for the specified environment file
func (c *Config) GetEnvFile(name string) (string, error) {
	if name == "" {
		name = "default"
	}

	if file, exists := c.Files[name]; exists {
		return file, nil
	}

	available := make([]string, 0, len(c.Files))
	for fileName := range c.Files {
		available = append(available, fileName)
	}

	return "", fmt.Errorf("file '%s' not found in configuration, available files: %v", name, available)
}

// Exists checks if a config file exists
func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigFile
	}
	_, err := os.Stat(path)
	return err == nil
}

// ShouldMaskKey checks if a key should be masked based on config
func (c *Config) ShouldMaskKey(key string) bool {
	for _, maskKey := range c.Security.MaskKeys {
		if strings.EqualFold(key, maskKey) {
			return true
		}
	}
	return false
}

// MaskValue masks a value according to config settings
func (c *Config) MaskValue(value string) string {
	if c.Security.ShowChars <= 0 {
		return strings.Repeat("*", len(value))
	}

	if len(value) <= c.Security.ShowChars {
		return strings.Repeat("*", len(value))
	}

	return strings.Repeat("*", len(value)-c.Security.ShowChars) + value[len(value)-c.Security.ShowChars:]
}
