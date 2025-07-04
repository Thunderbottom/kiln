package commands

import "github.com/thunderbottom/kiln/internal/core"

type SetCmd struct {
	Key   string `arg:"" help:"Environment variable key"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	envVars, err := core.LoadOrCreateEnvVars(ctx, globals.Config, c.File)
	if err != nil {
		return err
	}

	envVars[c.Key] = c.Value
	if err := core.SaveEnvVars(ctx, globals.Config, c.File, envVars); err != nil {
		return err
	}

	globals.Logger.Info("environment variable set successfully", "key", c.Key, "file", c.File)

	return nil
}
