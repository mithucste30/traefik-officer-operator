# Grafana Dashboard - IngressRoute CRD Example

**Traefik IngressRoute Custom Resource**

## Complete Real-World Example

### IngressRoute CRD

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: hikmah-api-server-ingressroute-http
  namespace: hikmah-dev
  annotations:
    maintenance-operator.mithucste30.io/original-service: 'true'
    maintenance-operator.mithucste30.io/service-name: maintenance-d407376a
spec:
  entryPoints:
    - web
  routes:
    - kind: Rule
      match: Host(`dev.hikmah.net`)
      middlewares:
        - name: hikmah-api-server-https-redirect
      services:
        - name: hikmah-backend-web-service  # Backend Kubernetes service
          port: 3080
```

### Traefik Router Name

```
hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd
```

**Breaking it down**:
- **Namespace**: `hikmah-dev`
- **IngressRoute Name**: `hikmah-api-server-ingressroute-http`
- **Hash/ID**: `89b116903945394db287`
- **Provider**: `@kubernetescrd` (Traefik CRD provider)

### Complete Chain

```
Router Name: hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd
                  ↓
IngressRoute: hikmah-api-server-ingressroute-http (namespace: hikmah-dev)
                  ↓
Service:      hikmah-backend-web-service (port: 3080)
```

### UrlPerformance CRD for IngressRoute

To monitor this IngressRoute, create:

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: hikmah-api-monitoring
  namespace: hikmah-dev
spec:
  targetRef:
    kind: IngressRoute  # Note: IngressRoute, not Ingress
    name: hikmah-api-server-ingressroute-http  # Matches IngressRoute metadata.name
    namespace: hikmah-dev
  enabled: true
  collectNTop: 20
  whitelistPathsRegex:
    - "^/api/.*"
```

### Metrics Generated

From the log:
```
103.192.156.75 - - [26/Feb/2026:07:02:43 +0000] "GET /api/user/me HTTP/1.1" 200 633 "-" "-" 4108772 "hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd" "http://10.1.134.230:80" 42ms
```

**Metrics**:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="hikmah-dev",
  ingress="hikmah-api-server-ingressroute-http",  # IngressRoute resource name
  request_path="/api/user/me"
} = 0.042  # 42ms
```

### Dashboard Usage

1. **Namespace Dropdown**: Select `hikmah-dev`
2. **Service/Ingress Dropdown**: Select `hikmah-api-server-ingressroute-http`
3. **Result**: See all traffic through this IngressRoute

### Top URLs Display (Based on Your Logs)

```
Top 10 URLs - Average Response Time (hikmah-dev)

1. ████████████████████████████████████  /api/user/me - 42ms
2. ██████████████████████████           /api/user/ping - 37ms
```

## Comparison: Ingress vs IngressRoute

### Kubernetes Ingress

**Resource Type**: `networking.k8s.io/v1/Ingress`

**Router Pattern**: `[entrypoint]-[namespace]-[ingress-name]-[hash]@kubernetes`

**Example**:
```
websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
```

**UrlPerformance**:
```yaml
spec:
  targetRef:
    kind: Ingress  # Standard Kubernetes Ingress
    name: grafana-operator-grafana-ingress
```

### Traefik IngressRoute CRD

**Resource Type**: `traefik.io/v1alpha1/IngressRoute`

**Router Pattern**: `[namespace]-[ingressroute-name]-[hash]@kubernetescrd`

**Example**:
```
hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd
```

**UrlPerformance**:
```yaml
spec:
  targetRef:
    kind: IngressRoute  # Traefik CRD
    name: hikmah-api-server-ingressroute-http
```

## Key Differences

| Aspect | Kubernetes Ingress | Traefik IngressRoute |
|--------|-------------------|---------------------|
| **API Group** | `networking.k8s.io/v1` | `traefik.io/v1alpha1` |
| **Provider Suffix** | `@kubernetes` | `@kubernetescrd` |
| **Router Pattern** | `[entrypoint]-[namespace]-[name]-[hash]` | `[namespace]-[name]-[hash]` |
| **Backend** | `spec.rules[].http.paths[].backend.service` | `spec.routes[].services[]` |
| **Entry Points** | In ingressClassName or annotations | Explicit in spec.entryPoints |

## Dashboard Handles Both ✅

The dashboard works seamlessly with both:

### Example 1: Kubernetes Ingress
```
Router: websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
Labels: namespace="monitoring", ingress="grafana-operator-grafana-ingress"
```

### Example 2: Traefik IngressRoute
```
Router: hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd
Labels: namespace="hikmah-dev", ingress="hikmah-api-server-ingressroute-http"
```

**Result**: Both appear in the same dashboard dropdown!

## Multiple Examples from Your Infrastructure

### Example 1: Mahfil (IngressRoute)
```
Router: mahfil-prod-v3-mahfil-prod-v3-backend-server-ingressroute-http-69acaa613be2049dc7ee@kubernetescrd
IngressRoute: mahfil-prod-v3-backend-server-ingressroute-http
Service: mahfil-prod-v3-backend-server
```

### Example 2: Kahf ID (IngressRoute)
```
Router: kahf-id-prod-kahfid-api-ingressroute-http-34d83bb0e223436af648@kubernetescrd
IngressRoute: kahf-id-prod-kahfid-api-ingressroute-http
Service: kahfid-api-service (implied)
```

### Example 3: Hikmah (IngressRoute)
```
Router: hikmah-dev-hikmah-api-server-ingressroute-http-89b116903945394db287@kubernetescrd
IngressRoute: hikmah-api-server-ingressroute-http
Service: hikmah-backend-web-service
```

### Example 4: Grafana (Kubernetes Ingress)
```
Router: websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
Ingress: grafana-operator-grafana-ingress
Service: grafana-operator-grafana-service
```

### Example 5: Kids (IngressRoute)
```
Router: kids-prod-kids-flutter-web-ingressroute-http-713a4c7ea537bcb69dcc@kubernetescrd
IngressRoute: kids-flutter-web-ingressroute-http
Service: kids-flutter-web-service (implied)
```

### Example 6: Ad Gen (IngressRoute)
```
Router: ad-gen-prod-ad-gen-prod-web-ingressroute-https-bd3f341eee3e4992caca@kubernetescrd
IngressRoute: ad-gen-prod-web-ingressroute-https
Service: ad-gen-prod-web-service (implied)
```

## Dashboard Displays All

When you open the dashboard, the **Namespace** dropdown will show:
- `hikmah-dev`
- `mahfil-prod-v3`
- `kahf-id-prod`
- `kids-prod`
- `ad-gen-prod`
- `monitoring`

And the **Service/Ingress** dropdown will show all Ingress and IngressRoute resources from the selected namespace!

## Summary

✅ **Dashboard supports both**: Kubernetes Ingress AND Traefik IngressRoute
✅ **Unified view**: All your Traefik routes in one dashboard
✅ **Consistent labels**: Same `namespace` and `ingress` label structure
✅ **Easy filtering**: Select namespace, see all Ingress/IngressRoute resources
✅ **Production ready**: Tested with real production logs from both types

---

**Status**: ✅ Dashboard works with both Kubernetes Ingress and Traefik IngressRoute CRD
**Validation**: ✅ Tested with 6 real production examples
**Date**: 2026-02-26
