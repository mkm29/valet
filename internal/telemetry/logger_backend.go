package telemetry

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LoggerBackend represents the logging backend type
type LoggerBackend string

const (
	BackendSimple    LoggerBackend = "simple"    // Basic slog logging
	BackendTelemetry LoggerBackend = "telemetry" // OpenTelemetry-integrated logging
)

// LoggerBackendInterface is the interface that all logging backends must implement
type LoggerBackendInterface interface {
	// CreateLogger creates a logr.Logger with the given options
	CreateLogger(opts LoggerOptions) logr.Logger
}

// LoggerOptions configures the logger creation
type LoggerOptions struct {
	Backend   LoggerBackend // logging backend to use
	Level     string        // "debug", "info", "warn", "error"
	Format    string        // "json" or "text"
	Component string        // component name for context
	AddSource bool          // include source file information
}

// DefaultLoggerOptions returns default logger options
func DefaultLoggerOptions() LoggerOptions {
	return LoggerOptions{
		Backend:   BackendSimple,
		Level:     "info",
		Format:    "json",
		Component: "",
		AddSource: false,
	}
}

// backendRegistry holds registered logger backends
var backendRegistry = map[LoggerBackend]LoggerBackendInterface{
	BackendSimple:    &simpleBackend{},
	BackendTelemetry: &telemetryBackend{},
}

// RegisterLoggerBackend registers a new logger backend
// This allows external packages to add new logging implementations
func RegisterLoggerBackend(name LoggerBackend, backend LoggerBackendInterface) {
	backendRegistry[name] = backend
}

// NewLoggerFromOptions creates a new logr.Logger with the given options
// This function is designed to be backend-agnostic, allowing for easy
// addition of new logging backends in the future without changing the API
func NewLoggerFromOptions(opts LoggerOptions) logr.Logger {
	backend, exists := backendRegistry[opts.Backend]
	if !exists {
		// Default to simple backend if backend not found
		backend = backendRegistry[BackendSimple]
	}

	return backend.CreateLogger(opts)
}

// createCommonSlogHandler creates a consistent slog handler with the given options.
// This eliminates duplication between logger.go and logger_backend.go
func createCommonSlogHandler(opts LoggerOptions, output *os.File) slog.Handler {
	if output == nil {
		output = os.Stdout
	}

	level := parseLevel(opts.Level)

	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			case slog.MessageKey:
				return slog.Attr{Key: "message", Value: a.Value}
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
	if opts.Format == "text" {
		handler = slog.NewTextHandler(output, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(output, handlerOpts)
	}

	// Add component context if specified
	if opts.Component != "" {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("component", opts.Component),
		})
	}

	return handler
}

// simpleBackend implements LoggerBackendInterface using basic slog
type simpleBackend struct{}

// CreateLogger implements LoggerBackendInterface for simple slog backend
func (s *simpleBackend) CreateLogger(opts LoggerOptions) logr.Logger {
	handler := createCommonSlogHandler(opts, nil)
	return logr.FromSlogHandler(handler)
}

// telemetryBackend implements LoggerBackendInterface with OpenTelemetry integration
type telemetryBackend struct{}

// CreateLogger implements LoggerBackendInterface for telemetry backend
func (t *telemetryBackend) CreateLogger(opts LoggerOptions) logr.Logger {
	// Create the base handler first
	handler := t.createHandler(opts)

	// Wrap with telemetry handler
	telemetryHandler := &telemetryHandler{
		base: handler,
	}

	return logr.FromSlogHandler(telemetryHandler)
}

// createHandler creates the base slog handler
func (t *telemetryBackend) createHandler(opts LoggerOptions) slog.Handler {
	// Use the common handler creation function
	return createCommonSlogHandler(opts, nil)
}

// telemetryHandler wraps a slog.Handler to add OpenTelemetry integration
type telemetryHandler struct {
	base slog.Handler
}

// Enabled implements slog.Handler
func (h *telemetryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle implements slog.Handler with OpenTelemetry span event recording
func (h *telemetryHandler) Handle(ctx context.Context, record slog.Record) error {
	// First, let the base handler do its work
	if err := h.base.Handle(ctx, record); err != nil {
		return err
	}

	// Then add to span if available
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		// Convert log record to span event
		attrs := []attribute.KeyValue{
			attribute.String("log.severity", record.Level.String()),
			attribute.String("log.message", record.Message),
		}

		// Add all attributes from the record
		record.Attrs(func(a slog.Attr) bool {
			switch a.Value.Kind() {
			case slog.KindString:
				attrs = append(attrs, attribute.String("log."+a.Key, a.Value.String()))
			case slog.KindInt64:
				attrs = append(attrs, attribute.Int64("log."+a.Key, a.Value.Int64()))
			case slog.KindFloat64:
				attrs = append(attrs, attribute.Float64("log."+a.Key, a.Value.Float64()))
			case slog.KindBool:
				attrs = append(attrs, attribute.Bool("log."+a.Key, a.Value.Bool()))
			default:
				attrs = append(attrs, attribute.String("log."+a.Key, a.Value.String()))
			}
			return true
		})

		span.AddEvent("log", trace.WithAttributes(attrs...))
	}

	return nil
}

// WithAttrs implements slog.Handler
func (h *telemetryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &telemetryHandler{
		base: h.base.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler
func (h *telemetryHandler) WithGroup(name string) slog.Handler {
	return &telemetryHandler{
		base: h.base.WithGroup(name),
	}
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
		Backend:   BackendSimple,
	}
	return NewLoggerFromOptions(opts)
}

// NewProductionLogger creates a logr logger suitable for production
func NewProductionLogger() logr.Logger {
	opts := LoggerOptions{
		Level:     "info",
		Format:    "json",
		AddSource: false,
		Backend:   BackendSimple,
	}
	return NewLoggerFromOptions(opts)
}

// NewTelemetryLogger creates a logr logger with OpenTelemetry integration
func NewTelemetryLogger(component string) logr.Logger {
	opts := LoggerOptions{
		Level:     "info",
		Format:    "json",
		Component: component,
		Backend:   BackendTelemetry,
	}
	return NewLoggerFromOptions(opts)
}

// NewLoggerFromSlog creates a new logr.Logger from an existing slog.Logger
// This is maintained for backward compatibility but should be avoided in new code
func NewLoggerFromSlog(logger *slog.Logger) logr.Logger {
	return logr.FromSlogHandler(logger.Handler())
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
