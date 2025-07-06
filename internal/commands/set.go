package commands

import "github.com/thunderbottom/kiln/internal/core"

type SetCmd struct {
	Key   string `arg:"" help:"Environment variable key"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	if err := core.SetVar(ctx, globals.Config, c.File, c.Key, c.Value); err != nil {
		return err
	}

	globals.Logger.Info("environment variable set successfully", "key", c.Key, "file", c.File)
	return nil
}
