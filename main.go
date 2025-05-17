package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mkm29/schemagen/cmd"
)

func main() {
	// Execute the root command
	log.Println("Starting schemagen")
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
