package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	"github.com/thunderbottom/kiln/internal/env"
)

type ExportCmd struct {
	File          string `short:"f" help:"Environment file to export" default:"default"`
	Format        string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	NoMask        bool   `help:"Disable masking sensitive values"`
	Expand        bool   `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
	AllowCommands bool   `help:"Allow command substitution ($$(command) syntax)"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	envVars, err := core.LoadEnvVars(ctx, globals.Config, c.File)
	if err != nil {
		return err
	}

	// Apply variable expansion if enabled
	if c.Expand {
		globals.Logger.Debug("applying variable expansion")
		if c.AllowCommands {
			globals.Logger.Debug("command substitution enabled")
		}
		envVars = env.ExpandVariables(envVars, c.AllowCommands)
	}

	// Load config for masking
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load config for masking: %w", err)
	}

	// Apply masking unless disabled
	if !c.NoMask {
		envVars = core.ProcessEnvVars(envVars, cfg)
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
		defer encoder.Close()
		return encoder.Encode(envVars)
	}
	return nil
}
