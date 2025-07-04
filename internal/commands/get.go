package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/utils"
)

type GetCmd struct {
	Key    string `arg:"" help:"Environment variable key"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) Run(globals *Globals) error {
	envVars, err := c.loadEnvVars(globals)
	if err != nil {
		return err
	}

	value, exists := envVars[c.Key]
	if !exists {
		return fmt.Errorf("variable %s not found", c.Key)
	}

	return c.outputValue(value)
}

func (c *GetCmd) loadEnvVars(globals *Globals) (map[string]string, error) {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	envFilePath := cfg.GetEnvFile(c.File)

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

func (c *GetCmd) outputValue(value string) error {
	switch c.Format {
	case "value":
		fmt.Println(value)
	case "json":
		result := map[string]string{c.Key: value}
		encoder := json.NewEncoder(os.Stdout)
		return encoder.Encode(result)
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}

	return nil
}
