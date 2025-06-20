package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OtelHandler is a slog.Handler that adds OpenTelemetry context
type OtelHandler struct {
	handler slog.Handler
}

// NewOtelHandler creates a new OpenTelemetry-aware slog handler
func NewOtelHandler(opts *slog.HandlerOptions) *OtelHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &OtelHandler{
		handler: slog.NewJSONHandler(os.Stdout, opts),
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *OtelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle handles the Record
func (h *OtelHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract trace information from context
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		spanCtx := span.SpanContext()
		if spanCtx.HasTraceID() {
			r.AddAttrs(
				slog.String("trace_id", spanCtx.TraceID().String()),
				slog.String("span_id", spanCtx.SpanID().String()),
			)
		}

		// Also add the log as an event to the span
		attrs := make([]attribute.KeyValue, 0, r.NumAttrs())
		attrs = append(attrs,
			attribute.String("log.severity", r.Level.String()),
			attribute.String("log.message", r.Message),
		)

		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, attribute.String("log."+a.Key, fmt.Sprint(a.Value)))
			return true
		})

		span.AddEvent("log", trace.WithAttributes(attrs...))
	}

	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments
func (h *OtelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OtelHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups
func (h *OtelHandler) WithGroup(name string) slog.Handler {
	return &OtelHandler{
		handler: h.handler.WithGroup(name),
	}
}

// Logger holds the structured logger
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger
func NewLogger(debug bool) *Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if debug {
		opts.Level = slog.LevelDebug
	}

	handler := NewOtelHandler(opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
	}
}

// SetDefault sets this logger as the default slog logger
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

// Debug logs a debug message with OpenTelemetry context
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.DebugContext(ctx, msg, args...)
}

// Info logs an info message with OpenTelemetry context
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message with OpenTelemetry context
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.WarnContext(ctx, msg, args...)
}

// Error logs an error message with OpenTelemetry context
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.ErrorContext(ctx, msg, args...)
}

// WithError adds an error to the log context
func (l *Logger) WithError(err error) *slog.Logger {
	if err == nil {
		return l.Logger
	}
	return l.With("error", err.Error())
}
