package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/thunderbottom/kiln/cmd"
)

func main() {
	// Set up graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
