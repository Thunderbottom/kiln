package commands

import (
	"github.com/rs/zerolog"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

// Command provides common functionality for all commands
type Command struct {
	globals *Globals
}

// NewCommand creates a base command with globals
func NewCommand(g *Globals) Command {
	return Command{globals: g}
}

// Session returns the session
func (cmd Command) Session() *core.Session {
	return cmd.globals.Session()
}

// Logger returns the logger
func (cmd Command) Logger() *zerolog.Logger {
	return &cmd.globals.Logger
}

// Config returns the config
func (cmd Command) Config() *config.Config {
	return cmd.Session().Config()
}
