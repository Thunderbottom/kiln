package commands

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/thunderbottom/kiln/internal/core"
)

// Globals contains global configuration shared across all commands
type Globals struct {
	Config  string
	Key     string
	Logger  zerolog.Logger
	session *core.Session // Cached session
}

// NewGlobals creates a new Globals instance with proper logger setup
func NewGlobals(config, key string, verbose bool) *Globals {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var logger zerolog.Logger
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
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
		Key:    key,
		Logger: logger,
	}
}

// Session returns a cached session, creating it if needed
func (g *Globals) Session() (*core.Session, error) {
	if g.session != nil {
		return g.session, nil
	}

	var err error
	g.session, err = core.NewSession(g.Config, g.Key)
	if err != nil {
		return nil, err
	}

	return g.session, nil
}

// Context returns a context with the logger attached
func (g *Globals) Context() context.Context {
	return g.Logger.WithContext(context.Background())
}
