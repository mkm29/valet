# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Security section in README documenting sensitive information handling
- Automatic redaction of sensitive credentials in debug output
- **Chart caching in helm package**: Remote charts are now cached in memory to avoid redundant downloads
  - Thread-safe implementation using read-write locks
  - Cache key includes registry URL, chart name, and version
  - Both `HasSchema` and `DownloadSchema` now benefit from caching
  - Debug logging shows cache hits and misses
  - Significantly improves performance when working with the same chart multiple times
- **Size limits for chart downloads**: Charts exceeding the configured size limit are rejected before loading
  - Default limit is 1MB (matching etcd's limit)
  - Configurable via `MaxChartSize` option in `HelmOptions`
  - Size check happens after download but before loading into memory
  - Human-readable size formatting in error messages and debug logs
  - Total cache size tracking for monitoring memory usage

### Changed

- Updated documentation to reflect removal of backward compatibility code
- `Helm` struct now includes a `cache` field for storing downloaded charts and `maxChartSize` field for size limits
- `HasSchema` and `DownloadSchema` methods now use `getOrLoadChart` instead of `loadChart` directly
- `HelmOptions` now includes `MaxChartSize` field for configuring the maximum allowed chart size
- Debug logging enhanced with chart size information and total cache size tracking

### Removed

- All backward compatibility wrapper functions have been removed:
  - `telemetry.Initialize()` - use `telemetry.NewTelemetry()` with options instead
  - `telemetry.NewTelemetryWithConfig()` - use `telemetry.NewTelemetry()` with options instead
  - `helm.NewHelmWithDebug()` - use `helm.NewHelm()` with options instead
  - `cmd.Generate()` - use `cmd.GenerateWithApp()` with dependency injection instead
  - `cmd.GetTelemetry()` - use proper dependency injection with App struct instead
  - Global `cmd.NewRootCmd()` wrapper - use `cmd.NewRootCmdWithApp()` instead
- Removed references to backward compatibility in documentation
- Removed undefined `globalApp` variable and associated backward compatibility code

### Fixed

- Build errors caused by undefined `inferSchema` function - updated to use `inferSchema` 
- Build errors from undefined `globalApp` variable after removing backward compatibility code
- Updated `inferArraySchema` to accept `app` parameter for proper dependency injection

### Security

- Configuration logging now requires proper redaction of sensitive fields (passwords, tokens, etc.)
- Debug output must redact registry credentials and authentication tokens as `[REDACTED]`
- All sensitive configuration values should be masked before logging

- Dependency injection pattern for improved testability:
  - Created `App` struct to hold dependencies (Config, Telemetry, Logger)
  - Added builder pattern with `WithConfig`, `WithTelemetry`, `WithLogger` methods
  - All commands now support `WithApp` variants for dependency injection
- Support for remote Helm charts configuration in `internal/helm` package
- Helm chart configuration in config file and CLI flags for the `generate` command
  - `--chart-name`: Name of the remote Helm chart
  - `--chart-version`: Version of the remote Helm chart
  - `--registry-url`: URL of the Helm chart registry
  - `--registry-type`: Type of registry (HTTP, HTTPS, OCI) - defaults to HTTPS
  - `--registry-insecure`: Allow insecure connections
  - Authentication flags: `--registry-username`, `--registry-password`, `--registry-token`
  - TLS flags: `--registry-tls-skip-verify`, `--registry-cert-file`, `--registry-key-file`, `--registry-ca-file`
- Default values for Helm configuration structs (e.g., registry type defaults to "HTTPS")
- Validation for Helm configuration to ensure required fields are present
- Example configuration file `examples/helm-config.yaml` demonstrating remote chart usage
- CUE language support added to roadmap for future schema generation
- Pretty printing of configuration to stdout when debug mode is enabled
- Options pattern for flexible package initialization:
  - `HelmOptions` for configuring helm package instances
  - `TelemetryOptions` for configuring telemetry package instances
- Named loggers for better debugging:
  - Helm package uses `helm` logger name
  - Each package can have its own named logger
- Convenience functions for simple use cases:
  - `NewHelmWithDebug` for helm package
  - `NewTelemetryWithConfig` for telemetry package

### Changed

- **Major refactoring** to improve code maintainability and testability:
  - Refactored `inferSchema` function from 290+ lines to ~20 smaller, focused functions:
    - Type-specific handlers: `inferArraySchema`, `inferBooleanSchema`, `inferIntegerSchema`, etc.
    - Object handling: `inferObjectSchemaWithApp`, `generateObjectPropertiesWithApp`, `determineRequiredFieldsWithApp`
    - Helper functions for validation and processing
  - Refactored `NewGenerateCmd` function from 200+ lines to ~15 smaller functions:
    - Configuration parsing: `parseGenerateCommandConfigWithApp`, `getContextDirectory`
    - Validation: `validateGenerateCommandConfig`, `validateOverridesFile`
    - Command execution: `generateCmdRunWithApp`, `handleRemoteChartGenerationWithApp`
    - Flag management: `addGenerateFlags`, `applyOptionalHelmFlags`
  - Applied Single Responsibility Principle throughout the codebase
  - Improved separation of concerns with clear function boundaries
- Replaced global variables (`cfg` and `tel`) with dependency injection:
  - Global variables created hidden dependencies and made unit testing difficult
  - Now using explicit dependency passing through the `App` struct
  - Improved testability by allowing mock dependencies to be injected
  - Context key properly typed to avoid collisions (`contextKey` type)
- `generate` command now accepts optional context directory: `generate [context-dir]` instead of `generate <context-dir>`
- `generate` command validates that either a local context directory OR remote chart configuration is provided (but not both)
- Improved configuration file loading to properly detect and load config files when specified
- Logging is now independent of telemetry - logger is always initialized based on debug setting
- Debug logging is available whenever `debug: true` is set, regardless of telemetry state
- Consolidated Helm configuration structs in the `internal/config` package to avoid duplication
- Migrated helm package from standard `log` to `zap` for consistent structured logging
- Refactored packages to follow Go best practices with consistent struct-based design:
  - **Helm package**:
    - Created `Helm` struct with encapsulated logger and configuration
    - Added `NewHelm` constructor with `HelmOptions` for flexible initialization
    - Converted standalone functions to methods on the `Helm` struct
    - Added `NewHelmWithDebug` convenience function for simple use cases
    - Functions now accept debug flag for conditional logging
  - **Telemetry package**:
    - Enhanced existing struct-based design with Options pattern
    - Added `NewTelemetry` constructor with `TelemetryOptions` for flexible initialization
    - Maintained backward compatibility with existing `Initialize` function
    - Added `NewTelemetryWithConfig` convenience function
    - Consistent with helm package architecture
- Helm functions (`HasSchema`, `DownloadSchema`) now use `chart.Raw` consistently for file iteration
- All packages now follow the same architectural patterns for consistency and maintainability
- Refactored helm package to eliminate code duplication:
  - Extracted common chart loading logic into private `loadChart` method
  - Both `HasSchema` and `DownloadSchema` now use the shared `loadChart` method
  - Centralized chart downloading, getter creation, and error handling
  - Improved debug logging with consistent use of package logger
  - Reduced code duplication by ~40 lines

### Fixed

- Configuration file loading when using `--config-file` flag with subcommands
- Persistent flags are now properly accessible in subcommands using `cmd.Root().PersistentFlags()`
- Logger initialization happens before telemetry to ensure debug logs are always available
- Fixed inconsistency in helm package where `HasSchema` used `chart.Raw` but `DownloadSchema` used `chart.Files`
- Logger level configuration now properly respects debug setting (Debug level when true, Info level when false)

### Technical Notes

- The refactoring maintains 100% backward compatibility - all existing code continues to work unchanged
- Helper functions were consolidated into `cmd/schema_helpers.go` for better organization
- Context keys now use a proper type (`contextKey`) to avoid potential collisions
- All linter issues in the cmd package have been resolved

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


[v0.2.3]: https://github.com/mkm29/valet/releases/tag/v0.2.3
[v0.2.2]: https://github.com/mkm29/valet/releases/tag/v0.2.2
[v0.2.1]: https://github.com/mkm29/valet/releases/tag/v0.2.1
[v0.2.0]: https://github.com/mkm29/valet/releases/tag/v0.2.0
[v0.1.2]: https://github.com/mkm29/valet/releases/tag/v0.1.2
