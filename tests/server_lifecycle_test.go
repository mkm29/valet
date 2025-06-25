package tests

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/stretchr/testify/suite"
)

// ServerLifecycleTestSuite tests the server lifecycle metrics
type ServerLifecycleTestSuite struct {
	suite.Suite
	metricsServer *telemetry.MetricsServer
	logger        *slog.Logger
}

func (suite *ServerLifecycleTestSuite) SetupTest() {
	// Create a test logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	suite.logger = logger

	// Create metrics server with custom port to avoid conflicts
	suite.metricsServer = telemetry.NewMetricsServer(&config.MetricsConfig{
		Enabled:                true,
		Port:                   0, // Use random available port
		Path:                   "/metrics",
		HealthCheckMaxAttempts: 5,
		HealthCheckBackoff:     100 * time.Millisecond,
		HealthCheckTimeout:     2 * time.Second,
	}, logger)
}

func (suite *ServerLifecycleTestSuite) TearDownTest() {
	if suite.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = suite.metricsServer.Shutdown(ctx)
	}
}

func (suite *ServerLifecycleTestSuite) TestServerLifecycleMetrics() {
	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- suite.metricsServer.Start(ctx)
	}()

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	// Get the actual port the server is listening on
	addr := suite.metricsServer.GetAddress()
	suite.Require().NotEmpty(addr, "Server address should not be empty")

	// Test health endpoint
	// The address is already in host:port format, so we just need to add the scheme
	healthURL := "http://" + addr + "/health"
	resp, err := http.Get(healthURL)
	suite.Require().NoError(err, "Health check should succeed")
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode, "Health check should return 200 OK")

	// Check custom headers
	suite.NotEmpty(resp.Header.Get("X-Server-Uptime"), "Should have uptime header")
	suite.Equal("running", resp.Header.Get("X-Server-State"), "Server should be in running state")
	suite.NotEmpty(resp.Header.Get("Retry-After"), "Should have Retry-After header")

	// Test metrics endpoint
	metricsURL := "http://" + addr + "/metrics"
	metricsResp, err := http.Get(metricsURL)
	suite.Require().NoError(err, "Metrics endpoint should be accessible")
	defer metricsResp.Body.Close()

	suite.Equal(http.StatusOK, metricsResp.StatusCode, "Metrics endpoint should return 200 OK")

	// Shutdown the server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err = suite.metricsServer.Shutdown(shutdownCtx)
	suite.NoError(err, "Shutdown should succeed")

	// Wait for server goroutine to finish
	select {
	case <-serverErrCh:
		// Server stopped as expected
	case <-time.After(1 * time.Second):
		suite.Fail("Server did not stop within timeout")
	}
}

func TestServerLifecycleTestSuite(t *testing.T) {
	suite.Run(t, new(ServerLifecycleTestSuite))
}
