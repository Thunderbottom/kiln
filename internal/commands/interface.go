package commands

import (
	"context"
	"io"
)

// Command represents a kiln command that can be executed
type Command interface {
	// Run executes the command with the given context
	Run(ctx context.Context) error

	// Name returns the command name
	Name() string

	// Description returns a brief description of the command
	Description() string
}

// IOStreams provides access to the standard input, output, and error streams
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// CommandContext provides shared context and dependencies for commands
type CommandContext struct {
	IOStreams   IOStreams
	ConfigPath  string
	Verbose     bool
	Debug       bool
	Version     string
	ProjectRoot string
}

// NewCommandContext creates a new command context with default values
func NewCommandContext() *CommandContext {
	return &CommandContext{
		IOStreams: IOStreams{
			In:     nil, // Will be set by Kong
			Out:    nil, // Will be set by Kong
			ErrOut: nil, // Will be set by Kong
		},
	}
}

// ValidatedCommand extends Command with input validation
type ValidatedCommand interface {
	Command
	// Validate validates the command arguments and flags
	Validate() error
}

// EncryptedCommand is for commands that work with encrypted files
type EncryptedCommand interface {
	Command
	// RequiresPrivateKey returns true if the command needs a private key
	RequiresPrivateKey() bool
}
