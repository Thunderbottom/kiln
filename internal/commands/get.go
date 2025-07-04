package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// GetCmd represents the get command for retrieving a single environment variable.
type GetCmd struct {
	Name   string `arg:"" help:"Environment variable name"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) validate() error {
	if c.Name == "" {
		return kerrors.ValidationError("variable name", "name is required")
	}

	if !core.IsValidVarName(c.Name) {
		return kerrors.ValidationError("variable name", "must start with letter or underscore, followed by letters, numbers, or underscores")
	}

	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	return nil
}

// Run executes the get command, retrieving and displaying a specific variable.
func (c *GetCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "get").Str("variable", c.Name).Str("file", c.File).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	identity, err := rt.Identity()
	if err != nil {
		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	value, cleanup, err := core.GetEnvVar(identity, cfg, c.File, c.Name)
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
