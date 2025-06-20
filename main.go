package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/mkm29/valet/cmd"
)

func main() {
	// Create a context that is canceled on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		// Cancel the context when a signal is received
		cancel()
	}()

	// Create root command
	rootCmd := cmd.NewRootCmd()

	// Execute the root command with the cancellable context
	if err := fang.Execute(ctx, rootCmd); err != nil {
		os.Exit(1)
	}
}
