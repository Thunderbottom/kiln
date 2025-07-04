package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/core"
)

type GetCmd struct {
	Key    string `arg:"" help:"Environment variable key"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	envVars, err := core.LoadEnvVars(ctx, globals.Config, c.File)
	if err != nil {
		return err
	}

	value, exists := envVars[c.Key]
	if !exists {
		return fmt.Errorf("variable %s not found", c.Key)
	}

	switch c.Format {
	case "value":
		fmt.Println(value)
	case "json":
		return json.NewEncoder(os.Stdout).Encode(map[string]string{c.Key: value})
	}
	return nil
}
