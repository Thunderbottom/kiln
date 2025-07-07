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
	Expand bool   `help:"Enable variable expansion ($${VAR} syntax)" default:"false"`
}

func (c *ExportCmd) Run(globals *Globals) error {
	sess, err := globals.Session()
	if err != nil {
		return err
	}

	envVars, err := sess.ExportVars(c.File, c.Expand)
	if err != nil {
		return err
	}

	stringVars := make(map[string]string)
	for key, value := range envVars {
		stringVars[key] = string(value)
		defer core.WipeData(value)
	}

	switch c.Format {
	case "shell":
		keys := core.SortedKeys(envVars)
		for _, key := range keys {
			value := stringVars[key]
			fmt.Printf("export %s='%s'\n", key, strings.ReplaceAll(value, "'", "'\"'\"'"))
		}
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(stringVars)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer func() { _ = encoder.Close() }()
		return encoder.Encode(stringVars)
	}
	return nil
}
