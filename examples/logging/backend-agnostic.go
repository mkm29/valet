// Example demonstrating backend-agnostic logging in Valet
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mkm29/valet/internal/telemetry"
)

func main() {
	ctx := context.Background()

	// Example 1: Create a logger with the simple backend (basic slog)
	fmt.Println("=== Example 1: Simple Backend ===")
	simpleLogger, err := telemetry.NewLoggerWithOptions(telemetry.LoggerOptions{
		Backend:   telemetry.BackendSimple,
		Level:     "info",
		Format:    "json",
		Component: "example-app",
	})
	if err != nil {
		fmt.Printf("Error creating simple logger: %v\n", err)
		os.Exit(1)
	}

	simpleLogger.Info(ctx, "This is a message from the simple backend", "key", "value")

	// Example 2: Create a logger with the telemetry backend (OpenTelemetry integration)
	fmt.Println("\n=== Example 2: Telemetry Backend ===")
	telemetryLogger, err := telemetry.NewLoggerWithOptions(telemetry.LoggerOptions{
		Backend:   telemetry.BackendTelemetry,
		Level:     "debug",
		Format:    "text",
		Component: "telemetry-app",
		AddSource: true,
	})
	if err != nil {
		fmt.Printf("Error creating telemetry logger: %v\n", err)
		os.Exit(1)
	}

	telemetryLogger.Debug(ctx, "Debug message with telemetry integration", "debug", true)
	telemetryLogger.Info(ctx, "Info message with telemetry integration", "info", "data")

	// Example 3: Using the convenience constructor for backward compatibility
	fmt.Println("\n=== Example 3: Convenience Constructor ===")
	debugLogger, err := telemetry.NewLogger(true) // debug = true
	if err != nil {
		fmt.Printf("Error creating debug logger: %v\n", err)
		os.Exit(1)
	}

	debugLogger.Debug(ctx, "Debug logging enabled", "verbose", true)

	// Example 4: Demonstrating how to register a custom backend
	fmt.Println("\n=== Example 4: Custom Backend (Mock) ===")

	// Register a custom backend
	customBackend := &telemetry.MockLoggerBackend{
		CreateLoggerFunc: func(opts telemetry.LoggerOptions) logr.Logger {
			fmt.Printf("Custom backend called with options: %+v\n", opts)
			// In a real implementation, you would create your custom logger here
			return telemetry.NewDebugLogger()
		},
	}
	telemetry.RegisterLoggerBackend("custom", customBackend)

	// Use the custom backend
	customLogger := telemetry.NewLoggerFromOptions(telemetry.LoggerOptions{
		Backend:   "custom",
		Level:     "info",
		Format:    "json",
		Component: "custom-app",
	})

	// The logr interface is backend-agnostic
	customLogger.Info("Message from custom backend", "custom", true)

	fmt.Println("\n=== Summary ===")
	fmt.Println("This example demonstrates:")
	fmt.Println("1. Creating loggers with different backends (simple, telemetry)")
	fmt.Println("2. Using the backend-agnostic LoggerOptions")
	fmt.Println("3. Backward compatibility with the NewLogger(debug) function")
	fmt.Println("4. How to register and use custom logging backends")
	fmt.Println("\nThe logging infrastructure is now fully decoupled from any specific backend implementation!")
}
