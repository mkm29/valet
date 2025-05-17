# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.1.1] - 2025-05-17

### Added

- `-version` flag to print embedded build information (module path, version, and commit hash)
- Integration with Go `debug/buildinfo` package to read build metadata from the binary
- CLI prints build info upon `schemagen -version`

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

[v0.1.1]: https://github.com/mkm29/schemagen/releases/tag/v0.1.1
[0.1.0]: https://github.com/mkm29/schemagen/releases/tag/v0.1.0