# Grafana Dashboards

This directory contains pre-built Grafana dashboards for k8s-cronjob-monitor.

## Available Dashboards

### CronJob Overview (`cronjob-overview.json`)
High-level view of all CronJobs across the cluster:
- Total monitored CronJobs
- Overall success rate
- Recent failures
- Active jobs

### CronJob Details (`cronjob-details.json`)
Per-CronJob deep dive:
- Execution history
- Duration trends
- Success/failure counts
- Schedule delay tracking

## Import Instructions

### Method 1: Grafana UI
1. Open Grafana
2. Click **+** > **Import**
3. Upload the JSON file or paste its contents
4. Select your Prometheus data source
5. Click **Import**

### Method 2: Grafana Provisioning
Add to your Grafana provisioning config:

```yaml
apiVersion: 1
providers:
  - name: 'k8s-cronjob-monitor'
    folder: 'Kubernetes'
    type: file
    options:
      path: /var/lib/grafana/dashboards/k8s-cronjob-monitor
```

Then mount the dashboard JSON files to `/var/lib/grafana/dashboards/k8s-cronjob-monitor/`.

### Method 3: ConfigMap
```bash
kubectl create configmap grafana-dashboard-cronjob-overview \
  --from-file=cronjob-overview.json \
  -n monitoring

kubectl create configmap grafana-dashboard-cronjob-details \
  --from-file=cronjob-details.json \
  -n monitoring
```

Label the ConfigMaps for auto-discovery if using the Grafana sidecar:
```bash
kubectl label configmap grafana-dashboard-cronjob-overview grafana_dashboard=1 -n monitoring
kubectl label configmap grafana-dashboard-cronjob-details grafana_dashboard=1 -n monitoring
```
