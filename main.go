package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mkm29/valet/cmd"
)

// exitFunc allows testing exit behavior.
var exitFunc = os.Exit

func main() {
	// Create root command
	cmd := cmd.NewRootCmd()
	// Execute the root command
	if err := fang.Execute(context.TODO(), cmd); err != nil {
		os.Exit(1)
	}
}
