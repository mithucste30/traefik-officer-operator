# Grafana Dashboard - Quick Start Guide

## TL;DR

Deploy the Traefik Officer Operator with Grafana dashboard to visualize response times of your top N URLs with interactive namespace and service selection.

## One-Line Installation

```bash
helm install traefik-officer-operator ./helm/traefik-officer-operator \
  --set grafanaDashboard.enabled=true \
  --set grafanaDashboard.namespace=monitoring \
  --set grafanaDashboard.datasource=Prometheus
```

## What You Get

A dashboard with **4 panels**:
1. **Top N URLs - Average Response Time** (Bar chart) - Main visualization
2. **Overall Average Response Time** (Gauge) - Single number overview
3. **Top N URLs - Maximum Response Time** (Line chart) - Spikes over time
4. **Top N URLs - Request Rate** (Bar chart) - Traffic volume

## Interactive Controls

- **Namespace Dropdown**: Select which namespace to monitor
- **Service/Ingress Dropdown**: Select which Ingress/IngressRoute to view
- **Top N Selector**: Choose 10, 20, 50, or 100 URLs to display
- **Time Range**: 1 minute to 24 hours

## Metrics Used

The dashboard queries these Prometheus metrics:

```promql
# Average latency per endpoint
traefik_officer_endpoint_avg_latency_seconds{namespace, ingress, request_path}

# Maximum latency per endpoint
traefik_officer_endpoint_max_latency_seconds{namespace, ingress, request_path}

# Request rate per endpoint
rate(traefik_officer_endpoint_requests_total{namespace, ingress, request_path}[5m])
```

## Key Features

✅ **Dynamic Dropdowns**: Namespace and Service dropdowns auto-populate from available metrics
✅ **Top N Filtering**: Shows only the top N URLs by response time (highest to lowest)
✅ **Bar Chart View**: Main panel uses bar chart for easy comparison
✅ **Auto-Refresh**: Dashboard updates every 5 seconds
✅ **Multi-Select**: Select multiple namespaces/services
✅ **Grafana Auto-Discovery**: Dashboard appears automatically when Grafana sidecar is configured

## Configuration

Edit `values.yaml`:

```yaml
grafanaDashboard:
  enabled: true
  namespace: monitoring   # Where Grafana is deployed
  datasource: Prometheus  # Prometheus datasource name in Grafana
```

## Accessing the Dashboard

### Option 1: Auto-Discovery (Recommended)

If Grafana sidecar is configured:
1. Dashboard appears automatically in Grafana
2. Navigate to: **Dashboards** → **Traefik Officer - Top URLs Performance**

### Option 2: Manual Import

1. Copy dashboard JSON:
   ```bash
   kubectl get configmap traefik-officer-operator-dashboard -n monitoring -o jsonpath='{.data.traefik-officer-dashboard\.json}' > dashboard.json
   ```

2. Import in Grafana:
   - Go to **Dashboards** → **Import**
   - Upload `dashboard.json`
   - Select Prometheus datasource

## Example Use Cases

### 1. Find Slowest Endpoints

**Goal**: Identify which URLs have the highest response times

**Steps**:
1. Select your namespace
2. Select your service/ingress
3. Set **Top N URLs** to 20
4. Look at "Top N URLs - Average Response Time" panel
5. URLs are sorted from highest to lowest latency

**Result**: Bar chart shows which endpoints need optimization

### 2. Monitor Multiple Services

**Goal**: Compare performance across all services in a namespace

**Steps**:
1. Select namespace
2. Select **"All"** in Service/Ingress dropdown
3. View aggregated metrics across all services

**Result**: See overall performance trends

### 3. Investigate Spikes

**Goal**: Find when response times spiked

**Steps**:
1. Set **Time Range** to "Last 1 hour" or "Last 6 hours"
2. Look at "Top N URLs - Maximum Response Time" panel
3. Identify time periods with spikes
4. Cross-reference with "Request Rate" panel

**Result**: Correlate traffic spikes with latency issues

### 4. Real-Time Monitoring

**Goal**: Monitor performance in real-time

**Steps**:
1. Set **Time Range** to "Last 1 minute"
2. Dashboard auto-refreshes every 5 seconds
3. Watch for changes in average/max latency

**Result**: Near real-time visibility into performance

## Troubleshooting

### Dashboard Not Showing

```bash
# Check if ConfigMap exists
kubectl get configmap traefik-officer-operator-dashboard -n monitoring

# Verify Grafana sidecar is running
kubectl logs -n monitoring deployment/grafana -c grafana-sc-dashboard

# Restart Grafana to pick up new dashboard
kubectl rollout restart deployment/grafana -n monitoring
```

