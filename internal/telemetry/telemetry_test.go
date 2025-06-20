package telemetry

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeAndShutdown(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.TelemetryConfig
		wantErr bool
	}{
		{
			name: "disabled telemetry",
			cfg: &config.TelemetryConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "stdout exporter",
			cfg: &config.TelemetryConfig{
				Enabled:        true,
				ExporterType:   "stdout",
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				SampleRate:     1.0,
			},
			wantErr: false,
		},
		{
			name: "none exporter",
			cfg: &config.TelemetryConfig{
				Enabled:        true,
				ExporterType:   "none",
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tel, err := Initialize(ctx, tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, tel)

			if tt.cfg.Enabled {
				assert.NotNil(t, tel.tracer)
				assert.NotNil(t, tel.meter)
				assert.NotNil(t, tel.logger)
			} else {
				// When disabled, providers should be nil
				assert.Nil(t, tel.tracer)
				assert.Nil(t, tel.meter)
				assert.Nil(t, tel.logger)
			}

			// Test shutdown (should work for both enabled and disabled)
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err = tel.Shutdown(shutdownCtx)
			// Ignore sync errors in tests
			if err != nil && !strings.Contains(err.Error(), "sync") {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTelemetryDisabled(t *testing.T) {
	cfg := &config.TelemetryConfig{
		Enabled: false,
	}

	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)

	assert.NoError(t, err)
	assert.NotNil(t, tel)
	assert.False(t, tel.IsEnabled())
	assert.Nil(t, tel.tracer)
	assert.Nil(t, tel.meter)
	assert.Nil(t, tel.logger)
}

func TestTelemetryShutdownTimeout(t *testing.T) {
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "stdout",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)

	// Create a context that times out immediately
	shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	// Give the context time to expire
	time.Sleep(10 * time.Millisecond)

	// Shutdown should handle the timeout gracefully
	err = tel.Shutdown(shutdownCtx)
	// May or may not error depending on timing, but should not panic
	_ = err
}

func TestTracerNilTelemetry(t *testing.T) {
	var tel *Telemetry
	tracer := tel.Tracer()
	assert.NotNil(t, tracer) // Should return a no-op tracer
}

func TestMeterNilTelemetry(t *testing.T) {
	var tel *Telemetry
	meter := tel.Meter()
	assert.NotNil(t, meter) // Should return a no-op meter
}

func TestLoggerNilTelemetry(t *testing.T) {
	var tel *Telemetry
	logger := tel.Logger()
	assert.NotNil(t, logger) // Should return global logger
}

func TestRecordError(t *testing.T) {
	// This test ensures RecordError doesn't panic with nil telemetry
	ctx := context.Background()

	// Should not panic
	assert.NotPanics(t, func() {
		RecordError(ctx, assert.AnError)
	})
}
