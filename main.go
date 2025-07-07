// Package main provides the kiln CLI tool for secure environment variable management.
package main

import (
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/thunderbottom/kiln/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// CLI represents the command-line interface structure for the kiln tool.
type CLI struct {
	Config  string `short:"c" help:"Configuration file path" default:"kiln.toml" type:"path"`
	Key     string `short:"k" help:"Path to private key file" default:"~/.kiln/kiln.key" type:"path"`
	Verbose bool   `short:"v" help:"Verbose output" default:"false"`

	Init    commands.InitCmd   `cmd:"" help:"Initialize new kiln project"`
	Edit    commands.EditCmd   `cmd:"" help:"Edit encrypted environment variables"`
	Export  commands.ExportCmd `cmd:"" help:"Export environment variables"`
	Run     commands.RunCmd    `cmd:"" help:"Run command with encrypted environment"`
	Set     commands.SetCmd    `cmd:"" help:"Set an environment variable"`
	Get     commands.GetCmd    `cmd:"" help:"Get an environment variable"`
	Rekey   commands.RekeyCmd  `cmd:"" help:"Rotate encryption keys"`
	Info    commands.InfoCmd   `cmd:"" help:"Show project and file information"`
	Version kong.VersionFlag   `help:"Show version"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("kiln"),
		kong.Description("Secure environment variable management tool"),
		kong.Vars{"version": fmt.Sprintf("%s (%s, built %s)", version, commit, date)},
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	globals, err := commands.NewGlobals(cli.Config, cli.Key, cli.Verbose)
	if err != nil {
		ctx.Fatalf("failed to initialize: %v", err)
	}

	err = ctx.Run(globals)
	ctx.FatalIfErrorf(err)
}
