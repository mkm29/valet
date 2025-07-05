package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"
	noop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// MockProvider is a mock implementation of Provider for testing
type MockProvider struct {
	mu sync.Mutex

	// Tracer behavior
	TracerFunc      func(name string) trace.Tracer
	TracerCallCount int
	TracerNames     []string

	// Meter behavior
	MeterFunc      func(name string) metric.Meter
	MeterCallCount int
	MeterNames     []string

	// Logger behavior
	LoggerFunc      func() *Logger
	LoggerCallCount int

	// Shutdown behavior
	ShutdownFunc      func(ctx context.Context) error
	ShutdownCallCount int
	ShutdownError     error

	// ForceFlush behavior
	ForceFlushFunc      func(ctx context.Context) error
	ForceFlushCallCount int
	ForceFlushError     error
}

// NewMockProvider creates a new mock provider with default implementations
func NewMockProvider() *MockProvider {
	return &MockProvider{
		TracerFunc: func(name string) trace.Tracer {
			return trace.NewNoopTracerProvider().Tracer(name)
		},
		MeterFunc: func(name string) metric.Meter {
			return noop.NewMeterProvider().Meter(name)
		},
		LoggerFunc: func() *Logger {
			return nil
		},
		ShutdownFunc: func(ctx context.Context) error {
			return nil
		},
		ForceFlushFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

// Tracer returns a tracer for creating spans
func (m *MockProvider) Tracer(name string) trace.Tracer {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TracerCallCount++
	m.TracerNames = append(m.TracerNames, name)

	if m.TracerFunc != nil {
		return m.TracerFunc(name)
	}
	return trace.NewNoopTracerProvider().Tracer(name)
}

// Meter returns a meter for creating metrics
func (m *MockProvider) Meter(name string) metric.Meter {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MeterCallCount++
	m.MeterNames = append(m.MeterNames, name)

	if m.MeterFunc != nil {
		return m.MeterFunc(name)
	}
	return noop.NewMeterProvider().Meter(name)
}

// Logger returns a telemetry-aware logger
func (m *MockProvider) Logger() *Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LoggerCallCount++

	if m.LoggerFunc != nil {
		return m.LoggerFunc()
	}
	return nil
}

// Shutdown gracefully shuts down the provider
func (m *MockProvider) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShutdownCallCount++

	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx)
	}
	return m.ShutdownError
}

// ForceFlush forces pending telemetry data to be exported
func (m *MockProvider) ForceFlush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ForceFlushCallCount++

	if m.ForceFlushFunc != nil {
		return m.ForceFlushFunc(ctx)
	}
	return m.ForceFlushError
}

// Reset resets all call counts and recorded data
func (m *MockProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TracerCallCount = 0
	m.TracerNames = nil
	m.MeterCallCount = 0
	m.MeterNames = nil
	m.LoggerCallCount = 0
	m.ShutdownCallCount = 0
	m.ForceFlushCallCount = 0
}

// MockTracerProvider is a mock implementation of TracerProvider for testing
type MockTracerProvider struct {
	mu sync.Mutex

	TracerFunc      func(name string) trace.Tracer
	TracerCallCount int
	TracerNames     []string

	ShutdownFunc      func(ctx context.Context) error
	ShutdownCallCount int
	ShutdownError     error

	ForceFlushFunc      func(ctx context.Context) error
	ForceFlushCallCount int
	ForceFlushError     error
}

// NewMockTracerProvider creates a new mock tracer provider
func NewMockTracerProvider() *MockTracerProvider {
	return &MockTracerProvider{
		TracerFunc: func(name string) trace.Tracer {
			return trace.NewNoopTracerProvider().Tracer(name)
		},
		ShutdownFunc: func(ctx context.Context) error {
			return nil
		},
		ForceFlushFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

// Tracer returns a tracer with the given name
func (m *MockTracerProvider) Tracer(name string) trace.Tracer {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TracerCallCount++
	m.TracerNames = append(m.TracerNames, name)

	if m.TracerFunc != nil {
		return m.TracerFunc(name)
	}
	return trace.NewNoopTracerProvider().Tracer(name)
}

// Shutdown shuts down the tracer provider
func (m *MockTracerProvider) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShutdownCallCount++

	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx)
	}
	return m.ShutdownError
}

// ForceFlush forces pending spans to be exported
func (m *MockTracerProvider) ForceFlush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ForceFlushCallCount++

	if m.ForceFlushFunc != nil {
		return m.ForceFlushFunc(ctx)
	}
	return m.ForceFlushError
}

// MockMeterProvider is a mock implementation of MeterProvider for testing
type MockMeterProvider struct {
	mu sync.Mutex

	MeterFunc      func(name string) metric.Meter
	MeterCallCount int
	MeterNames     []string

	ShutdownFunc      func(ctx context.Context) error
	ShutdownCallCount int
	ShutdownError     error

	ForceFlushFunc      func(ctx context.Context) error
	ForceFlushCallCount int
	ForceFlushError     error
}

// NewMockMeterProvider creates a new mock meter provider
func NewMockMeterProvider() *MockMeterProvider {
	return &MockMeterProvider{
		MeterFunc: func(name string) metric.Meter {
			return noop.NewMeterProvider().Meter(name)
		},
		ShutdownFunc: func(ctx context.Context) error {
			return nil
		},
		ForceFlushFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

// Meter returns a meter with the given name
func (m *MockMeterProvider) Meter(name string) metric.Meter {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MeterCallCount++
	m.MeterNames = append(m.MeterNames, name)

	if m.MeterFunc != nil {
		return m.MeterFunc(name)
	}
	return noop.NewMeterProvider().Meter(name)
}

// Shutdown shuts down the meter provider
func (m *MockMeterProvider) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShutdownCallCount++

	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx)
	}
	return m.ShutdownError
}

// ForceFlush forces pending metrics to be exported
func (m *MockMeterProvider) ForceFlush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ForceFlushCallCount++

	if m.ForceFlushFunc != nil {
		return m.ForceFlushFunc(ctx)
	}
	return m.ForceFlushError
}
