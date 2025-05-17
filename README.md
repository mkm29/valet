# Valet: Values YAML Schema Tool

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

## Makefile

A Makefile is provided with common development tasks:

- `make help`: Show available commands (default when running `make`).
- `make build`: Build the CLI (outputs `bin/valet`).
- `make test`: Run tests, generate `cover.out` and `cover.html`.
- `make check-coverage`: Install and run `go-test-coverage` to enforce coverage thresholds defined in `.testcoverage.yml`.
- `make clean`: Remove build artifacts (`bin/` and `valet`).

Make sure you have [GNU Make](https://www.gnu.org/software/make/) installed.

## Usage

Generate a JSON Schema from a `values.yaml` in the given `<context-dir>`:

```console
  valet [flags] <context-dir>

Flags:
  -overrides string
        path (relative to the context directory) to an overrides YAML file (optional)
  -version
        print version information
```

The tool writes a `values.schema.json` file in the `<context-dir>`.

### Examples

Generate schema from a directory containing `values.yaml`:

```bash
  ./bin/valet charts/mychart
```

Generate schema merging an override file:

```bash
  ./valet -overrides override.yaml charts/mychart
```

- Print version/build information:

```bash
  ./valet -version
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

Run:

```bash
./bin/valet .
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

1. Load `values.yaml` in the specified directory
2. Merge an overrides YAML if `-overrides` is provided
3. Recursively infer JSON Schema types and defaults
4. Write `values.schema.json` in the same directory
 
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