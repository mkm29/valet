package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TelemetryTestSuite struct {
	ValetTestSuite
}

func (suite *TelemetryTestSuite) TestInitializeAndShutdown() {
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
		suite.Run(tt.name, func() {
			ctx := context.Background()
			tel, err := telemetry.Initialize(ctx, tt.cfg)

			if tt.wantErr {
				suite.Error(err)
				return
			}

			suite.NoError(err)
			suite.NotNil(tel)

			if tt.cfg.Enabled {
				suite.NotNil(tel.Tracer())
				suite.NotNil(tel.Meter())
				suite.NotNil(tel.Logger())
			} else {
				// When disabled, providers should return no-op versions
				suite.NotNil(tel.Tracer())
				suite.NotNil(tel.Meter())
				suite.NotNil(tel.Logger())
			}

			// Test shutdown (should work for both enabled and disabled)
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err = tel.Shutdown(shutdownCtx)
			// Ignore sync errors in tests
			if err != nil && !strings.Contains(err.Error(), "sync") {
				suite.NoError(err)
			}
		})
	}
}

func (suite *TelemetryTestSuite) TestTelemetryDisabled() {
	cfg := &config.TelemetryConfig{
		Enabled: false,
	}

	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)

	suite.NoError(err)
	suite.NotNil(tel)
	suite.False(tel.IsEnabled())
}

func (suite *TelemetryTestSuite) TestTelemetryShutdownTimeout() {
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "stdout",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)
	suite.NoError(err)
	suite.NotNil(tel)

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

func (suite *TelemetryTestSuite) TestTracerNilTelemetry() {
	var tel *telemetry.Telemetry
	tracer := tel.Tracer()
	suite.NotNil(tracer) // Should return a no-op tracer
}

func (suite *TelemetryTestSuite) TestMeterNilTelemetry() {
	var tel *telemetry.Telemetry
	meter := tel.Meter()
	suite.NotNil(meter) // Should return a no-op meter
}

func (suite *TelemetryTestSuite) TestLoggerNilTelemetry() {
	var tel *telemetry.Telemetry
	logger := tel.Logger()
	suite.NotNil(logger) // Should return global logger
}

func (suite *TelemetryTestSuite) TestRecordError() {
	// This test ensures RecordError doesn't panic with nil telemetry
	ctx := context.Background()

	// Should not panic
	suite.NotPanics(func() {
		telemetry.RecordError(ctx, assert.AnError)
	})
}

func TestTelemetrySuite(t *testing.T) {
	suite.Run(t, new(TelemetryTestSuite))
}
