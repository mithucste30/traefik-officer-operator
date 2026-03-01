# Grafana Dashboard for Traefik Officer

**Overview**: This document describes the Grafana dashboard included with the Traefik Officer Operator Helm chart.

## Features

The Traefik Officer Grafana Dashboard provides real-time visibility into your Traefik ingress performance with the following features:

### Interactive Controls

1. **Namespace Dropdown**
   - Select one or multiple namespaces to filter metrics
   - Uses Prometheus label queries to dynamically populate available namespaces
   - Supports "All" option to view metrics across all namespaces

2. **Service/Ingress Dropdown**
   - Select one or multiple Ingress/IngressRoute resources
   - Dynamically filtered based on selected namespace
   - Displays all Ingress/IngressRoute resources being monitored

3. **Top N URLs Selector**
   - Choose how many top URLs to display (10, 20, 50, or 100)
   - URLs are ranked by response time (highest to lowest)
   - Default: 10 URLs

4. **Time Range Selector**
   - Select time range from 1 minute to 24 hours
   - Default: Last 5 minutes
   - Dashboard auto-refreshes every 5 seconds

### Dashboard Panels

#### Panel 1: Top N URLs - Average Response Time (Bar Chart)
- **Metric**: `traefik_officer_endpoint_avg_latency_seconds`
- **Visualization**: Bar chart showing average response time for top N URLs
- **Sorting**: URLs displayed from highest to lowest response time
- **Purpose**: Quickly identify which endpoints have the highest latency

#### Panel 2: Overall Average Response Time (Gauge)
- **Metric**: Average of all endpoint latencies for selected namespace/ingress
- **Visualization**: Single value gauge
- **Purpose**: At-a-glance view of overall system performance

#### Panel 3: Top N URLs - Maximum Response Time (Time Series)
- **Metric**: `traefik_officer_endpoint_max_latency_seconds`
- **Visualization**: Line chart showing maximum response times over time
- **Purpose**: Identify spikes and worst-case scenarios per endpoint

#### Panel 4: Top N URLs - Request Rate (Bar Chart)
- **Metric**: `traefik_officer_endpoint_requests_total` (rate)
- **Visualization**: Bar chart showing requests per second
- **Purpose**: Correlate traffic volume with response times

## Metrics Used

The dashboard utilizes the following Prometheus metrics exposed by Traefik Officer:

| Metric | Labels | Description |
|--------|--------|-------------|
| `traefik_officer_endpoint_avg_latency_seconds` | `namespace`, `ingress`, `request_path` | Average latency per endpoint |
| `traefik_officer_endpoint_max_latency_seconds` | `namespace`, `ingress`, `request_path` | Maximum latency per endpoint |
| `traefik_officer_endpoint_requests_total` | `namespace`, `ingress`, `request_path`, `request_method`, `response_code` | Total requests per endpoint |

### Understanding Labels

- **`namespace`**: Kubernetes namespace where the Ingress/IngressRoute resides
- **`ingress`**: Name of the Ingress or IngressRoute resource (from UrlPerformance CRD `targetRef.name`)
  - **IMPORTANT**: This is the Ingress/IngressRoute resource name, NOT the Kubernetes service name
  - Example: `monitoring-grafana-operator-grafana-ingress`
  - See `GRAFANA_DASHBOARD_CLARIFICATION.md` for detailed explanation
- **`request_path`**: Normalized URL path (with IDs/tokens replaced)
- **`request_method`**: HTTP method (GET, POST, etc.)
- **`response_code`**: HTTP status code (200, 404, 500, etc.)

**Note**: The "Service/Ingress" dropdown in the dashboard filters by Ingress/IngressRoute name, not by Kubernetes service name.

## Installation

### Prerequisites

1. **Prometheus** installed and configured to scrape Traefik Officer metrics
2. **Grafana** installed with Prometheus datasource configured
3. **Traefik Officer Operator** deployed via Helm chart

### Deploying the Dashboard

The dashboard is automatically deployed when you enable it in the Helm chart:

```bash
helm install traefik-officer-operator ./helm/traefik-officer-operator \
  --set grafanaDashboard.enabled=true \
  --set grafanaDashboard.namespace=monitoring \
  --set grafanaDashboard.datasource=Prometheus
```

### Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `grafanaDashboard.enabled` | Enable/disable dashboard creation | `true` |
| `grafanaDashboard.namespace` | Namespace where Grafana is deployed | `monitoring` |
| `grafanaDashboard.datasource` | Prometheus datasource name in Grafana | `Prometheus` |

### Manual Import

If you prefer to manually import the dashboard:

1. Navigate to Grafana → Dashboards → Import
2. Enter dashboard ID: `traefik-officer-top-urls`
3. Or paste the JSON from the ConfigMap:
   ```bash
   kubectl get configmap traefik-officer-operator-dashboard -n monitoring -o jsonpath='{.data.traefik-officer-dashboard\.json}'
   ```

## Dashboard Auto-Discovery

The dashboard uses Grafana's sidecar auto-discovery feature. When properly configured:

1. The ConfigMap is labeled with `grafana_dashboard: "1"`
2. Grafana automatically discovers and imports the dashboard
3. Dashboard is available at: Grafana → Dashboards → "Traefik Officer - Top URLs Performance"

### ConfigMap Labels

```yaml
metadata:
  labels:
    grafana_dashboard: "1"  # Required for auto-discovery
```

## Example Usage

### Monitoring a Specific Service

1. Select the **Namespace** where your service is deployed
2. Select the **Ingress/IngressRoute** name (e.g., "my-app-ingress")
3. Set **Top N URLs** to 20
4. View the bar chart to see which endpoints have the highest response times

