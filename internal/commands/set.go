package commands

import "github.com/thunderbottom/kiln/internal/core"

type SetCmd struct {
	Name  string `arg:"" help:"Environment variable name"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	if err := core.SetVar(ctx, globals.Config, c.File, c.Name, c.Value, globals.Key); err != nil {
		return err
	}

	globals.Logger.Info().
		Str("key", c.Name).
		Str("file", c.File).
		Msg("environment variable set successfully")

	return nil
}
