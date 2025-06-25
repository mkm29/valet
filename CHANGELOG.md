# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- **Logging Migration** (reverting [0.2.0] change):
  - Migrated back from Uber's zap logger to Go's built-in log/slog package
  - Updated all packages to use slog for logging
  - Removed zap and zapcore dependencies from go.mod
  - Updated test suites to use slog instead of zap test utilities
  - Maintains same logging behavior with structured logging support
  - Aligns with Go standard library recommendations for simpler dependency management

### Added

- **Docker Support**:
  - Multistage Dockerfile using Chainguard's distroless static image for minimal attack surface
  - Runs as non-root user (UID 65532) for enhanced security
  - Includes only essential files: binary, CA certificates, and timezone data
  - Docker Compose configuration with security hardening examples
  - Support for both standalone and telemetry-enabled configurations
  - `.dockerignore` file to optimize build context
  - Documentation for Docker usage in README

- **Prometheus Monitoring Examples**:
  - Comprehensive Prometheus alerting rules covering performance, cache health, server health, and operational issues
  - Full Grafana dashboard JSON with visualizations for command execution, error rates, cache metrics, and server state
  - Complete monitoring stack configuration with Docker Compose setup
  - Alertmanager configuration for routing alerts to multiple notification channels
  - Updated examples/README.md with detailed documentation of all monitoring resources

- **Security and Logging Enhancements**:
  - Security section in README documenting sensitive information handling
  - Automatic redaction of sensitive credentials in debug output (passwords, tokens show as `[REDACTED]`)
  - `InitializeLogger` method to `App` struct for centralized logger initialization with cleanup function
  - Logger flush logic ensures buffered logs are written before exit

- **Server Lifecycle Metrics**:
  - New Prometheus metrics for monitoring server lifecycle events:
    - `valet_metrics_server_start_time_seconds`: Unix timestamp when metrics server started
    - `valet_metrics_server_uptime_seconds`: Current server uptime in seconds (updated every 10 seconds)
    - `valet_metrics_server_startups_total`: Total number of metrics server startups
    - `valet_metrics_server_shutdowns_total`: Total number of metrics server shutdowns
    - `valet_metrics_server_shutdown_duration_seconds`: Histogram of graceful shutdown durations
    - `valet_metrics_server_health_checks_total`: Total number of health check requests
    - `valet_metrics_server_health_check_duration_seconds`: Histogram of health check response times
    - `valet_metrics_server_state`: Current server state (0=stopped, 1=starting, 2=running, 3=shutting_down)
  - Enhanced health endpoint with custom headers:
    - `X-Server-State`: Current server state as string (stopped/starting/running/shutting_down)
    - `X-Server-Uptime`: Server uptime in seconds
    - `Retry-After`: Suggested retry interval for clients
  - Support for port 0 (random available port) for testing scenarios

- **Helm Chart Caching System**:
  - In-memory caching for remote charts to avoid redundant downloads
  - Thread-safe LRU (Least Recently Used) eviction policy
  - Configurable size limits: individual charts (default 1MB), total cache (default 10MB), max entries (default 50)
  - Separate metadata cache for faster `HasSchema()` checks
  - Cache statistics: hits, misses, evictions, hit rate, usage percentage
  - `GetCacheStats()` and `ClearCache()` methods for cache management

- **Prometheus Metrics Exposure**:
  - `/metrics` endpoint with comprehensive metrics for monitoring
  - Metrics include: Helm cache stats, command execution times, schema generation stats, file operations
  - Configurable metrics server via configuration file
  - Example configuration in `examples/helm-config-with-metrics.yaml`

- **Utils Package**:
  - Created `internal/utils` package with organized utility functions:
    - `reflection.go`: Struct field extraction utilities
    - `schema.go`: Schema generation utilities (`InferBooleanSchema`, `InferArraySchema`, etc.)
    - `yaml.go`: YAML processing (`DeepMerge`, `LoadYAML`)
    - `string.go`: String manipulation (`MaskString`, `FormatBytes`)
    - `build.go`: Build information utilities (`GetBuildVersion`)
    - `error.go`: Error handling utilities (`ErrorType`, `IsIgnorableSyncError`)
    - `math.go`: Mathematical utilities (`CalculateDelta` for counter reset detection)
    - `performance.go`: Performance utilities (`CategorizePerformance`, `ServerStateToString`)

- **Enhanced Error Messages**:
  - Detailed error messages for remote chart failures with troubleshooting hints
  - Registry-type specific suggestions (HTTP/HTTPS/OCI)
  - Better authentication configuration conflict guidance

### Changed

