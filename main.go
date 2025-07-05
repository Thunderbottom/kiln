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

type CLI struct {
	Config  string `short:"c" help:"Configuration file path" default:"kiln.toml"`
	Verbose bool   `short:"v" help:"Verbose output"`

	Init    commands.InitCmd   `cmd:"" help:"Initialize new kiln project"`
	Edit    commands.EditCmd   `cmd:"" help:"Edit encrypted environment variables"`
	Export  commands.ExportCmd `cmd:"" help:"Export environment variables"`
	Run     commands.RunCmd    `cmd:"" help:"Run command with encrypted environment"`
	Set     commands.SetCmd    `cmd:"" help:"Set an environment variable"`
	Get     commands.GetCmd    `cmd:"" help:"Get an environment variable"`
	Rekey   commands.RekeyCmd  `cmd:"" help:"Rotate encryption keys"`
	Status  commands.StatusCmd `cmd:"" help:"Show project status"`
	Verify  commands.VerifyCmd `cmd:"" help:"Verify the access and integrity for encrypted files"`
	Version kong.VersionFlag   `help:"Show version"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("kiln"),
		kong.Description("Secure environment variable management tool"),
		kong.Vars{"version": fmt.Sprintf("%s (%s, built %s)", version, commit, date)},
	)

	globals := commands.NewGlobals(cli.Config, cli.Verbose)

	err := ctx.Run(globals)
	ctx.FatalIfErrorf(err)
}
