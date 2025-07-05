# Backend-Agnostic Logging in Valet

## Overview

The logging infrastructure in Valet has been refactored to be completely backend-agnostic, allowing for easy switching between different logging implementations without changing application code.

## Key Changes

### 1. New Logger Constructors

- **`NewLoggerWithOptions(opts LoggerOptions)`**: Primary constructor that accepts options for full backend control
- **`NewLogger(debug bool)`**: Convenience constructor that maintains backward compatibility

### 2. Backend Selection

The logging system now supports multiple backends through the `LoggerBackend` type:

```go
const (
    BackendSimple    LoggerBackend = "simple"    // Basic slog logging
    BackendTelemetry LoggerBackend = "telemetry" // OpenTelemetry-integrated logging
)
```

### 3. LoggerOptions Structure

```go
type LoggerOptions struct {
    Backend   LoggerBackend // logging backend to use
    Level     string        // "debug", "info", "warn", "error"
    Format    string        // "json" or "text"
    Component string        // component name for context
    AddSource bool          // include source file information
}
```

### 4. Custom Backend Registration

New backends can be registered dynamically:

```go
// Implement the interface
type MyCustomBackend struct{}

func (m *MyCustomBackend) CreateLogger(opts LoggerOptions) logr.Logger {
    // Create and return your custom logger
}

// Register it
telemetry.RegisterLoggerBackend("custom", &MyCustomBackend{})

// Use it
logger := telemetry.NewLoggerFromOptions(telemetry.LoggerOptions{
    Backend: "custom",
})
```

## Benefits

1. **Flexibility**: Easy to switch between logging backends without code changes
2. **Extensibility**: New backends can be added without modifying core code
3. **Testability**: Mock backends enable comprehensive testing
4. **Consistency**: All backends follow the same interface and options pattern
5. **Future-proof**: Ready for new logging libraries (zap, zerolog, etc.)

## Migration Guide

### Old Code
```go
// Tightly coupled to slog
logger, err := telemetry.NewLogger(true)
```

### New Code
```go
// Backend-agnostic with options
logger, err := telemetry.NewLoggerWithOptions(telemetry.LoggerOptions{
    Backend:   telemetry.BackendTelemetry,
    Level:     "debug",
    Format:    "text",
    Component: "my-service",
})

// Or use the backward-compatible function
logger, err := telemetry.NewLogger(true)
```

## Example Usage

See [examples/logging/backend-agnostic.go](examples/logging/backend-agnostic.go) for a complete working example.

## Testing

The new architecture includes comprehensive tests:
- `TestNewLogger`: Tests the convenience constructor
- `TestNewLoggerWithOptions`: Tests the options-based constructor
- `TestLoggerBackendAgnostic`: Verifies backend switching functionality

## Future Enhancements

The backend-agnostic design makes it easy to add new logging backends:

1. **Zap Backend**: High-performance structured logging
2. **Zerolog Backend**: Zero-allocation JSON logging
3. **Cloud Provider Backends**: AWS CloudWatch, Google Cloud Logging, Azure Monitor
4. **Custom Backends**: Organization-specific logging solutions

Simply implement the `LoggerBackendInterface` and register your backend!