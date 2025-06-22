package tests

import (
	"context"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TelemetryMetricsTestSuite struct {
	ValetTestSuite
}

func (suite *TelemetryMetricsTestSuite) TestCommandMetrics() {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)
	suite.NoError(err)
	suite.NotNil(tel)
	defer tel.Shutdown(ctx)

	// Create command metrics
	metrics, err := tel.NewCommandMetrics()
	suite.NoError(err)
	suite.NotNil(metrics)

	// Test recording command execution without error
	suite.NotPanics(func() {
		metrics.RecordCommandExecution(ctx, "test-command", 100*time.Millisecond, nil)
	})

	// Test recording command execution with error
	suite.NotPanics(func() {
		metrics.RecordCommandExecution(ctx, "test-command", 200*time.Millisecond, assert.AnError)
	})
}

func (suite *TelemetryMetricsTestSuite) TestFileOperationMetrics() {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)
	suite.NoError(err)
	suite.NotNil(tel)
	defer tel.Shutdown(ctx)

	// Create file operation metrics
	metrics, err := tel.NewFileOperationMetrics()
	suite.NoError(err)
	suite.NotNil(metrics)

	// Test recording file read
	suite.NotPanics(func() {
		metrics.RecordFileRead(ctx, "/test/file.yaml", 1024, nil)
	})

	// Test recording file read with error
	suite.NotPanics(func() {
		metrics.RecordFileRead(ctx, "/test/file.yaml", 0, assert.AnError)
	})

	// Test recording file write
	suite.NotPanics(func() {
		metrics.RecordFileWrite(ctx, "/test/output.json", 2048, nil)
	})
}

func (suite *TelemetryMetricsTestSuite) TestSchemaGenerationMetrics() {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)
	suite.NoError(err)
	suite.NotNil(tel)
	defer tel.Shutdown(ctx)

	// Create schema generation metrics
	metrics, err := tel.NewSchemaGenerationMetrics()
	suite.NoError(err)
	suite.NotNil(metrics)

	// Test recording schema generation
	suite.NotPanics(func() {
		metrics.RecordSchemaGeneration(ctx, 50, 300*time.Millisecond, nil)
	})

	// Test recording schema generation with error
	suite.NotPanics(func() {
		metrics.RecordSchemaGeneration(ctx, 0, 100*time.Millisecond, assert.AnError)
	})
}

func (suite *TelemetryMetricsTestSuite) TestWithCommandSpan() {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := telemetry.Initialize(ctx, cfg)
	suite.NoError(err)
	suite.NotNil(tel)
	defer tel.Shutdown(ctx)

	// Test successful command execution
	err = telemetry.WithCommandSpan(ctx, tel, "test-command", func(ctx context.Context) error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	suite.NoError(err)

	// Test command execution with error
	expectedErr := assert.AnError
	err = telemetry.WithCommandSpan(ctx, tel, "failing-command", func(ctx context.Context) error {
		return expectedErr
	})
	suite.Equal(expectedErr, err)
}

func TestTelemetryMetricsSuite(t *testing.T) {
	suite.Run(t, new(TelemetryMetricsTestSuite))
}
