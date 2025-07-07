package commands

import (
	"fmt"
	"syscall"

	"golang.org/x/term"
)

type SetCmd struct {
	Name  string `arg:"" help:"Environment variable name"`
	Value string `arg:"" help:"Environment variable value (if not provided, will prompt for input)" optional:""`
	File  string `short:"f" help:"Environment file to modify" default:"default"`
}

func (c *SetCmd) Run(globals *Globals) error {
	cmd := NewCommand(globals)

	var value []byte
	var err error

	if c.Value != "" {
		value = []byte(c.Value)
	} else {
		value, err = c.readValueFromStdin()
		if err != nil {
			return fmt.Errorf("read value from stdin: %w", err)
		}
	}
	defer func() {
		for i := range value {
			value[i] = 0
		}
	}()

	if err := cmd.Session().SetVar(c.File, c.Name, value); err != nil {
		return err
	}

	cmd.Logger().Info().
		Str("key", c.Name).
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
