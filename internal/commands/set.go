package commands

import (
	"fmt"
	"syscall"

	"github.com/thunderbottom/kiln/internal/core"
	"golang.org/x/term"
)

type SetCmd struct {
	Name  string `arg:"" help:"Environment variable name"`
	Value string `arg:"" help:"Environment variable value (if not provided, will prompt for input)" optional:""`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	globals.Logger.Debug().
		Str("variable", c.Name).
		Str("file", c.File).
		Bool("prompt", c.Value == "").
		Msg("setting environment variable")

	var value []byte
	if c.Value != "" {
		value = []byte(c.Value)
		globals.Logger.Debug().
			Str("variable", c.Name).
			Msg("using provided value")
	} else {
		value, err = c.readValueFromStdin()
		if err != nil {
			globals.Logger.Error().
				Err(err).
				Str("variable", c.Name).
				Msg("failed to read value from stdin")
			return fmt.Errorf("read value from stdin: %w", err)
		}
		globals.Logger.Debug().
			Str("variable", c.Name).
			Msg("value read from stdin")
	}
	defer core.WipeData(value)

	if err := session.SetVar(c.File, c.Name, value); err != nil {
		globals.Logger.Error().
			Err(err).
			Str("variable", c.Name).
			Str("file", c.File).
			Msg("failed to set variable")
		return err
	}

	globals.Logger.Info().
		Str("variable", c.Name).
		Str("file", c.File).
		Msg("environment variable set successfully")

	return nil
}

func (c *SetCmd) readValueFromStdin() ([]byte, error) {
	fmt.Printf("Enter value for %s: ", c.Name)
	value, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	return value, nil
}
