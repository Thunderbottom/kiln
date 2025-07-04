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
	DefaultConfigFile = "kiln.toml"
	DefaultEnvFile    = ".kiln.env"
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
	if path == "" {
		path = DefaultConfigFile
	}

	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kiln configuration not found at %s", path)
		}
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &config, nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigFile
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %v", err)
	}

	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %v", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Recipients) == 0 {
		return fmt.Errorf("no recipients configured - run 'kiln init' or add recipients")
	}

	for i, recipient := range c.Recipients {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			return fmt.Errorf("recipient %d is empty", i+1)
		}

		if !strings.HasPrefix(recipient, "age1") {
			return fmt.Errorf("recipient %d: invalid format (must start with 'age1')", i+1)
		}

		if len(recipient) != 62 {
			return fmt.Errorf("recipient %d: invalid length", i+1)
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

// FindConfigFile searches for a config file in current and parent directories
func FindConfigFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	for {
		configPath := filepath.Join(cwd, DefaultConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return "", fmt.Errorf("kiln configuration not found")
}
