package commands

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Globals contains global configuration shared across all commands
type Globals struct {
	Config string
	Logger zerolog.Logger
}

// NewGlobals creates a new Globals instance with proper logger setup
func NewGlobals(config string, verbose bool) *Globals {
	// Configure zerolog for performance
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var logger zerolog.Logger
	// Pretty console output for development
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Logger()

	// Set global log level based on verbose flag
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logger = logger.With().
			Caller().Logger()
	}

	return &Globals{
		Config: config,
		Logger: logger,
	}
}

// Context returns a context with the logger attached
func (g *Globals) Context() context.Context {
	return g.Logger.WithContext(context.Background())
}
