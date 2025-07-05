package main

import (
	"context"
	"fmt"

	"github.com/mkm29/valet/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	// Example 1: Simple logging without telemetry
	fmt.Println("=== Simple Backend Example ===")
	simpleLogger := telemetry.NewLoggerFromOptions(telemetry.LoggerOptions{
		Backend:   telemetry.BackendSimple,
		Level:     "info",
		Format:    "json",
		Component: "example",
	})
	simpleLogger.Info("This is a simple log message", "key", "value")

	// Example 2: Telemetry-aware logging
	fmt.Println("\n=== Telemetry Backend Example ===")
	telemetryLogger := telemetry.NewLoggerFromOptions(telemetry.LoggerOptions{
		Backend:   telemetry.BackendTelemetry,
		Level:     "info",
		Format:    "json",
		Component: "example",
	})

	// Create a span to demonstrate telemetry integration
	tracer := otel.Tracer("example")
	ctx, span := tracer.Start(ctx, "example-operation")
	defer span.End()

	// This log will be added as an event to the span
	telemetryLogger.Info("This log is added to the span", "operation", "demo")

	// Example 3: Using helper functions
	fmt.Println("\n=== Helper Functions Example ===")
	debugLogger := telemetry.NewDebugLogger()
	debugLogger.V(1).Info("Debug message", "verbose", true)

	prodLogger := telemetry.NewProductionLogger()
	prodLogger.Info("Production log", "environment", "prod")

	telLogger := telemetry.NewTelemetryLogger("myapp")
	telLogger.Info("App-specific telemetry log", "version", "1.0.0")

	// Example 4: Context-aware logging
	fmt.Println("\n=== Context-Aware Logging Example ===")
	contextLogger := telemetry.NewLogrWithContext(telemetryLogger)

	// In a real span context, this would add trace_id and span_id
	ctxLogger := contextLogger.WithContext(ctx)
	ctxLogger.Info("Log with trace context", "user", "john.doe")
}
