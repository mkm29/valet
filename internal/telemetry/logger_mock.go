package telemetry

import (
	"github.com/go-logr/logr"
)

// MockLoggerBackend is a mock implementation of LoggerBackendInterface for testing
type MockLoggerBackend struct {
	// CreateLoggerFunc can be set to customize the behavior of CreateLogger
	CreateLoggerFunc func(opts LoggerOptions) logr.Logger
	// CallCount tracks how many times CreateLogger was called
	CallCount int
	// LastOptions stores the last options passed to CreateLogger
	LastOptions LoggerOptions
}

// CreateLogger implements LoggerBackendInterface for testing
func (m *MockLoggerBackend) CreateLogger(opts LoggerOptions) logr.Logger {
	m.CallCount++
	m.LastOptions = opts

	if m.CreateLoggerFunc != nil {
		return m.CreateLoggerFunc(opts)
	}

	// Default to returning a discard logger for testing
	return logr.Discard()
}

// RegisterMockLoggerBackend registers the mock backend for testing
// This should only be used in test files
func RegisterMockLoggerBackend() *MockLoggerBackend {
	mock := &MockLoggerBackend{}
	RegisterLoggerBackend("mock", mock)
	return mock
}
