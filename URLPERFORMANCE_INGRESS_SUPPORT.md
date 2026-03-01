# UrlPerformance - Native Kubernetes Ingress Support

**Status**: ✅ **FULLY SUPPORTED**

## Overview

The UrlPerformance CRD **fully supports** native Kubernetes Ingress resources. You can monitor both:
- ✅ **Kubernetes Ingress** (`networking.k8s.io/v1/Ingress`)
- ✅ **Traefik IngressRoute** (`traefik.io/v1alpha1/IngressRoute`) - via log processing

## Native Kubernetes Ingress Example

### Step 1: Create a Kubernetes Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-api-ingress
  namespace: production
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - api.example.com
      secretName: api-tls
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: my-api-service
                port:
                  number: 8080
          - path: /graphql
            pathType: Exact
            backend:
              service:
                name: my-graphql-service
                port:
                  number: 4000
```

### Step 2: Create UrlPerformance CRD

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: my-api-monitoring
  namespace: production
spec:
  targetRef:
    kind: Ingress  # Native Kubernetes Ingress
    name: my-api-ingress  # Matches Ingress metadata.name
    namespace: production  # Matches Ingress namespace
  enabled: true
  collectNTop: 20
  whitelistPathsRegex:
    - "^/api/.*"  # Monitor all /api/* paths
    - "^/graphql$"  # Monitor GraphQL endpoint
  mergePathsWithExtensions:
    - "/api/v1/"  # Merge all /api/v1/* paths
  ignoredPathsRegex:
    - "^/health$"  # Don't monitor health check
    - "^/metrics$"  # Don't monitor metrics endpoint
```

### Step 3: Apply the Resources

```bash
kubectl apply -f my-api-ingress.yaml
kubectl apply -f my-api-ingress-urlperformance.yaml
```

### Step 4: Verify

```bash
# Check UrlPerformance status
kubectl get urlperformance my-api-monitoring -n production -o yaml

# Status should show:
# status:
#   phase: Active
#   conditions:
#   - type: TargetExists
#     status: "True"
#   - type: ConfigGenerated
#     status: "True"
#   - type: Ready
#     status: "True"
```

## How It Works

### Controller Processing

The controller performs these steps:

1. **Reads UrlPerformance CRD**
   ```yaml
   spec:
     targetRef:
       kind: Ingress
       name: my-api-ingress
       namespace: production
   ```

2. **Fetches the Ingress**
   ```go
   ingress := &networkingv1.Ingress{}
   r.Get(ctx, types.NamespacedName{
       Namespace: "production",
       Name: "my-api-ingress",
   }, ingress)
   ```

3. **Extracts Service Names**
   ```go
   serviceNames = extractServiceNamesFromIngress(ingress)
   // Returns: ["my-api-service", "my-graphql-service"]
   ```

4. **Creates RuntimeConfig**
   ```go
   runtimeConfig := &shared.RuntimeConfig{
       Key:          "production-my-api-ingress",
       Namespace:    "production",
       TargetName:   "my-api-ingress",
       TargetKind:   "Ingress",
       ServiceNames: ["my-api-service", "my-graphql-service"],
       // ... other config
   }
   ```

5. **Updates ConfigManager** (used by log processor)

### Log Processing

When Traefik logs arrive:

```
GET /api/users HTTP/1.1" 200 1234 "-" "-" 5614 "websecure-production-my-api-ingress-abc123@kubernetes" "http://10.0.0.1:8080" 5ms
```

The log processor:
1. Parses router name: `websecure-production-my-api-ingress-abc123@kubernetes`
2. Extracts namespace: `production`
3. Extracts ingress: `my-api-ingress`
4. Normalizes path: `/api/users`
5. Records metrics with labels

### Metrics Generated

```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="production",
  ingress="my-api-ingress",
  request_path="/api/users"
} = 0.005
```

## Multi-Service Ingress

One Ingress can route to multiple services:

### Ingress with Multiple Backends

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-service-ingress
  namespace: production
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /api
            backend:
              service:
                name: api-service        # Backend 1
          - path: /graphql
            backend:
              service:
                name: graphql-service    # Backend 2
          - path: /
            backend:
              service:
                name: frontend-service   # Backend 3
```

### UrlPerformance for Multi-Service Ingress

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: multi-service-monitoring
  namespace: production
spec:
  targetRef:
    kind: Ingress
    name: multi-service-ingress
    namespace: production
  enabled: true
  collectNTop: 30
```

**Result**: The dashboard will show metrics for ALL traffic through `multi-service-ingress`, regardless of which backend service handled the request.

**Dashboard Labels**:
```promql
traefik_officer_endpoint_avg_latency_seconds{
  namespace="production",
  ingress="multi-service-ingress",  # The Ingress, not individual services
  request_path="/api/users"
}
```

This is **correct behavior** because:
- You're monitoring the Ingress as an entry point
- All traffic flows through the Ingress
- You can see overall performance of the ingress routing

## Comparison: Ingress vs IngressRoute

### Native Kubernetes Ingress

**Pros**:
- ✅ Standard Kubernetes resource (no custom CRD needed)
- ✅ Works with any ingress controller (Traefik, NGINX, etc.)
- ✅ Native Kubernetes tooling support
- ✅ Standardized YAML structure

**Cons**:
- ❌ Less Traefik-specific features
- ❌ Limited routing options compared to IngressRoute

**UrlPerformance Support**:
```yaml
spec:
  targetRef:
    kind: Ingress  # ✅ FULLY SUPPORTED
    name: my-ingress
```

### Traefik IngressRoute CRD

**Pros**:
- ✅ Traefik-specific features (middlewares, etc.)
- ✅ More routing flexibility
- ✅ Better Traefik integration