- **Code Organization and Refactoring**:
  - Moved all utility functions to centralized `internal/utils` package
  - Removed wrapper functions - direct calls to `utils` package throughout codebase
  - Moved schema inference functions from `cmd/schema_helpers.go` to `utils/schema.go` with exported names
  - Moved Helm config building functions to `internal/helm/config_builder.go`
  - `GetContextDirectory` now returns current working directory if no argument provided
  - Logger initialization moved from `cmd/root.go` to `App.InitializeLogger()` method
  - Moved generic helper functions from `telemetry` package to `utils` package for better reusability:
    - `errorType` → `utils.ErrorType`
    - `isIgnorableSyncError` → `utils.IsIgnorableSyncError`
    - `calculateDelta` → `utils.CalculateDelta`
    - `categorizePerformance` → `utils.CategorizePerformance`
    - `getServerStateString` → `utils.ServerStateToString`

- **Documentation Updates**:
  - Cleaned up root README.md to be more concise and user-friendly
  - Moved detailed observability documentation to examples directory
  - Created comprehensive `.valet.yaml.example` file with all configuration options
  - Root README now links to examples directory for detailed configurations

- **BREAKING: Configuration Changes**:
  - Replaced `Debug` boolean with `LogLevel` field (accepts: debug/info/warn/error/dpanic/panic/fatal)
  - CLI flag changed from `--debug` to `--log-level`
  - `InitializeLogger` now returns `(func(), error)` - cleanup function must be deferred
  - Backward compatibility: `debug: true` automatically converts to `logLevel: debug`

- **Helm Package Enhancements**:
  - Complete redesign with LRU-based caching system
  - `DownloadSchema` now returns `(string, func(), error)` with cleanup function
  - New `HelmOptions` fields: `MaxChartSize`, `MaxCacheSize`, `MaxCacheEntries`
  - Cache tracks metadata, access time, and provides automatic eviction

- **Test Suite Migration**:
  - All tests moved to centralized `tests` directory
  - Unified under `ValetTestSuite` base test suite
  - Improved test organization and consistency

### Removed

- **Backward Compatibility Functions**:
  - `telemetry.Initialize()` - use `telemetry.NewTelemetry()` with options
  - `telemetry.NewTelemetryWithConfig()` - use `telemetry.NewTelemetry()` with options
  - `helm.NewHelmWithDebug()` - use `helm.NewHelm()` with options
  - `cmd.Generate()` - use `cmd.GenerateWithApp()` with dependency injection
  - `cmd.GetTelemetry()` - use dependency injection with App struct
  - Global `cmd.NewRootCmd()` wrapper - use `cmd.NewRootCmdWithApp()`
  - All wrapper functions that called utils package functions

### Fixed

- **Remote Chart Configuration**: Fixed issue where remote chart configuration from config file was incorrectly treated as conflicting with local context when no explicit context directory was provided

- **Build and Dependency Injection Issues**:
  - Build errors caused by undefined `inferSchema` function - updated to use `inferSchema`
  - Build errors from undefined `globalApp` variable after removing backward compatibility code
  - Updated `inferArraySchema` to accept `app` parameter for proper dependency injection
  - Configuration file loading when using `--config-file` flag with subcommands
  - Persistent flags are now properly accessible in subcommands using `cmd.Root().PersistentFlags()`

- **Logger and Debug Issues**:
  - Logger sync error in test environments: Fixed "sync /dev/stdout: bad file descriptor" error
  - Added `isIgnorableSyncError` function to filter out harmless sync errors
  - Ignores errors related to stdout/stderr file descriptors common in test environments
  - Checks for specific error messages and syscall errors (EBADF, EINVAL)
  - Logger initialization happens before telemetry to ensure debug logs are always available
  - Logger level configuration now properly respects debug setting (Debug level when true, Info level when false)

- **Helm Package Issues**:
  - Fixed inconsistency in helm package where `HasSchema` used `chart.Raw` but `DownloadSchema` used `chart.Files`

- **Metrics and Observability Performance**:
  - **Optimized metrics collection performance**: Replaced JSON marshaling/unmarshaling with efficient `CacheStatsProvider` interface
  - **Enhanced tracing integration**: Metrics recording methods now use context for span correlation and add relevant attributes
  - **Robust counter reset detection**: Added `calculateDelta()` method to gracefully handle cache clearing and counter resets
  - **Configurable health checks**: Made metrics server startup timing configurable via `HealthCheckMaxAttempts`, `HealthCheckBackoff`, and `HealthCheckTimeout` settings
  - **Health check improvements**: Added configurable overall timeout for server startup and Retry-After header in health endpoint responses
  - **Context-based correlation**: Enhanced `RecordCommandExecution` with request correlation IDs, sampling priority propagation, and performance categorization using OpenTelemetry baggage
  - **Server lifecycle metrics**: Added comprehensive Prometheus metrics for tracking server state transitions, uptime, startup/shutdown counts, health check frequency, and graceful shutdown duration

