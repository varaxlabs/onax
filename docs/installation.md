# Installation Guide

## Prerequisites

- Kubernetes cluster (1.19+)
- Prometheus installed and scraping
- Grafana installed (optional, for dashboards)
- Helm 3.0+ (for Helm installation)

## Installation Options

### Option 1: Helm (Recommended)

```bash
# Add Helm repository
helm repo add k8s-cronjob-monitor https://kubeshield.github.io/k8s-cronjob-monitor
helm repo update

# Install with default values
helm install k8s-cronjob-monitor k8s-cronjob-monitor/k8s-cronjob-monitor \
  --namespace monitoring \
  --create-namespace

# Or install with custom values
helm install k8s-cronjob-monitor k8s-cronjob-monitor/k8s-cronjob-monitor \
  --namespace monitoring \
  --create-namespace \
  --set metrics.serviceMonitor.enabled=true \
  --set logging.level=debug
```

### Option 2: Raw Manifests

```bash
kubectl apply -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/deploy/manifests/operator.yaml
```

### Option 3: Build from Source

```bash
# Clone repository
git clone https://github.com/kubeshield/k8s-cronjob-monitor.git
cd k8s-cronjob-monitor

# Deploy using Helm from local chart
helm install k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
  --namespace monitoring \
  --create-namespace
```

## Configure Prometheus Scraping

### If using Prometheus Operator

The Helm chart creates a ServiceMonitor by default when `metrics.serviceMonitor.enabled=true`.

### If using vanilla Prometheus

Add to your `prometheus.yaml`:

```yaml
scrape_configs:
  - job_name: 'k8s-cronjob-monitor'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - monitoring
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        action: keep
        regex: k8s-cronjob-monitor
      - source_labels: [__meta_kubernetes_pod_ip]
        action: replace
        target_label: __address__
        replacement: $1:8080
```

## Import Grafana Dashboards

### Method 1: Import by URL

1. Open Grafana
2. Click "+" → "Import"
3. Paste the raw GitHub URL:
   - Overview: `https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/dashboards/cronjob-overview.json`
   - Details: `https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/dashboards/cronjob-details.json`
4. Select your Prometheus data source
5. Click "Import"

### Method 2: Import from JSON

```bash
# Download dashboards
curl -o cronjob-overview.json \
  https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/dashboards/cronjob-overview.json

curl -o cronjob-details.json \
  https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/dashboards/cronjob-details.json
```

Then import via Grafana UI.

## Configure Alerts

### If using Prometheus Operator

```bash
kubectl apply -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/alerts/cronjob-alerts.yaml
```

### If using vanilla Prometheus

```bash
kubectl apply -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/alerts/cronjob-alerts-configmap.yaml
```

Then add the ConfigMap to your Prometheus rule files.

## Verify Installation

```bash
# Check pod is running
kubectl get pods -n monitoring -l app.kubernetes.io/name=k8s-cronjob-monitor

# Check logs
kubectl logs -n monitoring -l app.kubernetes.io/name=k8s-cronjob-monitor

# Port-forward to check metrics
kubectl port-forward -n monitoring svc/k8s-cronjob-monitor 8080:8080

# In another terminal
curl http://localhost:8080/metrics | grep cronjob_monitor
```

You should see metrics for all CronJobs in your cluster.

## Uninstallation

### Helm

```bash
helm uninstall k8s-cronjob-monitor --namespace monitoring
```

### Raw Manifests

```bash
kubectl delete -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/deploy/manifests/operator.yaml
```