### Comparing Multiple Services

1. Select **"All"** in the **Service/Ingress** dropdown
2. The dashboard will aggregate metrics across all services in the selected namespace
3. Compare response times across different services

### Troubleshooting Performance Issues

1. Set **Time Range** to "Last 1 hour" for historical context
2. Look for spikes in the **Maximum Response Time** panel
3. Check if high-traffic URLs (from **Request Rate** panel) correlate with high latency
4. Drill down into specific URLs to identify bottlenecks

## Customization

### Adding Custom Queries

To add custom panels, edit the dashboard JSON in `helm/traefik-officer-operator/templates/dashboard.yaml`:

```json
{
  "targets": [
    {
      "expr": "your_custom_prometheus_query{namespace=\"$namespace\", ingress=\"$ingress\"}",
      "legendFormat": "{{ request_path }}",
      "refId": "A"
    }
  ]
}
```

### Adjusting Time Ranges

Modify the `resolution` variable in the dashboard template:

```yaml
"query": "1m,5m,15m,30m,1h,3h,6h,12h,24h,7d"
```

### Changing Top N Options

Modify the `top_n` variable in the dashboard template:

```yaml
"query": "5,10,20,50,100,200"
```

## Troubleshooting

### Dashboard Not Appearing in Grafana

1. **Verify ConfigMap exists**:
   ```bash
   kubectl get configmap -l grafana_dashboard=1
   ```

2. **Check Grafana sidecar logs**:
   ```bash
   kubectl logs -n monitoring deployment/grafana -c grafana-sc-dashboard
   ```

3. **Verify namespace matches Grafana deployment**:
   ```bash
   kubectl get configmap traefik-officer-operator-dashboard -o yaml
   ```

4. **Ensure datasource name matches**:
   - Check Grafana datasources: Configuration → Data Sources
   - Update `grafanaDashboard.datasource` if needed

### No Data Showing in Panels

1. **Verify Prometheus is scraping metrics**:
   ```bash
   # Check if metrics endpoint is accessible
   kubectl port-forward -n <namespace> svc/traefik-officer-operator 8084:8084
   curl http://localhost:8084/metrics

   # Query Prometheus directly
   kubectl exec -n monitoring prometheus-0 -- promtool query instant http://prometheus:9090/api/v1/query?query=traefik_officer_endpoint_avg_latency_seconds
   ```

2. **Check UrlPerformance resources exist**:
   ```bash
   kubectl get urlperformances -A
   ```

3. **Verify labels match**:
   ```bash
   # Check available label values in Prometheus
   curl 'http://prometheus:9090/api/v1/label/__name__/values' | grep traefik_officer
   ```

### Dropdown Variables Empty

1. **Prometheus query syntax**: Ensure Prometheus version supports label_values queries
2. **Time range**: Variables may need time range adjustment in Grafana
3. **Metric availability**: Verify metrics are being scraped before dashboard queries

## Performance Considerations

### Dashboard Refresh Rate

- Default: 5 seconds
- For large deployments, consider increasing to 10-30 seconds
- Modify in dashboard JSON: `"refresh": "10s"`

### Query Optimization

The dashboard uses `topk()` to limit results:

```promql
topk($top_n, sort_by(avg(...) by (request_path), -avg_value))
```

This ensures:
- Only top N results are computed
- Query performance remains stable
- Memory usage is controlled

### Label Cardinality

The dashboard filters by `namespace` and `ingress` labels to:
- Reduce query complexity
- Improve dashboard responsiveness
- Prevent high cardinality issues

## Advanced Features

### Annotations

The dashboard includes a default "Annotations & Alerts" panel for:
- Recording deployment events
- Marking incident periods
- Adding context to metrics

### Exporting Dashboard Data

1. Click panel menu (three dots)
2. Select "Export" → "CSV" or "JSON"
3. Useful for post-analysis and reporting

### Sharing Snapshots

1. Click **Share** icon in top toolbar
2. Select **Snapshot** to create temporary link
3. Or export as JSON for permanent storage

## Integration with Alerting

The dashboard is designed to work with Grafana Alerting:

### Example Alert Rules

1. **High Average Latency**:
   - Query: `avg(traefik_officer_endpoint_avg_latency_seconds) > 1`
   - Condition: Average latency exceeds 1 second

2. **Spike in Maximum Latency**:
   - Query: `max(traefik_officer_endpoint_max_latency_seconds) > 5`
   - Condition: Any endpoint exceeds 5 seconds

3. **High Error Rate**:
   - Query: `rate(traefik_officer_endpoint_requests_total{response_code=~"5.."}[5m]) / rate(traefik_officer_endpoint_requests_total[5m]) > 0.05`
   - Condition: Error rate exceeds 5%

## Related Resources

- **UrlPerformance CRD Documentation**: See `OPERATOR.md`
- **Metrics Reference**: See `pkg/metrics.go`
- **Deployment Guide**: See `DEPLOYMENT.md`
- **Helm Chart Configuration**: See `helm/traefik-officer-operator/values.yaml`

## Support

For issues or questions:
1. Check logs: `kubectl logs -n <namespace> deployment/traefik-officer-operator`
2. Verify configuration: `kubectl get urlperformances -A -o yaml`
3. Test queries in Prometheus UI before debugging dashboard
4. Review Grafana dashboard JSON syntax using [Grafana Dashboard Validator](https://grafana.com/docs/grafana/latest/reference/dashboard/)

---

**Dashboard UID**: `traefik-officer-top-urls`
**Dashboard Version**: 1.0
**Compatible Grafana Versions**: 8.0+
**Required Metrics**: Traefik Officer Operator v1.0+
