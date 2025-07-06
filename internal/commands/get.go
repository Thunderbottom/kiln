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
	value, err := core.GetVar(ctx, globals.Config, c.File, c.Key)
	if err != nil {
		return err
	}

	switch c.Format {
	case "value":
		fmt.Println(value)
	case "json":
		return json.NewEncoder(os.Stdout).Encode(map[string]string{c.Key: value})
	}
	return nil
}
