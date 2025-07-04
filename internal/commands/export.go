package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/config"
)

type ExportCmd struct {
	File   string `short:"f" help:"Environment file to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	Mask   bool   `help:"Mask sensitive values"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	envVars, err := loadEnvVars(globals, c.File)
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

func (c *ExportCmd) processVars(envVars map[string]string, cfg *config.Config) map[string]string {
	processed := make(map[string]string)
	for key, value := range envVars {
		if c.Mask && cfg != nil && c.isSensitiveKey(key) {
			value = c.maskValue(value)
		}
		processed[key] = value
	}
	return processed
}

func (c *ExportCmd) isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	sensitivePatterns := []string{"password", "secret", "token", "key", "auth", "api"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
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
