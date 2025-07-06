package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/core"
)

type GetCmd struct {
	Name   string `arg:"" help:"Environment variable name"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) Run(globals *Globals) error {
	ctx := globals.Context()
	value, err := core.GetVar(ctx, globals.Config, c.File, c.Name, globals.Key)
	if err != nil {
		return err
	}

	switch c.Format {
	case "value":
		fmt.Println(value)
	case "json":
		return json.NewEncoder(os.Stdout).Encode(map[string]string{c.Name: value})
	}
	return nil
}
