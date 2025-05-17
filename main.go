package main

import (
	"fmt"
	"os"

	"github.com/mkm29/valet/cmd"
)

// exitFunc allows testing exit behavior.
var exitFunc = os.Exit

func main() {
	// Execute the root command
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitFunc(1)
	}
}
