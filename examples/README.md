# Valet Examples

This directory contains example configurations, monitoring setups, and a sample Helm chart to demonstrate Valet's capabilities.

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

### 3. `helm-config.yaml` & `helm-config-with-metrics.yaml`

Example configurations for generating schemas from remote Helm charts:

- Remote chart configuration with registry settings
- Authentication options (username/password, token)
- TLS configuration
- Metrics server configuration (with-metrics variant)

### 4. Monitoring and Observability

#### `prometheus-config.yaml`

Complete Prometheus configuration for monitoring Valet:

- Scrape configurations for single and multiple instances
- Kubernetes service discovery setup
- Remote write configuration for long-term storage
- Integration with alerting rules

#### `prometheus-alerts.yaml`

Production-ready alerting rules for Valet:

- **Performance alerts**: Slow command execution, high error rates
- **Cache alerts**: Low hit rate, cache nearly full, high eviction rate
- **Server alerts**: Metrics server down, frequent restarts, slow shutdown
- **Schema generation alerts**: High failure rate, slow generation
- **File operation alerts**: I/O errors, large file warnings

#### `grafana-dashboard.json`

Comprehensive Grafana dashboard for visualizing Valet metrics:

- Command execution rate and error rate gauges
- Command duration percentiles (p50, p95)
- Helm cache hit rate and usage gauges
- Cache operations time series
- Server state and uptime indicators
- Server lifecycle events tracking

#### `valet-targets.json`

Example Prometheus service discovery file for monitoring multiple Valet instances.

#### `docker-compose.yaml`

Complete Docker Compose setup for running the entire monitoring stack:

- OpenTelemetry Collector for receiving traces and metrics
- Prometheus for metrics storage and alerting
- Grafana for visualization with pre-configured dashboards
- Alertmanager for alert routing and notifications
- Jaeger for distributed trace visualization

#### `alertmanager-config.yaml`

Example Alertmanager configuration showing:

- Alert routing based on severity and component
- Multiple notification channels (email, Slack, PagerDuty, webhooks)
- Inhibition rules to prevent alert storms
- Grouping and timing configuration

### 5. `sample-chart/`

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

## Setting Up Monitoring

### Quick Start with Docker Compose

The included `docker-compose.yaml` provides a complete monitoring stack:

```bash
# Start the monitoring stack
docker-compose up -d

# Wait for services to be ready
sleep 10

# Start Valet with metrics enabled
valet generate --config-file examples/helm-config-with-metrics.yaml
```

Access the services:

- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3000 (auto-login enabled)
- **Jaeger**: http://localhost:16686
- **Alertmanager**: http://localhost:9093

### Step-by-Step Monitoring Setup

1. **Start Valet with metrics enabled**:

```bash
valet generate --config-file examples/helm-config-with-metrics.yaml
```

2. **Start Prometheus**:

```bash
docker run -d \
  -p 9091:9090 \
  -v $(pwd)/prometheus-config.yaml:/etc/prometheus/prometheus.yml \
  -v $(pwd)/prometheus-alerts.yaml:/etc/prometheus/alerts.yml \
  prom/prometheus:latest
```

3. **Import Grafana dashboard**:

   - Access Grafana at http://localhost:3000
   - Add Prometheus data source (http://prometheus:9090)
   - Import dashboard from `grafana-dashboard.json`

### Alert Examples

The provided alerting rules monitor:

- **Performance**: Commands taking >5s, error rates >10%
- **Cache Health**: Hit rate <50%, cache >90% full
- **Server Health**: Downtime, frequent restarts, slow shutdowns
- **Operational Issues**: Schema generation failures, file I/O errors

### Customizing Alerts

Modify `prometheus-alerts.yaml` thresholds based on your requirements:

```yaml
# Example: Adjust command execution threshold
- alert: ValetSlowCommandExecution
  expr: |
    histogram_quantile(0.95, sum(rate(valet_command_duration_seconds_bucket[5m])) by (command, le)) > 10  # Changed from 5s to 10s
```

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