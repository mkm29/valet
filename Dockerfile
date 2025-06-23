# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user for build
RUN adduser -D -u 10001 appuser

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags for smaller binary size and security
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o valet \
    .

# Final stage using Chainguard static image
FROM cgr.dev/chainguard/static:latest

# Set user to non-root (65532 is nonroot user in Chainguard images)
USER 65532

# Copy the binary from builder
COPY --from=builder /build/valet /valet

# Copy timezone data for time operations
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy SSL certificates for HTTPS operations
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set the entrypoint
ENTRYPOINT ["/valet"]

# Default command (can be overridden)
CMD ["--help"]