# schemagen: JSON Schema Generator for Helm Charts
<!-- GitHub Actions release status -->
[![Release](https://github.com/mkm29/schemagen/actions/workflows/release.yml/badge.svg)](https://github.com/mkm29/schemagen/actions/workflows/release.yml)

A command-line tool to generate a JSON Schema from a YAML `values.yaml` file, optionally merging an overrides file. Useful for Helm chart values and other YAML-based configurations.

## Requirements

- Go 1.23 or later

## Installation

Clone the repository and build:

  ```bash
  git clone https://github.com/mkm29/schemagen.git
  cd schemagen
  go build -o bin/schemagen main.go
  ```

Alternatively, install it directly (requires Go modules support):

```bash
  go install github.com/mkm29/schemagen@latest
```

## Usage

Generate a JSON Schema from a `values.yaml` in the given `<context-dir>`:

```console
  schemagen [flags] <context-dir>

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
  ./bin/schemagen charts/mychart
```

Generate schema merging an override file:

```bash
  ./schemagen -overrides override.yaml charts/mychart
```

- Print version/build information:

```bash
  ./schemagen -version
```
Output format:
```text
github.com/mkm29/schemagen@v0.1.1 (commit 9153c14b9ffddeaccba93268a0851d5da0ae8cbf)
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
./bin/schemagen .
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
- To run a local release:

  ```bash
  go install github.com/goreleaser/goreleaser@latest
  goreleaser release --rm-dist
  ```

## Testing & Coverage

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