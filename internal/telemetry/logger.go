package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Logger wraps slog logger with OpenTelemetry integration
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new slog logger with OpenTelemetry integration
func NewLogger(debug bool) (*Logger, error) {
	// Set log level based on debug flag
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	// Create JSON handler with options
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize attribute names to match previous format
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			case slog.MessageKey:
				return slog.Attr{Key: "message", Value: a.Value}
			case slog.SourceKey:
				return slog.Attr{Key: "caller", Value: a.Value}
			}
			return a
		},
	}

	// Use JSON handler for structured logs
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}, nil
}

// SetDefault sets this logger as the global slog logger
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

// WithContext returns a logger with trace information from the context
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return l.Logger
	}

	spanCtx := span.SpanContext()
	if !spanCtx.HasTraceID() {
		return l.Logger
	}

	return l.Logger.With(
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)
}

// Debug logs a debug message with OpenTelemetry context
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	logger := l.WithContext(ctx)
	logger.DebugContext(ctx, msg, args...)
	l.addSpanEvent(ctx, slog.LevelDebug, msg, args...)
}

// Info logs an info message with OpenTelemetry context
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	logger := l.WithContext(ctx)
	logger.InfoContext(ctx, msg, args...)
	l.addSpanEvent(ctx, slog.LevelInfo, msg, args...)
}

// Warn logs a warning message with OpenTelemetry context
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	logger := l.WithContext(ctx)
	logger.WarnContext(ctx, msg, args...)
	l.addSpanEvent(ctx, slog.LevelWarn, msg, args...)
}

// Error logs an error message with OpenTelemetry context
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	logger := l.WithContext(ctx)
	logger.ErrorContext(ctx, msg, args...)
	l.addSpanEvent(ctx, slog.LevelError, msg, args...)
}

// Fatal logs a message at FatalLevel with OpenTelemetry context and exits
func (l *Logger) Fatal(ctx context.Context, msg string, args ...any) {
	logger := l.WithContext(ctx)
	logger.ErrorContext(ctx, msg, args...)
	l.addSpanEvent(ctx, slog.LevelError, msg, args...)
	os.Exit(1)
}

// addSpanEvent adds a log event to the current span
func (l *Logger) addSpanEvent(ctx context.Context, level slog.Level, msg string, args ...any) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	// Convert slog args to OpenTelemetry attributes
	attrs := []attribute.KeyValue{
		attribute.String("log.severity", level.String()),
		attribute.String("log.message", msg),
	}

	// Process args in pairs (key, value)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		value := args[i+1]

		// Convert value to attribute based on type
		switch v := value.(type) {
		case string:
			attrs = append(attrs, attribute.String("log."+key, v))
		case int:
			attrs = append(attrs, attribute.Int64("log."+key, int64(v)))
		case int64:
			attrs = append(attrs, attribute.Int64("log."+key, v))
		case int32:
			attrs = append(attrs, attribute.Int64("log."+key, int64(v)))
		case float64:
			attrs = append(attrs, attribute.Float64("log."+key, v))
		case float32:
			attrs = append(attrs, attribute.Float64("log."+key, float64(v)))
		case bool:
			attrs = append(attrs, attribute.Bool("log."+key, v))
		case error:
			attrs = append(attrs, attribute.String("log."+key, v.Error()))
		default:
			// For complex types, use string representation
			attrs = append(attrs, attribute.String("log."+key, slog.AnyValue(v).String()))
		}
	}

	span.AddEvent("log", trace.WithAttributes(attrs...))
}

// WithError returns a logger with an error field
func (l *Logger) WithError(err error) *slog.Logger {
	if err == nil {
		return l.Logger
	}
	return l.With("error", err)
}

// Sync flushes any buffered log entries
// Note: slog doesn't have a Sync method, so this is a no-op
func (l *Logger) Sync() error {
	return nil
}
