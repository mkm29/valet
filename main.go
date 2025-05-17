package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mkm29/valet/cmd"
)

func main() {
	// Execute the root command
	log.Println("Starting valet")
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
