# Kubernetes Logs Processing Module

## Overview

This module provides a streamlined approach to processing and forwarding Kubernetes logs. It combines pod discovery with log processing capabilities to enhance log management within Kubernetes environments.

The module focuses on pod discovery and log enhancement while avoiding unnecessary filtering or data manipulation. It provides the following key features:

- **Pod Discovery**: Discovers pods based on configurable selectors and namespaces
- **Pod Name Embedding**: Optionally embeds pod names directly in log lines
- **Log Formatting**: Supports decolorizing, trimming whitespace, and deduplicating spaces
- **Annotation-Based Control**: Uses pod annotations to selectively apply processing features

## Features

### Pod Discovery
The module uses Kubernetes service discovery to find pods based on:
- Namespace filters
- Field selectors
- Label selectors

### Log Processing
The module applies a series of transformations to logs:

1. **Pod Name Embedding** (`logs.grafana.com/embed-pod`)
   - Adds pod name to log lines with configurable field name
   - Intelligent handling for both JSON and plain text logs

2. **Decolorizing** (`logs.grafana.com/decolorize`)
   - Removes ANSI color codes from log lines
   - Makes logs easier to read and reduces storage size

3. **Trimming** (`logs.grafana.com/trim`)
   - Removes leading and trailing whitespace from log lines

4. **Space Deduplication** (`logs.grafana.com/dedup-spaces`)
   - Replaces multiple consecutive spaces with a single space

## Usage

### Module Import

```alloy
import.git "consolidated_logs" {
  repository = "https://github.com/grafana/alloy-modules.git"
  revision   = "main"
  path       = "modules/kubernetes/consolidated_logs.alloy"
}
```

### Basic Setup

```alloy
consolidated_logs.default "logs" {
  forward_to = [loki.write.default.receiver]
}
```

### Advanced Configuration

```alloy
consolidated_logs.default "logs" {
  // Where to send the processed logs
  forward_to = [loki.write.default.receiver]
  
  // Pod discovery filters
  namespaces      = ["app", "monitoring"]
  label_selectors = ["app=myapp"]
  
  // Custom annotation namespace (default: logs.grafana.com)
  annotation = "my.logs.annotations"
  
  // Pod embedding config
  embed_pod_key = "__container"
}
```

### Annotation-Based Control

You can selectively enable features using pod annotations:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  annotations:
    logs.grafana.com/embed-pod: "true"
    logs.grafana.com/decolorize: "true"
    logs.grafana.com/trim: "true"
    logs.grafana.com/dedup-spaces: "true"
spec:
  containers:
  - name: app
    image: my-app:latest
```

## Arguments

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `forward_to` | `list(LogsReceiver)` | Yes | - | Destination for processed logs |
| `namespaces` | `list(string)` | No | `[]` (all namespaces) | Namespaces to search for pods |
| `field_selectors` | `list(string)` | No | `[]` | Kubernetes field selectors for pod discovery |
| `label_selectors` | `list(string)` | No | `[]` | Kubernetes label selectors for pod discovery |
| `annotation` | `string` | No | `logs.grafana.com` | Annotation namespace for log processing controls |
| `embed_pod_value` | `string` | No | `true` | Regex to enable pod name embedding |
| `embed_pod_key` | `string` | No | `__pod` | Field name used when embedding pod names |
| `decolorize_value` | `string` | No | `(?i)true` | Regex to enable log decolorizing |
| `trim_value` | `string` | No | `true` | Regex to enable log trimming |
| `dedup_value` | `string` | No | `true` | Regex to enable space deduplication |

## Exports

| Name | Type | Description |
|------|------|-------------|
| `output` | `list(map)` | Discovered pod targets (for use with other modules) |
| `receiver` | `LogsReceiver` | Entry point for the log processing pipeline |
| `embed_pod_receiver` | `LogsReceiver` | Direct access to the pod embedding stage |
| `decolorize_receiver` | `LogsReceiver` | Direct access to the decolorizing stage |
| `trim_receiver` | `LogsReceiver` | Direct access to the trimming stage |

## Comparison to Original Module

This module is a simplified version of the [Kubernetes Log Annotations Module](https://github.com/grafana/alloy-modules/tree/main/modules/kubernetes/annotations/logs.alloy). Key differences:

- **Consolidated Pipeline**: Combines multiple modules into a single declaration
- **Focused Features**: Removes features that discard or mask log content
- **Simplified Configuration**: Reduces configuration complexity while maintaining core functionality

The original module offers additional features like:
- Log level dropping (`logs.grafana.com/drop-*`)
- Data masking for sensitive information (`logs.grafana.com/mask-*`)
- JSON structure optimization (`logs.grafana.com/scrub-*`)
- Log sampling (`logs.grafana.com/sample`)

## Example Integration

```alloy
// Import the module
import.git "consolidated_logs" {
  repository = "https://github.com/grafana/alloy-modules.git"
  revision   = "main"
  path       = "modules/kubernetes/consolidated_logs.alloy"
}

// Configure Loki output
loki.write "default" {
  endpoint {
    url = env("LOKI_URL")
    basic_auth {
      username = env("LOKI_USERNAME")
      password = env("LOKI_PASSWORD")
    }
  }
  external_labels = {
    "cluster" = env("CLUSTER_NAME")
  }
}

// Configure the consolidated logs module
consolidated_logs.default "logs" {
  forward_to = [loki.write.default.receiver]
  
  // Only discover pods in these namespaces
  namespaces = ["default", "monitoring", "app"]
  
  // Only discover pods with these labels
  label_selectors = ["app.kubernetes.io/managed-by=alloy"]
}
```