**Cons**:
- ❌ Traefik-specific (not portable)
- ❌ Requires Traefik CRDs installed

**UrlPerformance Support**:
```yaml
spec:
  targetRef:
    kind: IngressRoute  # ✅ SUPPORTED via log processing
    name: my-ingressroute
```

## Real-World Example from Your Infrastructure

### Grafana Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana-operator-grafana-ingress
  namespace: monitoring
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - grafana.non-production.kahf.co
      secretName: grafana.local-tls
  rules:
    - host: grafana.non.production.kahf.co
      http:
        paths:
          - path: /
            pathType: ImplementationSpecific
            backend:
              service:
                name: grafana-operator-grafana-service
                port:
                  name: grafana
```

### UrlPerformance for Grafana

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: grafana-monitoring
  namespace: monitoring
spec:
  targetRef:
    kind: Ingress  # Native Kubernetes Ingress
    name: grafana-operator-grafana-ingress
    namespace: monitoring
  enabled: true
  collectNTop: 20
  whitelistPathsRegex:
    - "^/api/.*"
    - "^/public/.*"
    - "^/dashboard/.*"
```

### Metrics from Traefik Logs

```
"GET /api/user/preferences HTTP/2.0" 200 2 "-" "-" 5614962 "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes" "http://10.1.8.116:3000" 1ms
```

### Dashboard Display

**Settings**:
- Namespace: `monitoring`
- Service/Ingress: `grafana-operator-grafana-ingress`

**Bar Chart Shows**:
```
1. ████████████████████████████  /public/plugins/grafana-lokiexplore-app/{id}.js - 5ms
2. ████                         /api/user/preferences - 1ms
3. ████                         /api/login - 1ms
```

## Advanced Examples

### Example 1: Multiple Paths with Different Services

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: production
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /users
            backend:
              service:
                name: users-service
          - path: /orders
            backend:
              service:
                name: orders-service
          - path: /payments
            backend:
              service:
                name: payments-service
```

**UrlPerformance**:
```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: api-gateway-monitoring
  namespace: production
spec:
  targetRef:
    kind: Ingress
    name: api-gateway-ingress
    namespace: production
  enabled: true
  collectNTop: 50
  whitelistPathsRegex:
    - "^/users/.*"
    - "^/orders/.*"
    - "^/payments/.*"
```

**Dashboard Shows**: All three services' paths under one ingress entry point

### Example 2: Path-based Routing with Regex

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
  namespace: production
  annotations:
    traefik.ingress.kubernetes.io/router.middlewares: "stripprefix@file"
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /api/v1
            pathType: Prefix
            backend:
              service:
                name: api-v1-service
          - path: /api/v2
            pathType: Prefix
            backend:
              service:
                name: api-v2-service
```

**UrlPerformance**:
```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: app-monitoring
  namespace: production
spec:
  targetRef:
    kind: Ingress
    name: app-ingress
    namespace: production
  enabled: true
  collectNTop: 20
  mergePathsWithExtensions:
    - "/api/v1/"  # Merge all /api/v1/* paths
    - "/api/v2/"  # Merge all /api/v2/* paths
```

**Dashboard Shows**: Normalized paths like `/api/v1/{id}` instead of individual endpoints

## Benefits of Native Ingress Support

### 1. Standard Kubernetes Resources
- No custom Traefik CRDs required for basic routing
- Works with standard Kubernetes tooling
- Portable across different ingress controllers

### 2. Service Discovery
Controller automatically extracts service names from Ingress:
```go
serviceNames = extractServiceNamesFromIngress(ingress)
// ["api-service", "graphql-service", "frontend-service"]
```

### 3. Multi-Service Monitoring
One UrlPerformance CRD monitors an entire Ingress with multiple backend services

### 4. Path Normalization
Metrics show clean, normalized paths regardless of backend complexity

## Verification Checklist

Use this checklist to verify your Ingress is being monitored:

- [ ] UrlPerformance CRD created with `kind: Ingress`
- [ ] `targetRef.name` matches Ingress metadata.name
- [ ] `targetRef.namespace` matches Ingress namespace
- [ ] UrlPerformance status shows `phase: Active`
- [ ] Condition `TargetExists` is `True`
- [ ] Traefik logs show requests to your Ingress
- [ ] Dashboard shows metrics for your namespace
- [ ] Dashboard shows your Ingress in dropdown
- [ ] Bar chart displays top N URLs

## Troubleshooting

### Issue: UrlPerformance stuck in Pending phase

**Solution**: Check if Ingress exists
```bash
kubectl get ingress -n <namespace>
kubectl get urlperformance -n <namespace> -o yaml
```

### Issue: No metrics appearing in dashboard

**Solution**: Verify Traefik logs contain your router name
```bash
# Check recent logs
kubectl logs -n traefik deployment/traefik | grep "my-ingress"

# Verify router name format
# Should be: [entrypoint]-[namespace]-[ingress-name]-[hash]@kubernetes
```

### Issue: Service names not extracted

**Solution**: Check Ingress has backend services defined
```bash
kubectl get ingress my-ingress -n <namespace> -o jsonpath='{.spec.rules[*].http.paths[*].backend.service.name}'
```

## Summary

✅ **Native Kubernetes Ingress is FULLY SUPPORTED**
✅ **Controller validates Ingress existence**
✅ **Service names automatically extracted**
✅ **Metrics properly labeled**
✅ **Dashboard works seamlessly**
✅ **Production-ready**

You can confidently use UrlPerformance CRD with standard Kubernetes Ingress resources!

---

**Status**: ✅ Production Ready
**Supported**: Kubernetes Ingress v1 (networking.k8s.io/v1/Ingress)
**Date**: 2026-02-26
