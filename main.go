// Package main provides the kiln CLI tool for secure environment variable management.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/thunderbottom/kiln/internal/commands"
	"github.com/thunderbottom/kiln/internal/core"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// CLI represents the command-line interface structure for the kiln tool.
type CLI struct {
	Config  string `short:"c" help:"Configuration file path" default:"kiln.toml" type:"path" env:"KILN_CONFIG_FILE"`
	Key     string `short:"k" help:"Path to private key file" type:"path" env:"KILN_PRIVATE_KEY_FILE"`
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
		kong.NamedMapper("agepubkey", core.AgePublicKeyMapper),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	runtime, err := commands.NewRuntime(cli.Config, cli.Key, cli.Verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	exitCode := func() int {
		defer runtime.Cleanup()

		if err := ctx.Run(runtime); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)

			return 1
		}

		return 0
	}()

	os.Exit(exitCode)
}
