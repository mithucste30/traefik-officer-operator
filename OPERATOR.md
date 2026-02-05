# Traefik Officer Operator

A Kubernetes operator for monitoring Traefik access logs with Prometheus metrics, using custom CRDs for dynamic configuration.

## Overview

Traefik Officer Operator extends the original Traefik Officer with Kubernetes-native configuration through Custom Resource Definitions (CRDs). Instead of using static configuration files, you can now define monitoring rules declaratively using Kubernetes manifests.

## Features

- **Custom Resource Definition (CRD)**: `UrlPerformance` for declarative configuration
- **Dynamic Configuration**: Update monitoring rules without restarting pods
- **Per-Ingress Configuration**: Configure different monitoring rules for each Ingress/IngressRoute
- **Enhanced Metrics Labels**: All metrics now include `namespace` and `ingress` labels
- **Prometheus ServiceMonitor**: Auto-integration with Prometheus Operator
- **Path Filtering**: Whitelist and blacklist paths using regex patterns
- **URL Normalization**: Reduce metric cardinality with custom patterns
- **Top N Path Tracking**: Track top N paths by latency per ingress
- **Multi-Provider Support**: Works with both Ingress (networking.k8s.io) and IngressRoute (traefik.io) CRDs

## Architecture

### Components

1. **Operator Controller**: Watches `UrlPerformance` CRDs and updates runtime configuration
2. **Log Processor**: Streams logs from Traefik pods or files
3. **Metrics Collector**: Exposes Prometheus metrics with enhanced labels

### Router Name Parsing

The operator intelligently parses Traefik router names to extract namespace and ingress information:

**Standard Ingress (networking.k8s.io):**
```
websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
↓
Namespace: monitoring
Ingress: grafana-operator-grafana-ingress
Provider: kubernetes
```

**IngressRoute CRD (traefik.io):**
```
mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08@kubernetescrd
↓
Namespace: mahfil-dev
IngressRoute: mahfil-api-server-ingressroute-http
Provider: kubernetescrd
```

## Installation

### Prerequisites

- Kubernetes cluster (v1.19+)
- Traefik v3.0+
- Helm 3.x

### Install via Helm

```bash
# Add Helm repository (if hosted)
helm repo add traefik-officer https://0xvox.github.io/traefik-officer
helm repo update

# Install the operator
helm install traefik-officer-operator traefik-officer/traefik-officer-operator \
  --namespace traefik-officer \
  --create-namespace

# Or install from local directory
helm install traefik-officer-operator ./helm/traefik-officer-operator \
  --namespace traefik-officer \
  --create-namespace
```

### Configuration

Configure Traefik log source in `values.yaml`:

```yaml
traefik:
  logSource: kubernetes  # or "file"
  kubernetes:
    namespace: ingress-controller
    containerName: traefik
    podLabelSelector: app.kubernetes.io/name=traefik
```

## Usage

### Create a UrlPerformance Resource

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: grafana-monitoring
  namespace: monitoring
spec:
  targetRef:
    kind: Ingress
    name: grafana-operator-grafana-ingress
    namespace: monitoring

  whitelistPathsRegex:
    - "^/api/"
    - "^/dashboard/"

  ignoredPathsRegex:
    - "^/static/"
    - "\\.css$"

  mergePathsWithExtensions:
    - "/api/datasources"

  urlPatterns:
    - pattern: "/api/datasources/uid/[a-z0-9-]+"
      replacement: "/api/datasources/uid/{uid}"

  collectNTop: 20
  enabled: true
```

### Monitor an IngressRoute

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: api-monitoring
  namespace: production
spec:
  targetRef:
    kind: IngressRoute  # Note: using IngressRoute
    name: api-server-ingressroute
    namespace: production

  ignoredPathsRegex:
    - "^/health"
    - "\\.jpg$"

  mergePathsWithExtensions:
    - "/api/"

  collectNTop: 50
  enabled: true
```

## Metrics

### Updated Metrics with Enhanced Labels

All metrics now include `namespace` and `ingress` labels for better filtering and querying:

```promql
# Request rate per namespace
sum(rate(traefik_officer_requests_total[5m])) by (namespace)

# Request rate per ingress
sum(rate(traefik_officer_requests_total[5m])) by (namespace, ingress)

# Error rate per ingress
sum(rate(traefik_officer_endpoint_requests_total{response_code=~"5.."}[5m])) by (namespace, ingress)

# P95 latency per namespace
histogram_quantile(0.95,
  sum(rate(traefik_officer_request_duration_seconds_bucket[5m])) by (namespace, le)
)

# Average latency by ingress and path
traefik_officer_endpoint_avg_latency_seconds
```

### Available Metrics

