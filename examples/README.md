# Valet Examples

This directory contains example configurations and a sample Helm chart to demonstrate Valet's capabilities.

## Contents

### 1. `valet-config.yaml`

Example Valet configuration file showing all available options:

- Debug settings
- Output configuration
- Comprehensive telemetry configuration (tracing, metrics, logging)

To use this configuration:

```bash
# Copy to your home directory
cp valet-config.yaml ~/.valet.yaml

# Or specify it explicitly
valet generate values.yaml --config examples/valet-config.yaml
```

### 2. `otel-config.yaml`

Example OpenTelemetry Collector configuration for receiving telemetry from Valet:

- OTLP receiver configuration
- Multiple exporters (Jaeger, Prometheus, console)
- Processing pipelines for traces and metrics
- Health check and debugging extensions

To run the collector:

```bash
# Using Docker
docker run -p 4317:4317 -p 9090:9090 \
  -v $(pwd)/otel-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

### 3. `sample-chart/`

A comprehensive Helm chart demonstrating various patterns that Valet handles:

- **Component detection**: Services with `enabled` flags (app, database, redis, monitoring)
- **Complex nested structures**: Database connection pools, monitoring configuration
- **Arrays and objects**: External services, tenants, config maps
- **Various data types**: Strings, numbers, booleans, arrays, objects
- **Empty values**: Demonstrates how Valet handles empty strings and null values

## Using the Sample Chart

Generate a JSON Schema from the sample chart:

```bash
# Basic usage
valet generate examples/sample-chart/values.yaml

# With custom output
valet generate examples/sample-chart/values.yaml -o my-schema.json

# With debug output to see type inference
VALET_DEBUG=true valet generate examples/sample-chart/values.yaml
```

## Testing Telemetry

To see Valet's telemetry in action:

1. Start the OpenTelemetry Collector:

```bash
docker run -p 4317:4317 -p 9090:9090 \
  -v $(pwd)/examples/otel-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

1. Configure Valet to send telemetry:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
```

2. Run Valet:

```bash
valet generate examples/sample-chart/values.yaml
```

3. View metrics at `http://localhost:9090/metrics` (Prometheus format)

## Schema Generation Examples

The sample chart's `values.yaml` demonstrates:

1. **Component Detection**:
   - `app.enabled`, `database.enabled`, `redis.enabled` are recognized as components
   - Their nested properties become conditional based on the enabled flag

2. **Type Inference**:
   - `replicaCount: 3` → `"type": "integer"`
   - `enabled: true` → `"type": "boolean"`
   - `name: "sample-app"` → `"type": "string"`
   - Arrays and objects are properly detected

3. **Required Fields**:
   - Fields with non-empty default values are marked as required
   - Empty strings, empty arrays, and empty objects are not required

4. **Nested Structures**:
   - Deep nesting like `database.connectionPool.maxConnections`
   - Arrays of objects like `externalServices` and `tenants`