package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version, build information, and runtime details for kiln.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kiln %s\n", version)

		if IsVerbose() {
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built: %s\n", date)
			fmt.Printf("  go: %s\n", runtime.Version())
			fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

			// Print build info if available
			if info, ok := debug.ReadBuildInfo(); ok {
				fmt.Printf("  module: %s\n", info.Path)
				for _, setting := range info.Settings {
					if setting.Key == "vcs.revision" {
						fmt.Printf("  revision: %s\n", setting.Value)
					}
					if setting.Key == "vcs.time" {
						fmt.Printf("  time: %s\n", setting.Value)
					}
					if setting.Key == "vcs.modified" && setting.Value == "true" {
						fmt.Printf("  modified: true\n")
					}
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
