package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = ".kiln.yaml"
	DefaultEnvFile    = ".kiln.env"
	ConfigVersion     = 1
)

// Config represents the kiln configuration
type Config struct {
	Version    int               `yaml:"version"`
	Recipients []string          `yaml:"recipients"`
	Created    time.Time         `yaml:"created"`
	Updated    time.Time         `yaml:"updated"`
	Files      map[string]string `yaml:"files"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	now := time.Now()
	return &Config{
		Version:    ConfigVersion,
		Recipients: []string{},
		Created:    now,
		Updated:    now,
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

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kiln configuration not found at %s", path)
		}
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
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

	c.Updated = time.Now()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Create directory if needed
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Version != ConfigVersion {
		return fmt.Errorf("unsupported config version %d, expected %d", c.Version, ConfigVersion)
	}

	if len(c.Recipients) == 0 {
		return fmt.Errorf("no recipients configured")
	}

	// Validate recipient format
	for _, recipient := range c.Recipients {
		if strings.TrimSpace(recipient) == "" {
			return fmt.Errorf("empty recipient key")
		}
	}

	// Set default files if empty
	if c.Files == nil {
		c.Files = map[string]string{
			"default": DefaultEnvFile,
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
func (c *Config) GetEnvFile(name string) string {
	if name == "" {
		name = "default"
	}

	if file, exists := c.Files[name]; exists {
		return file
	}

	return DefaultEnvFile
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
			break // Reached root directory
		}
		cwd = parent
	}

	return "", fmt.Errorf("kiln configuration not found")
}