- `traefik_officer_requests_total{request_method, response_code, app, namespace, target_kind}`
- `traefik_officer_request_duration_seconds{request_method, response_code, app, namespace, target_kind}`
- `traefik_officer_endpoint_requests_total{namespace, ingress, request_path, request_method, response_code}`
- `traefik_officer_endpoint_request_duration_seconds{namespace, ingress, request_path, request_method, response_code}`
- `traefik_officer_endpoint_avg_latency_seconds{namespace, ingress, request_path}`
- `traefik_officer_endpoint_max_latency_seconds{namespace, ingress, request_path}`
- `traefik_officer_endpoint_error_rate{namespace, ingress, request_path}`
- `traefik_officer_endpoint_client_error_rate{namespace, ingress, request_path}`
- `traefik_officer_endpoint_server_error_rate{namespace, ingress, request_path}`

## CRD Specification

### UrlPerformance Spec

```yaml
spec:
  targetRef:
    kind: Ingress | IngressRoute  # Required
    name: string                  # Required
    namespace: string             # Optional, defaults to UrlPerformance namespace

  whitelistPathsRegex:           # Optional
    - string                      # Only monitor matching paths

  ignoredPathsRegex:             # Optional
    - string                      # Ignore these paths

  mergePathsWithExtensions:      # Optional
    - string                      # Merge paths under these prefixes

  urlPatterns:                   # Optional
    - pattern: string             # Regex pattern
      replacement: string         # Replacement template

  collectNTop: integer            # Optional, default 20

  enabled: boolean                # Optional, default true
```

### UrlPerformance Status

```yaml
status:
  phase: Pending | Active | Error | Disabled
  conditions:
    - type: Ready | TargetExists | ConfigGenerated
      status: "True" | "False" | "Unknown"
      lastTransitionTime: timestamp
      reason: string
      message: string
  monitoredPaths: integer
  lastScrapeTime: timestamp
  observedGeneration: integer
```

## Configuration Reference

### Helm Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image | `0xvox/traefik-officer` |
| `image.tag` | Image tag | `latest` |
| `traefik.logSource` | Log source mode | `kubernetes` |
| `traefik.kubernetes.namespace` | Traefik namespace | `ingress-controller` |
| `traefik.kubernetes.podLabelSelector` | Pod selector | `app.kubernetes.io/name=traefik` |
| `metrics.serviceMonitor.enabled` | Enable ServiceMonitor | `true` |
| `metrics.port` | Metrics port | `8084` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |

See `helm/traefik-officer-operator/values.yaml` for all options.

## Examples

See the `examples/` directory for complete examples:

- [Ingress Example](./examples/urlperformances/ingress-example.yaml)
- [IngressRoute Example](./examples/urlperformances/ingressroute-example.yaml)
- [Disabled Example](./examples/urlperformances/disabled-example.yaml)

## Troubleshooting

### Check Operator Status

```bash
# Check pod status
kubectl get pods -n traefik-officer

# View logs
kubectl logs -n traefik-officer deployment/traefik-officer-operator

# Check UrlPerformance resources
kubectl get urlperformances -A
kubectl describe urlperformance grafana-monitoring -n monitoring
```

### Verify CRD Installation

```bash
kubectl get crd urlperformances.traefikofficer.io
kubectl api-resources | grep traefikofficer
```

### Check Metrics Endpoint

```bash
# Port-forward to access metrics
kubectl port-forward -n traefik-officer svc/traefik-officer-operator 8084:8084

# Fetch metrics
curl http://localhost:8084/metrics
```

## Migration from Standalone

If you're migrating from the standalone Traefik Officer with a config file:

1. Install the operator via Helm
2. Convert your config file to UrlPerformance resources:

**Old config.json:**
```json
{
  "AllowedServices": [
    {"Name": "grafana-ingress", "Namespace": "monitoring"}
  ],
  "IgnoredPathsRegex": ["^/static/"],
  "TopNPaths": 20
}
```

**New UrlPerformance:**
```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: grafana-monitoring
  namespace: monitoring
spec:
  targetRef:
    kind: Ingress
    name: grafana-ingress
    namespace: monitoring
  ignoredPathsRegex:
    - "^/static/"
  collectNTop: 20
```

## Development

### Build the Operator

```bash
# Build Docker image
docker build -t traefik-officer-operator:latest .

# Or use Go
cd operator
go mod vendor
go build -o traefik-officer-operator .
```

### Run Locally

```bash
# Install CRDs
kubectl apply -f operator/crd/bases/

# Run operator locally
cd operator
export KUBECONFIG=~/.kube/config
go run main.go --leader-elect=false
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

- GitHub Issues: https://github.com/0xvox/traefik-officer/issues
- Documentation: https://github.com/0xvox/traefik-officer
