package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

type ExportCmd struct {
	File   string `short:"f" help:"Environment file to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	NoMask bool   `help:"Disable masking sensitive values"`
	Expand bool   `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
	Key    string `help:"Path to private key file to use for decryption" default:"~/.kiln/kiln.key" type:"path"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	envVars, err := core.ExportVars(ctx, globals.Config, c.File, c.Key, c.Expand)
	if err != nil {
		return err
	}

	// Apply masking unless disabled
	if !c.NoMask {
		cfg, err := config.Load(globals.Config)
		if err != nil {
			return fmt.Errorf("failed to load config for masking: %w", err)
		}
		envVars = core.MaskVars(envVars, cfg)
	}

	switch c.Format {
	case "shell":
		keys := core.SortedKeys(envVars)
		for _, key := range keys {
			value := envVars[key]
			fmt.Printf("export %s='%s'\n", key, strings.ReplaceAll(value, "'", "'\"'\"'"))
		}
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(envVars)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer func() {
			if err := encoder.Close(); err != nil {
				globals.Logger.Debug().Err(err).Msg("failed to close yaml encoder")
			}
		}()

		return encoder.Encode(envVars)
	}
	return nil
}
