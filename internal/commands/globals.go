// Package commands implements all CLI commands for the kiln secure environment variable management tool.
// It provides subcommands for initializing projects, editing encrypted files, running commands with
// decrypted environment variables, and managing encryption keys.
package commands

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/thunderbottom/kiln/internal/core"
)

// Globals contains shared configuration and logger for all commands
type Globals struct {
	Config  string
	Key     string
	Logger  zerolog.Logger
	session *core.Session
}

// NewGlobals creates a new Globals instance with configured logger
func NewGlobals(config, key string, verbose bool) (*Globals, error) {
	logLevel := zerolog.InfoLevel
	if verbose {
		logLevel = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}

	logger := zerolog.New(consoleWriter).With().Timestamp()
	if verbose {
		logger = logger.Caller()
	}

	return &Globals{
		Config: config,
		Key:    key,
		Logger: logger.Logger(),
	}, nil
}

// Session returns a session, creating it on first access
func (g *Globals) Session() (*core.Session, error) {
	if g.session != nil {
		return g.session, nil
	}

	var err error
	g.session, err = core.NewSession(g.Config, g.Key)

	return g.session, err
}

// Context returns a context with the logger attached
func (g *Globals) Context() context.Context {
	return g.Logger.WithContext(context.Background())
}
