package commands

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// Globals contains global configuration shared across all commands
type Globals struct {
	Config  string
	Verbose bool
	Logger  *slog.Logger
}

// NewGlobals creates a new Globals instance with proper logger setup
func NewGlobals(config string, verbose bool) *Globals {
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &tint.Options{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time for cleaner CLI output
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}
	logger := slog.New(tint.NewHandler(os.Stderr, opts))

	return &Globals{
		Config:  config,
		Verbose: verbose,
		Logger:  logger,
	}
}

// Context returns a context with the logger attached
func (g *Globals) Context() context.Context {
	return context.WithValue(context.Background(), "logger", g.Logger)
}
