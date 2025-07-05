package logging_test

import (
	"testing"

	"github.com/mkm29/valet/internal/logging"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name string
		opts logging.LoggerOptions
	}{
		{
			name: "default options",
			opts: logging.DefaultOptions(),
		},
		{
			name: "debug logger with slog",
			opts: logging.LoggerOptions{
				Backend:   logging.BackendSlog,
				Level:     "debug",
				Format:    "text",
				AddSource: true,
			},
		},
		{
			name: "production logger with slog",
			opts: logging.LoggerOptions{
				Backend:   logging.BackendSlog,
				Level:     "info",
				Format:    "json",
				AddSource: false,
			},
		},
		{
			name: "with component",
			opts: logging.LoggerOptions{
				Backend:   logging.BackendSlog,
				Level:     "info",
				Format:    "json",
				Component: "test-component",
			},
		},
		{
			name: "empty backend defaults to slog",
			opts: logging.LoggerOptions{
				Level:  "info",
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(tt.opts)
			assert.NotNil(t, logger)
			assert.True(t, logger.Enabled())
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewDebugLogger", func(t *testing.T) {
		logger := logging.NewDebugLogger()
		assert.NotNil(t, logger)
		assert.True(t, logger.Enabled())
	})

	t.Run("NewProductionLogger", func(t *testing.T) {
		logger := logging.NewProductionLogger()
		assert.NotNil(t, logger)
		assert.True(t, logger.Enabled())
	})
}

func TestLoggerLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			opts := logging.LoggerOptions{
				Backend: logging.BackendSlog,
				Level:   level,
				Format:  "json",
			}
			logger := logging.NewLogger(opts)
			assert.NotNil(t, logger)
		})
	}
}
