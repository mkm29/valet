// Package logging provides a backend-agnostic logging interface.
// Currently, it uses slog as the backend, but the architecture allows
// for easy addition of other backends (e.g., zap, zerolog) without
// changing the public API.
//
// To add a new backend:
// 1. Add a new Backend constant (e.g., BackendZap = "zap")
// 2. Create a new function to convert from that backend to logr.Logger
// 3. Add a case in the NewLogger switch statement
// 4. The rest of the codebase remains unchanged
package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

// Backend represents the logging backend type
type Backend string

const (
	BackendSlog Backend = "slog"
	// Additional backends can be added here in the future
	// e.g., BackendZap Backend = "zap"
)

// LoggerOptions configures the logger creation
type LoggerOptions struct {
	Backend   Backend // logging backend to use (currently only "slog" is supported)
	Level     string  // "debug", "info", "warn", "error"
	Format    string  // "json" or "text"
	Component string  // component name for context
	AddSource bool    // include source file information
}

// DefaultOptions returns default logger options
func DefaultOptions() LoggerOptions {
	return LoggerOptions{
		Backend:   BackendSlog,
		Level:     "info",
		Format:    "json",
		Component: "",
		AddSource: false,
	}
}

// NewLogger creates a new logr.Logger with the given options
// This function is designed to be backend-agnostic, allowing for easy
// addition of new logging backends in the future without changing the API
func NewLogger(opts LoggerOptions) logr.Logger {
	switch opts.Backend {
	case BackendSlog:
		return newLoggerFromSlog(opts)
	default:
		// Default to slog for any unknown backend
		return newLoggerFromSlog(opts)
	}
}

// newLoggerFromSlog creates a logr.Logger using slog backend
func newLoggerFromSlog(opts LoggerOptions) logr.Logger {
	handler := createHandler(opts)
	return logr.FromSlogHandler(handler)
}

// NewLoggerFromSlog creates a new logr.Logger from an existing slog.Logger
// This is maintained for backward compatibility but should be avoided in new code
func NewLoggerFromSlog(logger *slog.Logger) logr.Logger {
	return logr.FromSlogHandler(logger.Handler())
}

// createHandler creates the appropriate slog handler based on options
func createHandler(opts LoggerOptions) slog.Handler {
	level := parseLevel(opts.Level)

	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			case slog.SourceKey:
				if opts.AddSource {
					return slog.Attr{Key: "caller", Value: a.Value}
				}
				return slog.Attr{}
			}
			return a
		},
	}

	var handler slog.Handler
	switch opts.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	default: // "json" or any other value defaults to JSON
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	// Add component context if specified
	if opts.Component != "" {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("component", opts.Component),
		})
	}

	return handler
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default: // "info" or any other value defaults to Info
		return slog.LevelInfo
	}
}

// Helper functions for creating loggers with different configurations

// NewDebugLogger creates a logr logger suitable for debug output
func NewDebugLogger() logr.Logger {
	opts := LoggerOptions{
		Level:     "debug",
		Format:    "text",
		AddSource: true,
	}
	return NewLogger(opts)
}

// NewProductionLogger creates a logr logger suitable for production
func NewProductionLogger() logr.Logger {
	opts := LoggerOptions{
		Level:     "info",
		Format:    "json",
		AddSource: false,
	}
	return NewLogger(opts)
}

// LogrWithContext provides a context-aware logr logger with OpenTelemetry integration
type LogrWithContext struct {
	base logr.Logger
}

// NewLogrWithContext creates a new context-aware logr logger
func NewLogrWithContext(base logr.Logger) *LogrWithContext {
	return &LogrWithContext{base: base}
}

// WithContext returns a logger that includes trace information from the context
func (l *LogrWithContext) WithContext(ctx context.Context) logr.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return l.base
	}

	spanCtx := span.SpanContext()
	if !spanCtx.HasTraceID() {
		return l.base
	}

	// Add trace context to logger
	return l.base.WithValues(
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)
}

// V returns a logger with the specified verbosity level
func (l *LogrWithContext) V(level int) logr.Logger {
	return l.base.V(level)
}

// Info logs a non-error message
func (l *LogrWithContext) Info(msg string, keysAndValues ...any) {
	l.base.Info(msg, keysAndValues...)
}

// Error logs an error message
func (l *LogrWithContext) Error(err error, msg string, keysAndValues ...any) {
	l.base.Error(err, msg, keysAndValues...)
}

// WithValues returns a logger with additional key-value pairs
func (l *LogrWithContext) WithValues(keysAndValues ...any) *LogrWithContext {
	return &LogrWithContext{base: l.base.WithValues(keysAndValues...)}
}

// WithName returns a logger with the specified name
func (l *LogrWithContext) WithName(name string) *LogrWithContext {
	return &LogrWithContext{base: l.base.WithName(name)}
}

// GetLogger returns the underlying logr.Logger
func (l *LogrWithContext) GetLogger() logr.Logger {
	return l.base
}
