//go:build notelemetry
// +build notelemetry

package telemetry

import (
	"context"

	"go.uber.org/zap"
)

// Logger is a no-op logger wrapper when telemetry is disabled
type Logger struct {
	logger *zap.Logger
}

// NewLogger returns a no-op logger
func NewLogger(debug bool) (*Logger, error) {
	return &Logger{logger: zap.NewNop()}, nil
}

// SetDefault is a no-op
func (l *Logger) SetDefault() {}

// WithContext returns the logger unchanged
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	return l.logger
}

// Sync is a no-op
func (l *Logger) Sync() error {
	return nil
}

// Debug is a no-op
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {}

// Info is a no-op
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {}

// Warn is a no-op
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {}

// Error is a no-op
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {}

// DPanic is a no-op
func (l *Logger) DPanic(ctx context.Context, msg string, fields ...zap.Field) {}

// Panic is a no-op
func (l *Logger) Panic(ctx context.Context, msg string, fields ...zap.Field) {}

// Fatal is a no-op
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {}

// WithError returns the logger unchanged
func (l *Logger) WithError(err error) *zap.Logger {
	return l.logger
}