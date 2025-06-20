# Valet justfile
# A command-line tool to generate a JSON Schema from a YAML values.yaml file

# Use zsh for all shells
set shell := ["zsh", "-c"]

# The export setting causes all just variables to be exported as environment variables
set export := true

# Variables
gobin := env_var_or_default("GOBIN", `go env GOPATH` / "bin")

# Default recipe - show help
default:
    @just --list

# Build the project
build:
    @echo "Building valet..."
    @go build -o bin/valet main.go
    @echo "Build complete: bin/valet"

# Clean the project
clean:
    @echo "Cleaning build artifacts..."
    @rm -rf bin
    @rm -rf valet
    @rm -f cover.out cover.html
    @echo "Clean complete"

# Run the tests
test:
    @echo "Running tests with coverage..."
    @go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
    @go tool cover -html=./cover.out -o ./cover.html
    @echo "Tests complete. Coverage report: cover.html"

# Install go-test-coverage tool
install-go-test-coverage:
    @echo "Installing go-test-coverage..."
    @go install github.com/vladopajic/go-test-coverage/v2@latest
    @echo "go-test-coverage installed to {{gobin}}"

# Check the coverage against thresholds
check-coverage: install-go-test-coverage
    @echo "Running tests and checking coverage thresholds..."
    @go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
    @{{gobin}}/go-test-coverage --config=./.testcoverage.yml
    @echo "Coverage check complete"

# Run tests with verbose output
test-verbose:
    @echo "Running tests with verbose output..."
    @go test ./... -v -coverprofile=./cover.out -covermode=atomic -coverpkg=./...

# Run only unit tests
test-unit:
    @echo "Running unit tests..."
    @go test ./cmd/... ./internal/... -coverprofile=./cover.out -covermode=atomic

# Run only integration tests
test-integration:
    @echo "Running integration tests..."
    @go test ./tests/... -coverprofile=./cover.out -covermode=atomic

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    @go test -bench=. -benchmem ./cmd/...

# View coverage in terminal
coverage:
    @go test -coverprofile=./cover.out ./... -covermode=atomic -coverpkg=./...
    @go tool cover -func=./cover.out

# Install the binary to GOBIN
install: build
    @echo "Installing valet to {{gobin}}..."
    @cp bin/valet {{gobin}}/
    @echo "valet installed to {{gobin}}/valet"

# Run go mod tidy
tidy:
    @echo "Running go mod tidy..."
    @go mod tidy
    @echo "go.mod and go.sum updated"

# Run linters
lint:
    @echo "Running linters..."
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run; \
    else \
        echo "golangci-lint not installed. Install with:"; \
        echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b {{gobin}}"; \
        exit 1; \
    fi

# Format code
fmt:
    @echo "Formatting code..."
    @go fmt ./...
    @echo "Code formatted"

# Run go vet
vet:
    @echo "Running go vet..."
    @go vet ./...
    @echo "Vet complete"

# Quick check - format, vet, and test
check: fmt vet test

# Generate a schema from example values
example:
    @echo "Generating schema from example..."
    @go run main.go generate example/
    @echo "Schema generated: example/values.schema.json"

# Run the application with debug output
debug *args:
    @go run main.go --debug {{args}}

# Show version information
version:
    @go run main.go version

# Create a new release tag
release version:
    @echo "Creating release {{version}}..."
    @git tag -a {{version}} -m "Release {{version}}"
    @echo "Release tag created. Push with: git push origin {{version}}"

# Run GoReleaser in snapshot mode (dry run)
release-dry-run:
    @echo "Running GoReleaser in snapshot mode..."
    @if command -v goreleaser >/dev/null 2>&1; then \
        goreleaser release --snapshot --clean; \
    else \
        echo "goreleaser not installed. Install with:"; \
        echo "  go install github.com/goreleaser/goreleaser@latest"; \
        exit 1; \
    fi

# Update dependencies
update-deps:
    @echo "Updating dependencies..."
    @go get -u ./...
    @go mod tidy
    @echo "Dependencies updated"

# Run security audit
audit:
    @echo "Running security audit..."
    @if command -v gosec >/dev/null 2>&1; then \
        gosec ./...; \
    else \
        echo "gosec not installed. Install with:"; \
        echo "  go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
        exit 1; \
    fi

# Generate test coverage badge (requires gopherbadger)
badge:
    @echo "Generating coverage badge..."
    @if command -v gopherbadger >/dev/null 2>&1; then \
        gopherbadger -md="README.md"; \
    else \
        echo "gopherbadger not installed. Install with:"; \
        echo "  go install github.com/jpoles1/gopherbadger@latest"; \
        exit 1; \
    fi

# Development workflow - watch for changes and run tests
watch:
    @echo "Watching for changes..."
    @if command -v entr >/dev/null 2>&1; then \
        find . -name '*.go' | entr -c just test; \
    else \
        echo "entr not installed. Install with:"; \
        echo "  brew install entr  # on macOS"; \
        echo "  apt-get install entr  # on Ubuntu/Debian"; \
        exit 1; \
    fi