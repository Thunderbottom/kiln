package commands

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/core"
)

type SetCmd struct {
	Key   string `arg:"" help:"Environment variable key"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	envVars, err := core.LoadOrCreateEnvVars(globals.Config, c.File)
	if err != nil {
		return err
	}

	envVars[c.Key] = c.Value
	if err := core.SaveEnvVars(globals.Config, c.File, envVars); err != nil {
		return err
	}

	fmt.Printf("Set %s in %s\n", c.Key, c.File)
	return nil
}