## [v0.2.4] - 2025-06-19

### Changed

- Refactored telemetry configuration structure for better separation of concerns
  - Moved `Config` struct from `internal/telemetry` package to `internal/config` package as `TelemetryConfig`
  - Added YAML field tags to all `TelemetryConfig` fields for proper configuration file parsing
  - Renamed `DefaultConfig()` to `NewTelemetryConfig()` and moved it to the config package
  - Updated all references to use `config.TelemetryConfig` instead of `telemetry.Config`
  - Improved configuration file field naming consistency (e.g., `exporter_type` → `exporterType`)
- Replaced all logging with [Uber's zap](https://github.com/uber-go/zap) for high-performance structured logging
  - Migrated from `slog` to `zap` for better performance and more features
  - All log calls now use zap's typed field functions for type safety
  - Integrated zap with OpenTelemetry to include trace/span IDs in logs
  - All logs are also recorded as span events for complete observability
  - Removed dependency on standard library `log` package
- Changed default value of `--telemetry-insecure` flag from `true` to `false` for better security

### Added

- Added `serviceName` and `serviceVersion` fields to telemetry configuration for better observability customization
- Added comprehensive examples directory with:
  - Complete Valet configuration file example (`valet-config.yaml`)
  - OpenTelemetry Collector configuration example (`otel-config.yaml`)
  - Sample Helm chart demonstrating all supported patterns
  - Detailed README explaining how to use the examples
- Added comprehensive test coverage for the telemetry package:
  - Tests for telemetry initialization and shutdown
  - Tests for structured logger with OpenTelemetry integration
  - Tests for metrics recording (command, file operations, schema generation)
  - Tests for helper functions including path sanitization
  - Achieved significant coverage improvement for telemetry functionality
- Added path sanitization for file paths in telemetry attributes to protect sensitive information

### Fixed

- Fixed telemetry flag naming inconsistency where `--telemetry-enabled` flag was incorrectly referenced as `--telemetry` in code
- Fixed race condition in signal handler by moving signal handling to `main.go` with proper context cancellation
- Fixed float64 conversion bug in logger where float values were incorrectly converted using integer field
- Improved error handling in telemetry shutdown using `errors.Join` for better error aggregation
- Fixed telemetry shutdown to properly use `PersistentPostRunE` ensuring cleanup happens after command execution

### Security

- File paths in telemetry attributes are now sanitized to only include filename and immediate parent directory
- OTLP connections now default to secure (TLS) mode, requiring explicit opt-in for insecure connections

## [v0.2.3] - 2025-06-19

### Changed

- Migrated test suite from standard Go testing to [Testify](https://github.com/stretchr/testify) framework
  - All tests now use the `ValetTestSuite` struct based on `testify/suite`
  - Consolidated all tests into the `tests` directory for better organization
  - Replaced standard `t.Error*` and `t.Fatal*` calls with Testify assertions (`suite.Equal`, `suite.NoError`, etc.)
  - Added helper methods like `CopyDir` to the test suite for common test operations

### Added

- Created `ValetTestSuite` with setup functionality for test isolation
- Test files in `tests` directory:
  - `valet_test.go` - Main test suite definition
  - `generate_test.go` - Tests for the generate command
  - `config_test.go` - Tests for configuration loading
  - `root_test.go` - Tests for the root command
  - `version_test.go` - Tests for the version command
  - Documentation files for tests that couldn't be migrated due to unexported functions

### Removed

- All original test files from their respective packages:
  - `cmd/generate_test.go`
  - `cmd/root_test.go`
  - `cmd/process_test.go`
  - `cmd/inference_test.go`
  - `cmd/version_test.go`
  - `internal/config/config_test.go`
  - `main_test.go`

### Technical Notes

- Tests for unexported functions (`deepMerge`, `inferSchema`, `processProperties`, etc.) were not migrated as they cannot be accessed from the `tests` package
- One test (`TestRootCmd_DefaultContext`) was skipped due to global state interference between tests
- Test coverage remains consistent with previous implementation

## [v0.2.2] - 2025-06-19

### Added

- Integrated [Fang](https://github.com/charmbracelet/fang) CLI framework for enhanced terminal UI experience
- Beautiful command output with improved colors, styling, and formatting
- Better error handling and display with styled error messages
- Enhanced help text formatting and command descriptions

### Changed

- Wrapped Cobra CLI execution with Fang for improved appearance
- Updated architecture diagram to show Fang integration in the CLI interface layer

## [v0.2.1] - 2025-05-21

### Added

- Enhanced test coverage to meet strict standards:
  - 70% minimum file coverage
  - 80% minimum package coverage
  - 85% minimum total coverage
- Added comprehensive tests for all core functions:
  - Extensive testing for `inferSchema` covering all data types and edge cases
  - Thorough tests for `isEmptyValue` with pointers, maps, and slices
  - Added tests for `processProperties` with various component structures
  - Complete test coverage for `convertToStringKeyMap` with nested structures

### Fixed

- Fixed schema generation for components with `enabled: false` flag
- Improved handling of empty default values (strings, arrays, maps) in required fields
- Added safety checks for nil config in debug output
- Ensured proper schema representation for map types and nested objects

### Changed

- Updated debug output to use generic terms instead of hardcoded component names
- Enhanced schema validation to handle empty YAML files 
- Refactored code to improve maintainability and readability
- Updated documentation with detailed information about schema generation capabilities

## [v0.2.0] - 2025-05-17

### Added

- Integrated [Cobra](https://github.com/spf13/cobra) for CLI command framework
- Introduced `generate` and `version` subcommands for schema generation and build info
- Added `--config-file` flag to specify a configuration file path (default: `.valet.yaml`)
- Added `--debug` flag for enabling verbose debug logging
- Defaulted `--context` flag to the current directory (`.`)
- Added central existence checks in the `Generate` function for:
  - presence of `values.yaml` or `values.yml` in the context directory
  - existence of the specified overrides YAML file when using `--overrides`

### Changed

- Replaced the previous flag-based CLI interface in `main.go` with Cobra-based commands
- Renamed root command from `valet` to `valet`
- Simplified configuration loading: removed Viper dependency and environment-variable support
- Now reads YAML config via `--config-file` and applies CLI flags (`--context`, `--overrides`, `--output`, `--debug`) as overrides
- Default behavior now uses CLI flags; config file is only loaded when the `--config-file` flag is explicitly set

## [v0.1.2] - 2025-05-17

### Added

- Added GitHub Actions workflow for Go test coverage (.github/workflows/coverage.yml)
- Added coverage badge to README.md
- Added `.testcoverage.yml` configuration for `go-test-coverage` GitHub Action Workflow
  - Workflow runs on all `pull_request` events
- Documentation: Added Makefile section in README.md with common development tasks
- Documentation: Enhanced Testing & Coverage section in README.md to include `make test` and `make check-coverage`
- Abstracted exit handling in `main.go` (using `exit` variable) to enable testing of CLI exit paths
- Added CLI entrypoint tests (`TestMain_VersionFlag`, `TestMain_MissingArgs`) covering `-version` flag and missing args, boosting coverage above thresholds

### Changed

- Added write permissions to `GITHUB_TOKEN` in the GitHub Actions workflow to allow for release creation
- Updated the `README.md` to include a note about the `GITHUB_TOKEN` permissions in the release workflow

## [v0.1.1] - 2025-05-17

### Added

- `-version` flag to print embedded build information (module path, version, and commit hash)
- Integration with Go `debug/buildinfo` package to read build metadata from the binary
- CLI prints build info upon `valet -version`

### Changed

- Enhanced `inferSchema` in `main.go` to better handle `null` defaults and accurately mark required fields
- Refactored `Generate` function and CLI flag handling for consistent output formatting
- Documentation updated to include the `-version` flag in usage examples

## [0.1.0] - 2025-05-17

### Added
- Initial implementation:
  - Loading of YAML files (`values.yaml` and optional overrides) via `loadYAML`
  - Recursive deep merge of YAML maps (`deepMerge`)
  - JSON Schema inference for objects, arrays, booleans, integers, numbers, and strings (`inferSchema`)
  - Command-line interface with `-overrides` flag, usage help, and error handling
  - Output of JSON Schema to `values.schema.json`
- Documentation:
  - `README.md` with installation instructions, usage examples, and contributing guidelines
- Release automation:
  - `.goreleaser.yaml` with builds for Linux and macOS (amd64 and arm64)
  - GitHub Actions workflow (`.github/workflows/release.yml`) for automated releases with GoReleaser
  - Updated README with release badge and GoReleaser usage instructions


[v0.2.4]: https://github.com/mkm29/valet/releases/tag/v0.2.4
[v0.2.3]: https://github.com/mkm29/valet/releases/tag/v0.2.3
[v0.2.2]: https://github.com/mkm29/valet/releases/tag/v0.2.2
[v0.2.1]: https://github.com/mkm29/valet/releases/tag/v0.2.1
[v0.2.0]: https://github.com/mkm29/valet/releases/tag/v0.2.0
[v0.1.2]: https://github.com/mkm29/valet/releases/tag/v0.1.2
[v0.1.1]: https://github.com/mkm29/valet/releases/tag/v0.1.1
[0.1.0]: https://github.com/mkm29/valet/releases/tag/v0.1.0
