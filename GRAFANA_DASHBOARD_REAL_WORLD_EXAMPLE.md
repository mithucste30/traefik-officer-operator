# Grafana Dashboard - Real World Example

**Complete Example from Production Traffic**

## Scenario

You have a Grafana deployment with the following Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana-operator-grafana-ingress
  namespace: monitoring
  labels:
    app.kubernetes.io/instance: grafana-operator
    app.kubernetes.io/name: grafana-operator
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

## Step 1: Create UrlPerformance CRD

```yaml
apiVersion: traefikofficer.io/v1alpha1
kind: UrlPerformance
metadata:
  name: grafana-monitoring
  namespace: monitoring
spec:
  targetRef:
    kind: Ingress
    name: grafana-operator-grafana-ingress  # Matches Ingress metadata.name
    namespace: monitoring
  enabled: true
  collectNTop: 20  # Track top 20 URLs
  whitelistPathsRegex:
    - "^/api/.*"       # Monitor API endpoints
    - "^/public/.*"    # Monitor static assets
    - "^/dashboard/.*" # Monitor dashboards
```

**Apply it**:
```bash
kubectl apply -f grafana-urlperformance.yaml
```

## Step 2: Generate Traffic

Users access Grafana:
```
GET /api/user/preferences
GET /public/plugins/grafana-lokiexplore-app/747.js
GET /api/dashboards/home
POST /api/login
GET /dashboard/home
```

## Step 3: Traefik Generates Logs

```
114.130.157.18 - - [26/Feb/2026:07:00:10 +0000] "GET /api/user/preferences HTTP/2.0" 200 2 "-" "-" 5614962 "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes" "http://10.1.8.116:3000" 1ms
114.130.157.18 - - [26/Feb/2026:07:00:10 +0000] "GET /public/plugins/grafana-lokiexplore-app/747.js HTTP/2.0" 304 0 "-" "-" 5614969 "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes" "http://10.1.8.116:3000" 5ms
114.130.157.18 - - [26/Feb/2026:07:00:10 +0000] "POST /api/login HTTP/2.0" 200 718 "-" "-" 5614963 "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes" "http://10.1.8.116:3000" 1ms
```

## Step 4: Traefik Officer Processes Logs

Extracts metrics from router name:
- **Router**: `websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes`
- **Namespace**: `monitoring`
- **Ingress**: `grafana-operator-grafana-ingress`
- **Request Path**: `/api/user/preferences`
- **Duration**: 1ms

## Step 5: Prometheus Scrapes Metrics

```promql
# Metric 1
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="grafana-operator-grafana-ingress",
  request_path="/api/user/preferences"
} 0.001

# Metric 2
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="grafana-operator-grafana-ingress",
  request_path="/public/plugins/grafana-lokiexplore-app/{id}.js"
} 0.005

# Metric 3
traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="grafana-operator-grafana-ingress",
  request_path="/api/login"
} 0.001
```

## Step 6: Grafana Dashboard Displays Data

### Dashboard Configuration

```yaml
grafanaDashboard:
  enabled: true
  namespace: monitoring
  datasource: Prometheus
```

### Access the Dashboard

1. Open Grafana: `https://grafana.example.com`
2. Navigate to: **Dashboards** → **"Traefik Officer - Top URLs Performance"**

### Use the Dashboard

**Panel 1: Top N URLs - Average Response Time**

**Settings**:
- **Namespace**: `monitoring` ✅
- **Service/Ingress**: `grafana-operator-grafana-ingress` ✅
- **Top N**: `10`
- **Time Range**: `Last 5 minutes`

**Bar Chart Display** (sorted highest to lowest):
```
1. ████████████████████████████  /public/plugins/grafana-lokiexplore-app/{id}.js - 5ms
2. ████                         /api/user/preferences - 1ms
3. ████                         /api/login - 1ms
4. ██                           /api/dashboards/home - 0.8ms
5. ██                           /dashboard/home - 0.6ms
6. █                            /api/search - 0.5ms
7. █                            /api/user/orgs - 0.4ms
8. █                            /public/build/{id}.js - 0.3ms
9. █                            /api/plugins - 0.2ms
10.                            /health - 0.1ms
```

**Panel 2: Overall Average Response Time**

```
┌─────────────────────┐
│   AVG: 1.2ms       │
├─────────────────────┤
│     ████████       │
│     0.0    2.5ms   │
└─────────────────────┘
```

