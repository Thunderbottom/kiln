package commands

type SetCmd struct {
	Name  string `arg:"" help:"Environment variable name"`
	Value string `arg:"" help:"Environment variable value"`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	cmd := NewCommand(globals)

	if err := cmd.Session().SetVar(c.File, c.Name, []byte(c.Value)); err != nil {
		return err
	}

	cmd.Logger().Info().
		Str("key", c.Name).
		Str("file", c.File).
		Msg("environment variable set successfully")

	return nil
}
