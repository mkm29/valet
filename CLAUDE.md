# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Valet is a command-line tool that generates JSON Schema definitions from Helm chart `values.yaml` files. It automatically infers types, preserves defaults, and intelligently handles component configurations. The project emphasizes beautiful CLI UX (using Charm libraries) and comprehensive observability through OpenTelemetry.

## Essential Commands

### Build and Development
- `make build` - Build the binary to `bin/valet`
- `make test` - Run all tests with coverage reporting
- `make check-coverage` - Run tests and verify coverage meets thresholds (85% total required)
- `make clean` - Remove build artifacts
- `make help` - Display available make commands

### Running Tests
- `go test ./tests -v` - Run all tests with verbose output
- `go test ./tests -run TestName` - Run a specific test
- `go test ./tests -cover` - Run tests with coverage summary

### Valet Usage
- `./bin/valet generate <path-to-values.yaml>` - Generate JSON schema from values file
- `./bin/valet generate <path> -o schema.json` - Specify output file
- `./bin/valet version` - Display version information
- `VALET_DEBUG=true ./bin/valet generate <path>` - Run with debug logging

## Architecture and Key Patterns

### Project Structure
```
.
├── cmd/              # CLI commands (root, generate, version)
├── internal/         # Internal packages
│   ├── config/       # Configuration management
│   └── telemetry/    # OpenTelemetry integration
├── tests/            # Test suite using Testify
├── testdata/         # Test fixtures
└── bin/             # Build output (gitignored)
```

### Command Flow
1. **Entry Point**: `main.go` creates root command via `cmd.NewRootCmd()`
2. **Configuration Loading**: Hierarchy: CLI flags → env vars → config file → defaults
3. **Telemetry Initialization**: Sets up tracing, metrics, and structured logging
4. **Schema Generation**: Smart type inference with component awareness

### Schema Generation Intelligence
The generator (`cmd/generate.go`) includes special handling for:
- **Component Detection**: Recognizes structures with `enabled` fields as components
- **Type Inference**: Automatically determines JSON Schema types from YAML values
- **Empty Value Handling**: Omits fields with empty defaults from required lists
- **Recursive Processing**: Handles nested structures and arrays

### Telemetry Integration
The project uses OpenTelemetry for comprehensive observability:
- **Tracing**: All operations are traced with spans
- **Metrics**: Tracks command execution, file operations, schema generation
- **Logging**: Structured logs with trace correlation
- **Exporters**: Supports stdout and OTLP (configured via env vars)

Key telemetry environment variables:
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP collector endpoint
- `OTEL_TRACES_EXPORTER`: Trace exporter type (stdout/otlp)
- `OTEL_METRICS_EXPORTER`: Metrics exporter type (stdout/otlp)
- `OTEL_SERVICE_NAME`: Service name for telemetry

### Testing Approach
- **Framework**: Testify suite-based testing in `tests/` directory
- **Test Suite**: All tests use `ValetTestSuite` for consistency
- **Coverage Requirements**: 85% total, 80% package, 70% file (enforced in CI)
- **Test Fixtures**: Located in `testdata/` directory

### Configuration Management
Configuration sources (in order of precedence):
1. Command-line flags
2. Environment variables (prefix: `VALET_`)
3. Configuration file (`~/.valet.yaml` or specified via `--config`)
4. Default values

Configuration file example:
```yaml
output_file: "schema.json"
debug: false
override_file: "overrides.yaml"
```

## Key Dependencies
- `github.com/charmbracelet/fang`: Beautiful CLI framework
- `github.com/spf13/cobra`: Command structure
- `github.com/stretchr/testify`: Testing framework
- `gopkg.in/yaml.v2`: YAML parsing
- OpenTelemetry packages: Distributed tracing and metrics

## CI/CD Workflows
- **Coverage Check**: Runs on all PRs, enforces coverage thresholds
- **Release Process**: Automated via GoReleaser, builds for Linux/Darwin (amd64/arm64)
- **Code Review**: Claude AI integration for automated review