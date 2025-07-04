package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type ExportCmd struct {
	File   string `short:"f" help:"Environment file to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	Mask   bool   `help:"Mask sensitive values"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	envVars, err := c.loadEnvVars(globals)
	if err != nil {
		return err
	}

	cfg, _ := config.Load(globals.Config)
	processedVars := c.processVars(envVars, cfg)

	switch c.Format {
	case "shell":
		return c.exportShell(processedVars)
	case "json":
		return c.exportJSON(processedVars)
	case "yaml":
		return c.exportYAML(processedVars)
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}
}

func (c *ExportCmd) loadEnvVars(globals *Globals) (map[string]string, error) {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	envFilePath := cfg.GetEnvFile(c.File)
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("environment file not found: %s", envFilePath)
	}

	ctx := context.Background()
	privateKey, _, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to setup encryption: %w", err)
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return nil, fmt.Errorf("failed to add identity: %w", err)
	}

	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt environment file: %w", err)
	}

	return env.ParseEnvFile(string(plaintext))
}

func (c *ExportCmd) processVars(envVars map[string]string, cfg *config.Config) map[string]string {
	processed := make(map[string]string)
	for key, value := range envVars {
		if c.Mask && cfg != nil && cfg.IsSensitiveKey(key) {
			value = c.maskValue(value)
		}
		processed[key] = value
	}
	return processed
}

func (c *ExportCmd) maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

func (c *ExportCmd) exportShell(envVars map[string]string) error {
	keys := c.getSortedKeys(envVars)
	for _, key := range keys {
		value := envVars[key]
		fmt.Printf("export %s='%s'\n", key, strings.ReplaceAll(value, "'", "'\"'\"'"))
	}
	return nil
}

func (c *ExportCmd) exportJSON(envVars map[string]string) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(envVars)
}

func (c *ExportCmd) exportYAML(envVars map[string]string) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(envVars)
}

func (c *ExportCmd) getSortedKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
