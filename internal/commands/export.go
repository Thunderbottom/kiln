package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// ExportCmd represents the export command for outputting environment variables.
type ExportCmd struct {
	File   string `short:"f" help:"Environment file from the configuration to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell" placeholder:"[shell|json|yaml]"`
}

func (c *ExportCmd) validate() error {
	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	return nil
}

// Run executes the export command, outputting variables in the specified format.
func (c *ExportCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "export").Str("file", c.File).Str("format", c.Format).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	identity, err := rt.Identity()
	if err != nil {
		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	variables, cleanup, err := core.GetAllEnvVars(identity, cfg, c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	switch c.Format {
	case "shell":
		c.exportShell(variables)

		return nil
	case "json":
		return c.exportJSON(variables)
	case "yaml":
		return c.exportYAML(variables)
	}

	return nil
}

func (c *ExportCmd) exportJSON(variables map[string][]byte) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	stringMap := make(map[string]string, len(variables))
	for key, value := range variables {
		stringMap[key] = string(value)
	}

	return encoder.Encode(stringMap)
}

func (c *ExportCmd) exportYAML(variables map[string][]byte) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer func() {
		if closeErr := encoder.Close(); closeErr != nil {
			// Log YAML encoder close error to stderr without failing the export
			// since the data has already been written successfully
			fmt.Fprintf(os.Stderr, "warning: YAML encoder close error: %v\n", closeErr)
		}
	}()

	stringMap := make(map[string]string, len(variables))
	for key, value := range variables {
		stringMap[key] = string(value)
	}

	return encoder.Encode(stringMap)
}

func (c *ExportCmd) exportShell(variables map[string][]byte) {
	var builder strings.Builder

	keys := core.SortedKeys(variables)

	for _, key := range keys {
		value := string(variables[key])

		builder.WriteString("export ")
		builder.WriteString(key)
		builder.WriteString("='")
		builder.WriteString(strings.ReplaceAll(value, "'", "'\"'\"'"))
		builder.WriteString("'\n")
	}

	fmt.Print(builder.String())
}
