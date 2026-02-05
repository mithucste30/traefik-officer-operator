# Traefik Officer Operator 🚀

[![CI](https://github.com/mithucste30/traefik-officer-operator/workflows/CI/badge.svg)](https://github.com/mithucste30/traefik-officer-operator/actions/workflows/ci.yml)
[![Release](https://github.com/mithucste30/traefik-officer-operator/workflows/Release/badge.svg)](https://github.com/mithucste30/traefik-officer-operator/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mithucste30/traefik-officer-operator)](https://goreportcard.com/report/github.com/mithucste30/traefik-officer-operator)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> A Kubernetes operator for monitoring Traefik access logs with Prometheus metrics using CRD-based configuration

Traefik Officer Operator extends the standalone Traefik Officer with cloud-native capabilities, allowing you to configure monitoring rules declaratively using Kubernetes Custom Resources.

## ✨ Features

- 🔔 **Custom Resource Definition (CRD)** - `UrlPerformance` for declarative configuration
- 🔄 **Dynamic Configuration** - Update monitoring rules without restarting pods
- 🎯 **Per-Ingress Configuration** - Configure different rules for each Ingress/IngressRoute
- 📊 **Enhanced Metrics** - All metrics include `namespace` and `ingress` labels
- 🐳 **Multi-Architecture Support** - Docker images for AMD64 and ARM64
- 📦 **Helm Chart** - Easy deployment with Helm 3
- 🔍 **Prometheus Integration** - Auto ServiceMonitor creation
- 🎛️ **Path Filtering** - Whitelist and blacklist paths using regex
- 🔧 **URL Normalization** - Reduce metric cardinality with custom patterns
- 📈 **Top N Path Tracking** - Track top paths by latency per ingress
- 🌐 **Multi-Provider** - Works with Ingress and IngressRoute CRDs

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- Helm 3.x
- Traefik v3.0+ running in Kubernetes

### Install the Operator

```bash
# Add Helm repository
helm repo add traefik-officer https://mithucste30.github.io/traefik-officer-operator
helm repo update

# Install the operator
helm install traefik-officer-operator traefik-officer/traefik-officer-operator \
  --namespace traefik-officer \
  --create-namespace
```

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
    name: grafana-ingress
    namespace: monitoring

  # Only monitor API paths
  whitelistPathsRegex:
    - "^/api/"

  # Ignore static assets
  ignoredPathsRegex:
    - "^/static/"
    - "\\.css$"
    - "\\.js$"

  # Merge API paths
  mergePathsWithExtensions:
    - "/api/"

  # Track top 20 paths by latency
  collectNTop: 20

  enabled: true
```

```bash
kubectl apply -f -f urlperformance.yaml
```

### View Metrics

```bash
# Port-forward to access metrics
kubectl port-forward -n traefik-officer svc/traefik-officer-operator 8084:8084

# Fetch metrics
curl http://localhost:8084/metrics
```

## 📊 Metrics

All metrics now include `namespace` and `ingress` labels for better filtering:

```promql
# Request rate per namespace
sum(rate(traefik_officer_requests_total[5m])) by (namespace, ingress)

# Error rate per ingress
sum(rate(traefik_officer_endpoint_requests_total{response_code=~"5.."}[5m])) by (namespace, ingress)

# P95 latency
histogram_quantile(0.95,
  sum(rate(traefik_officer_request_duration_seconds_bucket[5m])) by (namespace, ingress, le)
)
```

## 📚 Documentation

- **[Operator Documentation](OPERATOR.md)** - Comprehensive operator guide
- **[Implementation Summary](IMPLEMENTATION_SUMMARY.md)** - Technical details
- **[Examples](./examples/)** - Example UrlPerformance resources

## 🏗️ Architecture

### Router Name Parsing

The operator intelligently parses Traefik router names:

**Standard Ingress:**
```
websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
↓
Namespace: monitoring
Ingress: grafana-operator-grafana-ingress
```

**IngressRoute CRD:**
```
mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08@kubernetescrd
↓
Namespace: mahfil-dev
IngressRoute: mahfil-api-server-ingressroute-http
```

## 🔧 Configuration

### Helm Values

```yaml
traefik:
  logSource: kubernetes  # or "file"
  kubernetes:
    namespace: ingress-controller
    containerName: traefik
    podLabelSelector: app.kubernetes.io/name=traefik

metrics:
  serviceMonitor:
    enabled: true
    namespace: monitoring
```

See [helm/traefik-officer-operator/values.yaml](./helm/traefik-officer-operator/values.yaml) for all options.

## 🛠️ Development

### Build

```bash
# Build binaries
make build

# Build Docker images
make docker

# Run tests
make test

# Run linters
make lint
```

### Run Locally

```bash
# Install CRDs
make install-crds

# Run operator locally
make run-operator
```

## 📦 Installation

### Docker Images

```bash
# Standalone mode
docker pull ghcr.io/mithucste30/traefik-officer:latest

# Operator mode
docker pull ghcr.io/mithucste30/traefik-officer-operator:latest
```

### Helm Chart

```bash
# From OCI registry (future)
helm install traefik-officer-operator oci://ghcr.io/mithucste30/traefik-officer-operator

# From local chart
helm install traefik-officer-operator ./helm/traefik-officer-operator
```

## 🔄 CI/CD

This project uses GitHub Actions for CI/CD:

- **CI Pipeline** - Runs on every push and PR
  - Linting (golangci-lint)
  - Testing (unit tests with coverage)
  - Docker build tests
  - Helm lint
  - Security scanning (Trivy)

- **Release Pipeline** - Triggered on version tags
  - GoReleaser for binary releases
  - Multi-arch Docker builds (AMD64/ARM64)
  - Helm chart publishing
  - GitHub release creation

## 🤝 Contributing

Contributions are welcome! Please feel free to open an issue or submit a pull request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Traefik](https://traefik.io/) - The cloud-native edge router
- [Kubebuilder](https://book.kubebuilder.io/) - Kubernetes API building
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller framework

## 📧 Support

- **Issues**: [GitHub Issues](https://github.com/mithucste30/traefik-officer-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mithucste30/traefik-officer-operator/discussions)

## 🌟 Star History

If you find this project useful, please consider giving it a ⭐️ on [GitHub](https://github.com/mithucste30/traefik-officer-operator)!

---

**Note**: This project was originally based on the standalone [Traefik Officer](https://github.com/0xvox/traefik-officer) and has been extended with full Kubernetes operator capabilities.
