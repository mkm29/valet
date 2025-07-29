# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Valet is a CLI tool that generates JSON Schema definitions from Helm chart `values.yaml` files. It automatically infers types, preserves defaults, and intelligently handles Helm components with enabled flags.

## Key Development Commands

```bash
# Build the project
make build

# Run tests with coverage report (HTML output at ./cover.html)
make test

# Run tests and verify coverage meets thresholds (70% file, 80% package, 85% total)
make check-coverage

# Clean build artifacts
make clean

# Run a single test
go test -run TestName ./tests/

# Generate schema for a Helm chart
./bin/valet generate charts/mychart

# Generate with override values
./bin/valet generate --overrides override.yaml charts/mychart

# Run with telemetry enabled
./bin/valet generate --telemetry-enabled --telemetry-exporter stdout charts/mychart
```

## Architecture Overview

### Core Components

1. **Command Structure** (`cmd/`):
   - Commands use Cobra framework with shared configuration
   - Each command in its own file (generate.go, version.go)
   - Root command handles global flags and configuration

2. **Internal Packages** (`internal/`):
   - `config/`: Configuration management with cascade (CLI > ENV > file > defaults)
   - `telemetry/`: OpenTelemetry integration for tracing, metrics, and logging

3. **Schema Generation Logic**:
   - Recursive type inference from YAML values
   - Special handling for Helm patterns (components with `.enabled` flags)
   - Preserves defaults and handles empty values intelligently
   - Merges override files when specified

4. **Testing** (`tests/`):
   - All tests use Testify suite pattern with `ValetTestSuite`
   - Test data in `testdata/` directory
   - Coverage requirements enforced in CI

### Configuration Precedence

1. CLI flags (highest priority)
2. Environment variables (`VALET_*`)
3. Configuration file (`.valet.yaml`)
4. Default values

### Key Patterns

- **Error Handling**: Clean error messages without usage printing
- **Logging**: Structured logging with zap (zero-allocation)
- **Telemetry**: Optional but comprehensive with OpenTelemetry
- **Graceful Shutdown**: Context cancellation on signals
- **Path Security**: Sanitized paths in telemetry output

## Working with the Codebase

### Adding New Commands
1. Create new file in `cmd/` directory
2. Follow existing command pattern (see `generate.go`)
3. Register command in `cmd/root.go`
4. Add tests in `tests/` directory

### Testing Guidelines
- Write tests in `tests/` directory using `ValetTestSuite`
- Use test fixtures from `testdata/`
- Ensure coverage meets thresholds
- Run `make check-coverage` before committing

### OpenTelemetry Integration
- Tracing spans for all major operations
- Metrics for performance monitoring
- Structured logging with correlation
- Multiple exporters supported (stdout, OTLP)

### Release Process
- GoReleaser handles releases on tag push
- GitHub Actions runs coverage checks on PRs
- Claude Code review integration available