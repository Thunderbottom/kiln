// Package config handles kiln configuration file management, including loading, saving,
// and validating configuration files that specify recipients and environment file mappings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// DefaultConfigFile is the default name for kiln configuration files.
	DefaultConfigFile = "kiln.toml"
	// DefaultEnvFile is the default name for encrypted environment files.
	DefaultEnvFile = ".kiln.env"
)

// Config represents the kiln configuration
type Config struct {
	Recipients []string          `toml:"recipients"`
	Files      map[string]string `toml:"files"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Recipients: []string{},
		Files: map[string]string{
			"default": DefaultEnvFile,
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Recipients) == 0 {
		return fmt.Errorf("no recipients configured - run 'kiln init' or add recipients")
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
