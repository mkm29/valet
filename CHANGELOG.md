# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.2.0] - 2025-05-17

### Added

Integrated [Cobra](https://github.com/spf13/cobra) for CLI command framework
Introduced `generate` and `version` subcommands for schema generation and build info
Added `--config-file` flag to specify a configuration file path (default: `.schemagen.yaml`)
Added `--debug` flag for enabling verbose debug logging
Defaulted `--context` flag to the current directory (`.`)

### Changed

Replaced the previous flag-based CLI interface in `main.go` with Cobra-based commands
Renamed root command from `valet` to `schemagen`
Simplified configuration loading: removed Viper dependency and environment-variable support
Now reads YAML config via `--config-file` and applies CLI flags (`--context`, `--overrides`, `--output`, `--debug`) as overrides

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

[v0.1.1]: https://github.com/mkm29/valet/releases/tag/v0.1.1
[0.1.0]: https://github.com/mkm29/valet/releases/tag/v0.1.0
[v0.2.0]: https://github.com/mkm29/valet/releases/tag/v0.2.0