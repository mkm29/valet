package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

// NewLogger creates a new logr.Logger backed by slog
func NewLogger(handler slog.Handler) logr.Logger {
	// Use the official logr slog bridge
	return logr.FromSlogHandler(handler)
}

// NewLoggerFromSlog creates a new logr.Logger from an existing slog.Logger
func NewLoggerFromSlog(logger *slog.Logger) logr.Logger {
	// Use the official logr slog bridge with the logger's handler
	return logr.FromSlogHandler(logger.Handler())
}

// Helper functions for creating loggers with different configurations

// NewDebugLogger creates a logr logger suitable for debug output
func NewDebugLogger() logr.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			case slog.SourceKey:
				return slog.Attr{Key: "caller", Value: a.Value}
			}
			return a
		},
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return logr.FromSlogHandler(handler)
}

// NewProductionLogger creates a logr logger suitable for production
func NewProductionLogger() logr.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			}
			return a
		},
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return logr.FromSlogHandler(handler)
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
