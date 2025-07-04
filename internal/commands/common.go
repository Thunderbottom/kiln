package commands

import "github.com/thunderbottom/kiln/internal/core"

// Globals contains global configuration shared across all commands
type Globals struct {
	Config  string
	Verbose bool
}

func loadEnvVars(globals *Globals, file string) (map[string]string, error) {
	return core.LoadEnvVars(globals.Config, file)
}
