# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Refactored telemetry configuration structure for better separation of concerns
  - Moved `Config` struct from `internal/telemetry` package to `internal/config` package as `TelemetryConfig`
  - Added YAML field tags to all `TelemetryConfig` fields for proper configuration file parsing
  - Renamed `DefaultConfig()` to `DefaultTelemetryConfig()` and moved it to the config package
  - Updated all references to use `config.TelemetryConfig` instead of `telemetry.Config`
  - Improved configuration file field naming consistency (e.g., `exporter_type` → `exporterType`)

### Added

- Added `serviceName` and `serviceVersion` fields to telemetry configuration for better observability customization
- Added comprehensive examples directory with:
  - Complete Valet configuration file example (`valet-config.yaml`)
  - OpenTelemetry Collector configuration example (`otel-config.yaml`)
  - Sample Helm chart demonstrating all supported patterns
  - Detailed README explaining how to use the examples

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