### No Data in Panels

```bash
# Verify metrics are being scraped
kubectl port-forward -n <namespace> svc/traefik-officer-operator 8084:8084
curl http://localhost:8084/metrics | grep traefik_officer_endpoint

# Check UrlPerformance resources exist
kubectl get urlperformances -A

# Verify Prometheus datasource name matches
# In Grafana: Configuration → Data Sources → Check name
# In values.yaml: grafanaDashboard.datasource should match
```

### Dropdown Variables Empty

- Ensure Prometheus is scraping Traefik Officer metrics
- Check Prometheus query syntax works in Prometheus UI
- Verify metric labels exist: `traefik_officer_endpoint_avg_latency_seconds`

## Customization

### Change Default Top N

Edit `helm/traefik-officer-operator/templates/dashboard.yaml`:

```json
{
  "name": "top_n",
  "query": "10,20,50,100,200"  // Add 200
}
```

### Add More Time Ranges

Edit `helm/traefik-officer-operator/templates/dashboard.yaml`:

```json
{
  "name": "resolution",
  "query": "1m,5m,15m,30m,1h,3h,6h,12h,24h,7d"  // Add 7d
}
```

### Adjust Refresh Rate

Edit dashboard JSON:

```json
{
  "refresh": "10s"  // Change from 5s to 10s
}
```

## Dashboard Panels Details

### Panel 1: Top N URLs - Average Response Time

**Query**:
```promql
topk($top_n, sort_by(avg(traefik_officer_endpoint_avg_latency_seconds{namespace="$namespace", ingress="$ingress"}) by (request_path), -avg_value))
```

**What it shows**:
- Average response time for top N URLs
- Sorted from highest to lowest
- Bar chart for easy comparison
- Updates in real-time

### Panel 2: Overall Average Response Time

**Query**:
```promql
avg(traefik_officer_endpoint_avg_latency_seconds{namespace="$namespace", ingress="$ingress"})
```

**What it shows**:
- Single number: average latency across all endpoints
- Gauge visualization
- Quick health check

### Panel 3: Top N URLs - Maximum Response Time

**Query**:
```promql
topk($top_n, max(traefik_officer_endpoint_max_latency_seconds{namespace="$namespace", ingress="$ingress"}) by (request_path))
```

**What it shows**:
- Maximum response time over time period
- Line chart showing trends
- Identifies spikes and worst-case scenarios

### Panel 4: Top N URLs - Request Rate

**Query**:
```promql
topk($top_n, sum(rate(traefik_officer_endpoint_requests_total{namespace="$namespace", ingress="$ingress"}[5m])) by (request_path))
```

**What it shows**:
- Requests per second for top N URLs
- Bar chart showing traffic volume
- Correlates load with latency

## Advanced: Create Alerts

### High Average Latency Alert

```yaml
# In Grafana: Alerting → New Alert Rule
name: High Average Latency
query: avg(traefik_officer_endpoint_avg_latency_seconds{namespace="$namespace", ingress="$ingress"})
condition: > 1.0  # 1 second
```

### Spike in Max Latency Alert

```yaml
name: Latency Spike
query: max(traefik_officer_endpoint_max_latency_seconds{namespace="$namespace", ingress="$ingress"})
condition: > 5.0  # 5 seconds
```

## Best Practices

1. **Start with Top 10**: Use default Top N = 10 for overview
2. **Drill down**: Increase to 50 or 100 for detailed analysis
3. **Use time ranges**: Check historical data (1h, 6h, 24h) to identify patterns
4. **Correlate metrics**: Compare latency panels with request rate panel
5. **Monitor specific services**: Don't use "All" unless needed - query performance is better with specific service selection

## Related Documentation

- **Full Documentation**: See `GRAFANA_DASHBOARD.md`
- **UrlPerformance CRD**: See `OPERATOR.md`
- **Deployment Guide**: See `DEPLOYMENT.md`
- **Helm Chart**: See `helm/traefik-officer-operator/values.yaml`

## Summary

✅ **Easy deployment**: One Helm install with dashboard enabled
✅ **Interactive**: Dynamic dropdowns for namespace and service selection
✅ **Visual**: Bar charts show top N URLs by response time (highest to lowest)
✅ **Real-time**: Auto-refreshes every 5 seconds
✅ **Flexible**: Configure top N, time range, and filters
✅ **Production-ready**: Uses optimized PromQL queries with `topk()` and `sort_by()`

---

**Dashboard ID**: `traefik-officer-top-urls`
**Version**: 1.0
**Requires**: Grafana 8.0+, Prometheus, Traefik Officer Operator
