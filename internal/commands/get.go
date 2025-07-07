package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/core"
)

// GetCmd represents the get command for retrieving a single environment variable.
type GetCmd struct {
	Name   string `arg:"" help:"Environment variable name"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

// Run executes the get command, retrieving and displaying a specific variable.
func (c *GetCmd) Run(globals *Globals) error {
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	globals.Logger.Debug().
		Str("file", c.File).
		Str("variable", c.Name).
		Msg("retrieving environment variable")

	variables, cleanup, err := session.LoadVars(c.File)
	if err != nil {
		globals.Logger.Error().
			Err(err).
			Str("variable", c.Name).
			Str("file", c.File).
			Msg("failed to load variables")

		return err
	}
	defer cleanup()

	value, exists := variables[c.Name]
	if !exists {
		globals.Logger.Error().
			Str("variable", c.Name).
			Str("file", c.File).
			Msg("variable not found")

		return fmt.Errorf("variable %s not found", c.Name)
	}

	result := make([]byte, len(value))
	copy(result, value)
	defer core.WipeData(result)

	globals.Logger.Debug().
		Str("variable", c.Name).
		Str("format", c.Format).
		Msg("variable retrieved successfully")

	switch c.Format {
	case "value":
		fmt.Print(string(result))
	case "json":
		return json.NewEncoder(os.Stdout).Encode(map[string]string{c.Name: string(result)})
	}

	return nil
}
