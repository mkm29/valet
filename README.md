# Valet: Helm Values to JSON Schema

![Valet Logo](./images/valet.png)

## Fast. Flexible. Clean

<!-- GitHub Actions release status -->
[![Release](https://github.com/mkm29/valet/actions/workflows/release.yml/badge.svg)](https://github.com/mkm29/valet/actions/workflows/release.yml)
[![Coverage](https://github.com/mkm29/valet/actions/workflows/coverage.yml/badge.svg)](https://github.com/mkm29/valet/actions/workflows/coverage.yml)

A command-line tool to generate a JSON Schema from a YAML `values.yaml` file, optionally merging an overrides file. Useful for Helm chart values and other YAML-based configurations.

## Requirements

- Go 1.23 or later

## Installation

Clone the repository and build:

  ```bash
  git clone https://github.com/mkm29/valet.git
  cd valet
  go build -o bin/valet main.go
  ```

Alternatively, install it directly (requires Go modules support):

  ```bash
  go install github.com/mkm29/valet@latest
  ```

## Configuration

The CLI supports a YAML configuration file (default: `.valet.yaml`) in the current directory. Use the `--config-file` flag to specify a custom path. The following keys are supported:

- `context`: directory containing `values.yaml`
- `overrides`: path to an overrides YAML file
- `output`: name of the output schema file (default: `values.schema.json`)
- `debug`: enable debug logging (boolean)

Values can also be set via environment variables (`valet_CONTEXT`, `valet_OVERRIDES`, etc.) and are overridden by CLI flags.

## Makefile

A Makefile is provided with common development tasks:

- `make help`: Show available commands (default when running `make`).
- `make build`: Build the CLI (outputs `bin/valet`).
- `make test`: Run tests, generate `cover.out` and `cover.html`.
- `make check-coverage`: Install and run `go-test-coverage` to enforce coverage thresholds defined in `.testcoverage.yml`.
- `make clean`: Remove build artifacts (`bin/` and `valet`).

Make sure you have [GNU Make](https://www.gnu.org/software/make/) installed.

## Usage

Generate a JSON Schema from a `values.yaml` in the given `<context-dir>` using the `generate` command:

```console
valet [global options] generate [flags] <context-dir>

Global options:
  --config-file string   config file path (default: .valet.yaml)
  -d, --debug            enable debug logging

Generate flags:
  -f, --overrides string   path (relative to context dir) to an overrides YAML file (optional)
  -o, --output string      output file (default: values.schema.json)
```

The tool writes a `values.schema.json` (or custom output file) in the `<context-dir>`.

### Examples

Generate schema from a directory containing `values.yaml`:

  ```bash
  ./bin/valet generate charts/mychart
  ```

Generate schema merging an override file:

  ```bash
  ./bin/valet generate --overrides override.yaml charts/mychart
  ```

Print version/build information:

  ```bash
  ./bin/valet version
  ```

Output format:

  ```text
  github.com/mkm29/valet@v0.1.1 (commit 9153c14b9ffddeaccba93268a0851d5da0ae8cbf)
  ```

## Example

Given a `values.yaml`:

  ```yaml
  replicaCount: 3
  image:
    repository: nginx
    tag: stable
  env:
    - name: LOG_LEVEL
      value: debug
  ```

Run the `generate` command:

  ```bash
  ./bin/valet generate .
  ```

Produces `values.schema.json` with contents:

  ```bash
  ./bin/valet generate .
  ```

Produces `values.schema.json` with contents:

  ```json
  {
    "$schema": "http://json-schema.org/schema#",
    "type": "object",
    "properties": {
      "replicaCount": {
        "type": "integer",
        "default": 3
      },
      "image": {
        "type": "object",
        "properties": {
          "repository": {
            "type": "string",
            "default": "nginx"
          },
          "tag": {
            "type": "string",
            "default": "stable"
          }
        },
        "default": {}
      },
      "env": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": {
              "type": "string",
              "default": "LOG_LEVEL"
            },
            "value": {
              "type": "string",
              "default": "debug"
            }
          },
          "default": {}
        },
        "default": []
      }
    },
    "required": ["replicaCount", "image", "env"]
  }
  ```

## How it works

1. Load configuration from the file specified by `--config-file` (default: `.valet.yaml`), environment variables, and CLI flags
2. Load `values.yaml` in the specified directory
3. Merge an overrides YAML if the `--overrides` flag is provided
4. Recursively infer JSON Schema types and defaults
5. Write `values.schema.json` (or custom output file) in the same directory
 
## Release

This project uses [GoReleaser](https://goreleaser.com) to automate builds and releases. Binaries for Linux and macOS (amd64 and arm64) are built when tags (e.g., `v0.1.0`) are pushed.

- A GitHub Actions workflow (`.github/workflows/release.yml`) runs GoReleaser on push tags and via manual dispatch.
- **Note**: The release workflow sets `permissions.contents: write` so that the `GITHUB_TOKEN` has sufficient permissions to create releases.
- To run a local release:

  ```bash
  go install github.com/goreleaser/goreleaser@latest
  goreleaser release --rm-dist
  ```

## Testing & Coverage

You can also use the Makefile to run tests and check coverage:

  ```bash
  make test
  make check-coverage
  ```

To run the test suite:

```bash
  go test ./...
```

To generate a coverage report:

  ```bash
  go test -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out
  ```

To view an HTML coverage report:

  ```bash
  go tool cover -html=coverage.out
  ```

## Contributing

Contributions are welcome! Feel free to open issues and submit pull requests.