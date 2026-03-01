# Grafana Dashboard - Label Clarification

## Important: Understanding Labels in Metrics

The Traefik Officer metrics use the following labels:

### `namespace`
- **Kubernetes namespace** where the Ingress/IngressRoute resides
- Example: `monitoring`, `production`, `staging`

### `ingress`
- **Name of the Ingress or IngressRoute resource** (from UrlPerformance CRD `targetRef.name`)
- This is **NOT** the Kubernetes service name
- Example: `monitoring-grafana-operator-grafana-ingress`, `mahfil-dev-mahfil-api-server-ingressroute`

### `request_path`
- **Normalized URL path** (with IDs/tokens replaced)
- Example: `/api/users/{id}`, `/public/plugins/grafana-lokiexplore-app/{id}.js`

## Traefik Router Name Pattern

Traefik generates router names in the format:
```
[entryPoint]-[ingress-name]-[hash]@[provider]
```

**Example from real logs**:
```
websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
```

Breaking this down:
- **EntryPoint**: `websecure`
- **Ingress Name**: `monitoring-grafana-operator-grafana-ingress`
- **Hash/ID**: `grafana-non-production-kahf-co`
- **Provider**: `kubernetes`

The `ingress` label in our metrics corresponds to **`monitoring-grafana-operator-grafana-ingress`**.

## Ingress vs Service

An **Ingress** resource can point to one or more **Services**:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monitoring-grafana-operator-grafana-ingress  # This is the "ingress" label value
  namespace: monitoring
spec:
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: grafana-operator-grafana  # This is the Kubernetes service name
            port:
              number: 3000
```

In this example:
- **`ingress` label value**: `monitoring-grafana-operator-grafana-ingress`
- **Kubernetes service name**: `grafana-operator-grafana`

## Why Ingress Name Instead of Service Name?

Using the Ingress/IngressRoute name as the label provides:
1. **Direct mapping to UrlPerformance CRD**: Each CRD instance monitors one Ingress/IngressRoute
2. **Consistent with CRD structure**: The CRD references Ingress by name, not service
3. **Multi-service support**: An Ingress can route to multiple services; using Ingress name captures all traffic
4. **Simpler configuration**: Users create UrlPerformance for an Ingress, not individual services

## Current Dashboard Behavior

The dashboard's "Service/Ingress" dropdown actually filters by **Ingress/IngressRoute name**, not Kubernetes service name.

**Example**:
- If you have a UrlPerformance CRD monitoring `monitoring-grafana-operator-grafana-ingress`
- Select this in the dashboard's "Service/Ingress" dropdown
- You'll see all traffic through that Ingress, regardless of which backend services it routes to

## Future Enhancement: Add Service Label

We're working on adding a `service` label that will contain the actual Kubernetes service name. This will allow:
- Filtering by specific backend services
- Seeing metrics per service (not just per Ingress)
- Better correlation with Kubernetes service-level monitoring

**Status**: Infrastructure is in place (ServiceNames field added to RuntimeConfig), but needs:
1. Service name extraction from Traefik router names
2. Metrics label addition (breaking change)
3. Dashboard update to include service dropdown
4. Testing with various router name patterns

## Example: Real Traefik Logs

### Complete Real-World Example

**Traefik Log Entry**:
```
114.130.157.18 - - [26/Feb/2026:07:00:10 +0000] "GET /api/user/preferences HTTP/2.0" 200 2 "-" "-" 5614962 "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes" "http://10.1.8.116:3000" 1ms
```

**Corresponding Ingress Resource**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana-operator-grafana-ingress
  namespace: monitoring
spec:
  ingressClassName: traefik
  rules:
    - host: grafana.non.production.kahf.co
      http:
        paths:
          - path: /
            pathType: ImplementationSpecific
            backend:
              service:
                name: grafana-operator-grafana-service  # Backend service
                port:
                  name: grafana
```

**Complete Chain**:
```
Router Name: websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
                  ↓
Ingress:     grafana-operator-grafana-ingress (namespace: monitoring)
                  ↓
Service:     grafana-operator-grafana-service
```

**Metrics Generated**:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="grafana-operator-grafana-ingress",  # Ingress resource name
  request_path="/api/user/preferences"
} = 0.001
```

**Key Points**:
- ✅ `namespace` = `monitoring` (from Ingress namespace)
- ✅ `ingress` = `grafana-operator-grafana-ingress` (from Ingress metadata.name)
- ❌ Service name `grafana-operator-grafana-service` is **NOT** in metrics (future enhancement)

**Dashboard Usage**:
1. Select **Namespace**: `monitoring`
2. Select **Service/Ingress**: `grafana-operator-grafana-ingress`
3. View all traffic through this Ingress in the bar chart

## Querying Examples

### View metrics for a specific Ingress:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="monitoring-grafana-operator-grafana-ingress"
}
```

### View metrics across all Ingresses in a namespace:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring"
}
```

### Compare two Ingresses:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  ingress=~"monitoring-.*-ingress"
}
```

## Dashboard Dropdown Mapping

| Dashboard Field | Actual Label | Example Value |
|----------------|--------------|---------------|
| **Namespace** | `namespace` | `monitoring` |
| **Service/Ingress** | `ingress` | `monitoring-grafana-operator-grafana-ingress` |
| *(Future)* Service | *(not yet implemented)* | `grafana-operator-grafana` |

## Summary

✅ **Current**: Dashboard filters by Ingress/IngressRoute name
✅ **Benefit**: Aligns with UrlPerformance CRD structure
✅ **Use Case**: Monitor all traffic through a specific Ingress
⏳ **Future**: Will add Kubernetes service name as additional label

---

**For questions or issues**, please refer to:
- `GRAFANA_DASHBOARD.md` - Full dashboard documentation
- `OPERATOR.md` - UrlPerformance CRD documentation
- `pkg/metrics.go` - Metrics definitions
