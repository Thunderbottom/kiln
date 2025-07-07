package commands

import (
	"encoding/json"
	"fmt"
	"os"
)

type GetCmd struct {
	Name   string `arg:"" help:"Environment variable name"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) Run(globals *Globals) error {
	cmd := NewCommand(globals)

	value, cleanup, err := cmd.Session().GetVar(c.File, c.Name)
	if err != nil {
		return err
	}
	defer cleanup()

	switch c.Format {
	case "value":
		fmt.Print(string(value))
	case "json":
		return json.NewEncoder(os.Stdout).Encode(map[string]string{c.Name: string(value)})
	}
	return nil
}
