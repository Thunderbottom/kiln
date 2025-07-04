package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thunderbottom/kiln/internal/errors"
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
	Teams      []Team            `yaml:"teams,omitempty"`
}

// Team represents a team configuration for access control
type Team struct {
	Name       string   `yaml:"name"`
	Recipients []string `yaml:"recipients"`
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
		Teams: []Team{},
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
			return nil, errors.ErrConfigNotFound
		}
		return nil, errors.New("config.Load", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, "failed to parse config file")
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
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
		return errors.New("config.Save", err)
	}

	// Create directory if it doesn't exist
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.New("config.Save", err)
		}
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return errors.New("config.Save", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Version != ConfigVersion {
		return errors.ErrInvalidConfig
	}

	if len(c.Recipients) == 0 {
		return errors.Wrap(errors.ErrInvalidConfig, "no recipients configured")
	}

	// Validate recipient format
	for _, recipient := range c.Recipients {
		if !strings.HasPrefix(recipient, "age1") {
			return errors.Wrapf(errors.ErrInvalidConfig, "invalid recipient format: %s", recipient)
		}
	}

	// Set default files if empty
	if c.Files == nil {
		c.Files = map[string]string{
			"default": DefaultEnvFile,
		}
	}

	// Validate team recipients
	for _, team := range c.Teams {
		for _, recipient := range team.Recipients {
			if !strings.HasPrefix(recipient, "age1") {
				return errors.Wrapf(errors.ErrInvalidConfig,
					"invalid team recipient format in team %s: %s", team.Name, recipient)
			}
		}
	}

	return nil
}

// AddRecipient adds a recipient if not already present
func (c *Config) AddRecipient(recipient string) {
	for _, r := range c.Recipients {
		if r == recipient {
			return // Already exists
		}
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

// ListEnvFiles returns all configured environment files
func (c *Config) ListEnvFiles() map[string]string {
	return c.Files
}

// AddTeam adds a team configuration
func (c *Config) AddTeam(team Team) {
	c.Teams = append(c.Teams, team)
}

// GetTeamRecipients returns recipients for a specific team
func (c *Config) GetTeamRecipients(teamName string) []string {
	for _, team := range c.Teams {
		if team.Name == teamName {
			return team.Recipients
		}
	}
	return nil
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
		return "", errors.New("config.FindConfigFile", err)
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

	return "", errors.ErrConfigNotFound
}
