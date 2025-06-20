package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger with OpenTelemetry integration
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new zap logger with OpenTelemetry integration
func NewLogger(debug bool) (*Logger, error) {
	config := zap.NewProductionConfig()

	// Set log level based on debug flag
	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Use JSON encoding for structured logs
	config.Encoding = "json"

	// Add caller information
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.FunctionKey = zapcore.OmitKey
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{Logger: logger}, nil
}

// SetDefault sets this logger as the global zap logger
func (l *Logger) SetDefault() {
	zap.ReplaceGlobals(l.Logger)
}

// WithContext returns a logger with trace information from the context
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return l.Logger
	}

	spanCtx := span.SpanContext()
	if !spanCtx.HasTraceID() {
		return l.Logger
	}

	return l.Logger.With(
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	)
}

// Debug logs a debug message with OpenTelemetry context
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Debug(msg, fields...)
	l.addSpanEvent(ctx, zap.DebugLevel, msg, fields...)
}

// Info logs an info message with OpenTelemetry context
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Info(msg, fields...)
	l.addSpanEvent(ctx, zap.InfoLevel, msg, fields...)
}

// Warn logs a warning message with OpenTelemetry context
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Warn(msg, fields...)
	l.addSpanEvent(ctx, zap.WarnLevel, msg, fields...)
}

// Error logs an error message with OpenTelemetry context
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Error(msg, fields...)
	l.addSpanEvent(ctx, zap.ErrorLevel, msg, fields...)
}

// DPanic logs a message at DPanicLevel with OpenTelemetry context
func (l *Logger) DPanic(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.DPanic(msg, fields...)
	l.addSpanEvent(ctx, zap.DPanicLevel, msg, fields...)
}

// Panic logs a message at PanicLevel with OpenTelemetry context
func (l *Logger) Panic(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Panic(msg, fields...)
	l.addSpanEvent(ctx, zap.PanicLevel, msg, fields...)
}

// Fatal logs a message at FatalLevel with OpenTelemetry context
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	logger := l.WithContext(ctx)
	logger.Fatal(msg, fields...)
	l.addSpanEvent(ctx, zap.FatalLevel, msg, fields...)
}

// addSpanEvent adds a log event to the current span
func (l *Logger) addSpanEvent(ctx context.Context, level zapcore.Level, msg string, fields ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	// Convert zap fields to OpenTelemetry attributes
	attrs := []attribute.KeyValue{
		attribute.String("log.severity", level.String()),
		attribute.String("log.message", msg),
	}

	for _, field := range fields {
		// Convert zap field to attribute
		// This is a simplified conversion - you might want to handle more types
		switch field.Type {
		case zapcore.StringType:
			attrs = append(attrs, attribute.String("log."+field.Key, field.String))
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			attrs = append(attrs, attribute.Int64("log."+field.Key, field.Integer))
		case zapcore.Float64Type, zapcore.Float32Type:
			attrs = append(attrs, attribute.Float64("log."+field.Key, float64(field.Integer)))
		case zapcore.BoolType:
			attrs = append(attrs, attribute.Bool("log."+field.Key, field.Integer == 1))
		case zapcore.ErrorType:
			if err, ok := field.Interface.(error); ok {
				attrs = append(attrs, attribute.String("log."+field.Key, err.Error()))
			}
		default:
			// For complex types, use string representation
			attrs = append(attrs, attribute.String("log."+field.Key, field.String))
		}
	}

	span.AddEvent("log", trace.WithAttributes(attrs...))
}

// WithError returns a logger with an error field
func (l *Logger) WithError(err error) *zap.Logger {
	if err == nil {
		return l.Logger
	}
	return l.With(zap.Error(err))
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
