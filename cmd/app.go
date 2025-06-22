package cmd

import (
	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"go.uber.org/zap"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// appContextKey is the context key for storing the App instance
const appContextKey contextKey = "app"

// App holds the application dependencies
type App struct {
	Config    *config.Config
	Telemetry *telemetry.Telemetry
	Logger    *zap.Logger
}

// NewApp creates a new App instance
func NewApp() *App {
	return &App{}
}

// WithConfig sets the config
func (a *App) WithConfig(cfg *config.Config) *App {
	a.Config = cfg
	return a
}

// WithTelemetry sets the telemetry
func (a *App) WithTelemetry(tel *telemetry.Telemetry) *App {
	a.Telemetry = tel
	return a
}

// WithLogger sets the logger
func (a *App) WithLogger(logger *zap.Logger) *App {
	a.Logger = logger
	return a
}
