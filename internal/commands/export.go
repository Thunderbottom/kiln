package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thunderbottom/kiln/internal/core"
)

type ExportCmd struct {
	File   string `short:"f" help:"Environment file to export" default:"default"`
	Format string `help:"Output format" enum:"shell,json,yaml" default:"shell"`
	Mask   bool   `help:"Mask sensitive values"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	envVars, err := core.LoadEnvVars(globals.Config, c.File)
	if err != nil {
		return err
	}

	envVars = core.ProcessEnvVars(envVars, c.Mask)

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
