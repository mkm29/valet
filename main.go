package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mkm29/valet/cmd"
)

func main() {
	// Create root command
	cmd := cmd.NewRootCmd()
	// Execute the root command
	if err := fang.Execute(context.TODO(), cmd); err != nil {
		os.Exit(1)
	}
}
