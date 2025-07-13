// Package config handles kiln configuration file management, including loading, saving,
// and validating configuration files that specify recipients and environment file mappings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	Recipients map[string]string     `toml:"recipients"`
	Groups     map[string][]string   `toml:"groups"`
	Files      map[string]FileConfig `toml:"files"`
}

// FileConfig represents the configuration for an environment file
type FileConfig struct {
	Filename string   `toml:"filename"`
	Access   []string `toml:"access"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Recipients: make(map[string]string),
		Groups:     make(map[string][]string),
		Files: map[string]FileConfig{
			"default": {
				Filename: DefaultEnvFile,
				Access:   []string{"*"},
			},
		},
	}
}

// Load reads and validates a configuration file
func Load(path string) (*Config, error) {
	configPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients in configuration")
	}

	// Resolve relative file paths relative to the configuration directory
	configDir := filepath.Dir(configPath)

	for name, fileConfig := range config.Files {
		if !filepath.IsAbs(fileConfig.Filename) {
			fileConfig.Filename = filepath.Join(configDir, fileConfig.Filename)
			config.Files[name] = fileConfig
		}
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
		return fmt.Errorf("no recipients configured")
	}

	for name, fileConfig := range c.Files {
		if strings.TrimSpace(fileConfig.Filename) == "" {
			return fmt.Errorf("file path for '%s' is empty", name)
		}

		if len(fileConfig.Access) == 0 {
			return fmt.Errorf("no access control defined for file '%s'", name)
		}
	}

	return nil
}

// AddRecipient adds a recipient if not already present
func (c *Config) AddRecipient(name, publicKey string) {
	if c.Recipients == nil {
		c.Recipients = make(map[string]string)
	}

	c.Recipients[name] = publicKey
}

// RemoveRecipient removes a recipient
func (c *Config) RemoveRecipient(name string) bool {
	if c.Recipients == nil {
		return false
	}

	_, exists := c.Recipients[name]
	if exists {
		delete(c.Recipients, name)
	}

	return exists
}

// ResolveFileAccess resolves the list of public keys that have access to a specific file
func (c *Config) ResolveFileAccess(fileName string) ([]string, error) {
	fileConfig, exists := c.Files[fileName]
	if !exists {
		return nil, fmt.Errorf("file '%s' not found in configuration", fileName)
	}

	recipientSet := make(map[string]bool)

	for _, accessor := range fileConfig.Access {
		// If access is a wildcard, add all and break early
		if accessor == "*" {
			for _, pubKey := range c.Recipients {
				recipientSet[pubKey] = true
			}

			break
		}

		// Check if accessor is a group
		if groupMembers, isGroup := c.Groups[accessor]; isGroup {
			for _, member := range groupMembers {
				if pubKey, exists := c.Recipients[member]; exists {
					recipientSet[pubKey] = true
				}
			}

			continue
		}

		// Check for individual recipients
		if pubKey, exists := c.Recipients[accessor]; exists {
			recipientSet[pubKey] = true
		}
	}

	recipients := make([]string, 0, len(recipientSet))
	for pubKey := range recipientSet {
		recipients = append(recipients, pubKey)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no valid recipients found for file '%s'", fileName)
	}

	return recipients, nil
}

// GetEnvFile returns the path for the specified environment file
func (c *Config) GetEnvFile(name string) (string, error) {
	if name == "" {
		name = "default"
	}

	if fileConfig, exists := c.Files[name]; exists {
		return fileConfig.Filename, nil
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