**Panel 3: Top N URLs - Maximum Response Time**

Line chart showing spikes over time for top 10 URLs.

**Panel 4: Top N URLs - Request Rate**

```
1. ████████████████████████████  /api/search - 45 req/s
2. ████████████████             /api/user/preferences - 28 req/s
3. ██████████                   /public/plugins/.../{id}.js - 15 req/s
4. ██████                       /api/dashboards/home - 10 req/s
5. ████                         /api/login - 5 req/s
```

## Step 7: Analyze Performance

### Use Case 1: Identify Slowest Endpoints

**Observation**: `/public/plugins/grafana-lokiexplore-app/{id}.js` has highest avg latency (5ms)

**Action**:
1. Check if this is expected (large JavaScript file)
2. Consider caching or CDN for static assets
3. Verify Grafana plugin loading performance

### Use Case 2: Correlate Traffic with Latency

**Observation**: `/api/search` has highest request rate (45 req/s) but low latency (0.5ms)

**Action**:
1. System is performing well under load
2. Monitor if latency increases with traffic
3. Set up alert if latency exceeds threshold

### Use Case 3: Investigate Spikes

**Observation**: `/api/user/preferences` shows occasional spikes to 50ms in Panel 3

**Action**:
1. Check time range when spikes occur
2. Cross-reference with database performance
3. Investigate if there's a blocking query

## Step 8: Create Alerts (Optional)

### Alert 1: High Average Latency

```yaml
# Grafana Alert
name: Grafana High Latency
query: avg(traefik_officer_endpoint_avg_latency_seconds{namespace="monitoring", ingress="grafana-operator-grafana-ingress"}) > 0.100  # 100ms
condition: avg > 0.1
```

### Alert 2: Latency Spike Detection

```yaml
name: Grafana Latency Spike
query: max(traefik_officer_endpoint_max_latency_seconds{namespace="monitoring", ingress="grafana-operator-grafana-ingress"}) > 0.500  # 500ms
condition: max > 0.5
```

## Complete Flow Diagram

```
User Request
    ↓
Ingress: grafana-operator-grafana-ingress
    ↓
Traefik Router: websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes
    ↓
Traefik Log: "GET /api/user/preferences ... 1ms"
    ↓
Traefik Officer: Parses log, extracts labels
    ↓
Prometheus Metric: traefik_officer_endpoint_avg_latency_seconds{namespace="monitoring", ingress="grafana-operator-grafana-ingress", request_path="/api/user/preferences"} = 0.001
    ↓
Grafana Dashboard: Bar chart shows /api/user/preferences at 1ms
    ↓
You: See performance, identify issues, optimize
```

## Dashboard Query Examples

### View Grafana's Top 10 Slowest URLs
```promql
topk(10, sort_by(avg(traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring",
  ingress="grafana-operator-grafana-ingress"
}) by (request_path), -avg_value))
```

### Compare Grafana with Other Services
```promql
avg(traefik_officer_endpoint_avg_latency_seconds{
  namespace="monitoring"
}) by (ingress)
```

### Find Slow GraphQL Endpoints
```promql
traefik_officer_endpoint_avg_latency_seconds{
  request_path="/graphql"
} > 0.050  # Greater than 50ms
```

## Real Production Data

From your actual logs, here are the top endpoints you'd see:

**For `mahfil-prod-v3-mahfil-prod-v3-backend-server-ingressroute-http`**:
```
1. ████████████████████████████████████  /api/user/me - 42ms
2. ██████████████████████████           /api/user/ping - 37ms
3. ██████████                          /graphql - 11ms
4. ██████                               /api/v1/streaming-partner/homefeed - 8ms
5. ████                                 /api/getRelatedVideos - 7ms
```

**For `kahf-id-prod-kahfid-api-ingressroute-http`**:
```
1. ████████████████████████  /graphql - 12ms (avg)
2. ██████████████████       /api/track - 0ms
```

## Summary

✅ **One CRD** monitors one Ingress
✅ **Dashboard** shows all traffic through that Ingress
✅ **Bar chart** displays top N URLs sorted by response time
✅ **Real-time** monitoring with 5-second refresh
✅ **Easy deployment** via Helm chart

---

**Ready to use!** Deploy the Helm chart with `grafanaDashboard.enabled=true` and start monitoring your Ingress performance.
