# Deployment Guide for Traefik Officer Operator

This guide covers deploying the Traefik Officer Operator with embedded log processing to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (v1.19+)
- Helm 3.x installed
- kubectl configured to access your cluster
- Traefik installed and running

## Quick Start

### 1. Install the Operator

```bash
# Add the Helm repository (if hosted)
helm repo add traefik-officer https://mithucste30.github.io/traefik-officer-operator
helm repo update

# Install the operator
helm install traefik-officer-operator ./helm/traefik-officer-operator
```

### 2. Create a UrlPerformance CRD

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: my-app-monitoring
  namespace: default
spec:
  # Reference to the Ingress to monitor
  targetRef:
    name: my-app-ingress
    kind: Ingress
    namespace: default

  # Enable monitoring
  enabled: true

  # Whitelist specific paths (optional - regex patterns)
  whitelistPathsRegex:
    - "/api/.*"
    - "/health"

  # Ignore specific paths (optional - regex patterns)
  ignoredPathsRegex:
    - "/metrics"
    - "/favicon.ico"

  # Merge paths with extensions (optional)
  mergePathsWithExtensions:
    - "/api/users/"

  # URL patterns to transform (optional)
  urlPatterns:
    - pattern: "/api/users/\\d+"
      replacement: "/api/users/{id}"

  # Collect top N paths
  collectNTop: 10
```

```bash
kubectl apply -f urlperformance-example.yaml
```

### 3. Verify Metrics

```bash
# Port-forward to access metrics
kubectl port-forward -n default svc/traefik-officer-operator 8084:8084

# Check metrics
curl http://localhost:8084/metrics
```

## Configuration Options

### Helm Values

Key configuration options in `values.yaml`:

```yaml
# Operator configuration
operator:
  enabled: true
  leaderElection:
    enabled: true

# Traefik log source
traefik:
  logSource: kubernetes  # or "file"
  kubernetes:
    namespace: ingress-controller
    containerName: traefik
    podLabelSelector: app.kubernetes.io/name=traefik

# Log format
logFormat:
  format: json  # or "common"

# Metrics
metrics:
  port: 8084
  healthPort: 8085
  serviceMonitor:
    enabled: true
    namespace: monitoring
```

### Log Source Modes

#### Kubernetes Mode (Default)
The operator streams logs directly from Traefik pods:

```yaml
traefik:
  logSource: kubernetes
  kubernetes:
    namespace: ingress-controller
    containerName: traefik
    podLabelSelector: app.kubernetes.io/name=traefik
```

**RBAC Requirements:**
- `pods` and `pods/log` permissions (already included in ClusterRole)

#### File Mode
The operator reads logs from a file (requires volume mount):

```yaml
traefik:
  logSource: file
  file:
    path: /var/log/traefik/access.log
```

**Volume Mount Example:**
```yaml
volumes:
  - name: traefik-logs
    hostPath:
      path: /var/log/traefik
volumeMounts:
  - name: traefik-logs
    mountPath: /var/log/traefik
    readOnly: true
```

## Metrics

The operator exposes Prometheus metrics with the following labels:

- `namespace`: Kubernetes namespace from the Ingress/IngressRoute
- `ingress`: Ingress/IngressRoute name
- `target_kind`: "Ingress" or "IngressRoute"
- `request_path`: Normalized URL path
- `request_method`: HTTP method
- `response_code`: HTTP status code

### Example Metrics

```
# Request count per endpoint
traefik_officer_endpoint_requests_total{namespace="default",ingress="my-app",request_path="/api/users",request_method="GET",response_code="200"} 1234

# Request latency histogram
traefik_officer_endpoint_request_duration_seconds{namespace="default",ingress="my-app",request_path="/api/users",request_method="GET",response_code="200",quantile="0.99"} 0.245

# Average latency
traefik_officer_endpoint_avg_latency_seconds{namespace="default",ingress="my-app",request_path="/api/users"} 0.123

# Error rate
traefik_officer_endpoint_error_rate{namespace="default",ingress="my-app",request_path="/api/users"} 0.01
```

## ServiceMonitor

The operator includes a ServiceMonitor for Prometheus Operator:

```yaml
metrics:
  serviceMonitor:
    enabled: true
    namespace: monitoring
    interval: 30s
    labels:
      release: prometheus
```

## Troubleshooting

### Check Operator Status

```bash
kubectl get pods -n default
kubectl logs -n default deployment/traefik-officer-operator
```

### Check UrlPerformance CRD

```bash
kubectl get urlperformances -A
kubectl describe urlperformance my-app-monitoring -n default
```

### Common Issues

**1. No metrics appearing**
- Verify the UrlPerformance CRD has `enabled: true`
- Check that the target Ingress/IngressRoute exists
- Verify Traefik is generating access logs

**2. Permission errors**
- Ensure RBAC is created: `kubectl get clusterrole traefik-officer-operator`
- Check service account is bound correctly

**3. Logs not being processed**
- Check log source configuration matches your Traefik deployment
- Verify pod label selector matches Traefik pods
- Check operator logs for errors

## Advanced Configuration

### Multiple Namespaces

The operator uses ClusterRole/ClusterRoleBinding and can monitor Ingresses across all namespaces:

```bash
kubectl get urlperformances --all-namespaces
```

### Custom Resource Limits

```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi
```

### High Availability

```yaml
replicaCount: 3
operator:
  leaderElection:
    enabled: true
```

## Uninstallation

```bash
# Delete all UrlPerformance CRDs
kubectl delete urlperformance --all --all-namespaces

# Uninstall Helm chart
helm uninstall traefik-officer-operator

# Delete CRDs
kubectl delete crd urlperformances.traefikofficer.io
```

## Next Steps

- Create Grafana dashboards using the metrics
- Set up alerts based on error rates and latency
- Configure log retention and rotation policies
- Tune `collectNTop` for your specific use case
