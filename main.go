package main

import (
	"os"

	"github.com/thunderbottom/kiln/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
