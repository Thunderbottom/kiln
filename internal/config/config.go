package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/thunderbottom/kiln/internal/errors"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = ".kiln.yaml"
	DefaultEnvFile    = ".kiln.env"
	ConfigVersion     = 1
)

type Config struct {
	Version    int               `yaml:"version"`
	Recipients []string          `yaml:"recipients"`
	Created    time.Time         `yaml:"created"`
	Updated    time.Time         `yaml:"updated"`
	Files      map[string]string `yaml:"files"`
}

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

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "failed to read config file")
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

func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigFile
	}

	c.Updated = time.Now()

	data, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Version != ConfigVersion {
		return errors.ErrInvalidConfig
	}

	if len(c.Recipients) == 0 {
		return errors.ErrInvalidConfig
	}

	if c.Files == nil {
		c.Files = map[string]string{
			"default": DefaultEnvFile,
		}
	}

	return nil
}

func (c *Config) AddRecipient(recipient string) {
	for _, r := range c.Recipients {
		if r == recipient {
			return
		}
	}
	c.Recipients = append(c.Recipients, recipient)
}

func (c *Config) GetEnvFile(name string) string {
	if name == "" {
		name = "default"
	}

	if file, exists := c.Files[name]; exists {
		return file
	}

	return DefaultEnvFile
}

func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigFile
	}

	_, err := os.Stat(path)
	return err == nil
}

func FindConfigFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current directory")
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

	return "", errors.ErrConfigNotFound
}
