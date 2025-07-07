package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/thunderbottom/kiln/internal/core"
	"gopkg.in/yaml.v3"
)

type ExportCmd struct {
	File   string `short:"f" help:"Environment file to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	NoMask bool   `help:"Disable masking sensitive values"`
	Expand bool   `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	sess, err := globals.Session()
	if err != nil {
		return err
	}

	ctx := globals.Context()
	envVars, err := sess.ExportVars(ctx, c.File, c.Expand)
	if err != nil {
		return err
	}

	// Apply masking unless disabled
	if !c.NoMask {
		envVars = sess.MaskVars(envVars)
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
