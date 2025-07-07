package commands

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/thunderbottom/kiln/internal/core"
)

// Globals contains global configuration shared across all commands
type Globals struct {
	Config  string
	Key     string
	Logger  zerolog.Logger
	session *core.Session
}

// NewGlobals creates a new Globals instance with proper logger setup
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

// Session returns a cached session, creating it if needed
func (g *Globals) Session() *core.Session {
	if g.session != nil {
		return g.session
	}

	var err error
	g.session, err = core.NewSession(g.Config, g.Key)
	if err != nil {
		g.Logger.Fatal().Err(err).Msg("failed to create session")
	}

	return g.session
}

// Context returns a context with the logger attached
func (g *Globals) Context() context.Context {
	return g.Logger.WithContext(context.Background())
}
