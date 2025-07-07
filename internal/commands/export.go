package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/core"
)

// ExportCmd represents the export command for outputting environment variables.
type ExportCmd struct {
	File   string `short:"f" help:"Environment file from the configuration to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell" placeholder:"[shell|json|yaml]"`
	Expand bool   `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
}

// Run executes the export command, outputting variables in the specified format.
func (c *ExportCmd) Run(globals *Globals) error {
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	globals.Logger.Debug().
		Str("file", c.File).
		Str("format", c.Format).
		Bool("expand", c.Expand).
		Msg("exporting environment variables")

	variables, cleanup, err := session.ExportVars(c.File, c.Expand)
	if err != nil {
		globals.Logger.Error().
			Err(err).
			Str("file", c.File).
			Msg("failed to export variables")

		return err
	}
	defer cleanup()

	globals.Logger.Debug().
		Int("count", len(variables)).
		Str("format", c.Format).
		Msg("variables loaded successfully")

	stringVariables := make(map[string]string)
	for key, value := range variables {
		stringVariables[key] = string(value)
	}

	switch c.Format {
	case "shell":
		keys := core.SortedKeys(variables)
		for _, key := range keys {
			value := stringVariables[key]
			fmt.Printf("export %s='%s'\n", key, strings.ReplaceAll(value, "'", "'\"'\"'"))
		}
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		return encoder.Encode(stringVariables)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer func() {
			if err := encoder.Close(); err != nil {
				globals.Logger.Debug().Err(err).Msg("failed to close yaml encoder")
			}
		}()

		return encoder.Encode(stringVariables)
	}

	globals.Logger.Debug().
		Int("exported", len(variables)).
		Str("format", c.Format).
		Msg("export completed successfully")

	return nil
}
