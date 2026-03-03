# k8s-cronjob-monitor

[![GitHub Stars](https://img.shields.io/github/stars/kubeshield/k8s-cronjob-monitor)](https://github.com/kubeshield/k8s-cronjob-monitor)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeshield/k8s-cronjob-monitor)](https://goreportcard.com/report/github.com/kubeshield/k8s-cronjob-monitor)

**Dead-simple CronJob monitoring for Kubernetes with Prometheus and Grafana**

Stop paying for SaaS CronJob monitoring. Monitor your Kubernetes CronJobs with your existing Prometheus and Grafana stack—completely free and self-hosted.

## Features

- **Zero Configuration** - Auto-discovers all CronJobs in your cluster
- **Prometheus Metrics** - Native Prometheus integration
- **Beautiful Dashboards** - Pre-built Grafana dashboards included
- **One Command Install** - Up and running in 30 seconds
- **Self-Hosted** - All data stays in your cluster
- **100% Free** - Apache 2.0 licensed
- **Lightweight** - <50MB memory, <0.05 CPU cores

## Quick Start

```bash
# Add Helm repository
helm repo add k8s-cronjob-monitor https://kubeshield.github.io/k8s-cronjob-monitor

# Install
helm install k8s-cronjob-monitor k8s-cronjob-monitor/k8s-cronjob-monitor \
  --namespace monitoring \
  --create-namespace

# Verify installation
kubectl get pods -n monitoring -l app.kubernetes.io/name=k8s-cronjob-monitor
```

That's it! Your CronJobs are now being monitored.

## What You Get

### Metrics

All metrics use the `cronjob_monitor_` prefix:

| Metric | Type | Description |
|--------|------|-------------|
| `cronjob_monitor_info` | Gauge | CronJob metadata (schedule, suspended) |
| `cronjob_monitor_execution_status` | Gauge | Last execution status (1=success, 0=failed, -1=unknown) |
| `cronjob_monitor_execution_duration_seconds` | Gauge | Duration of last execution |
| `cronjob_monitor_executions_total` | Counter | Total executions by status |
| `cronjob_monitor_last_success_timestamp` | Gauge | Timestamp of last success |
| `cronjob_monitor_last_failure_timestamp` | Gauge | Timestamp of last failure |
| `cronjob_monitor_next_schedule_timestamp` | Gauge | Expected next run time |
| `cronjob_monitor_missed_schedules_total` | Counter | Total missed schedules |
| `cronjob_monitor_active_jobs` | Gauge | Currently running jobs |
| `cronjob_monitor_success_rate` | Gauge | Success rate (0-1) |
| `cronjob_monitor_schedule_delay_seconds` | Gauge | Delay between scheduled and actual start |

### Dashboards

Import our pre-built Grafana dashboards:

1. **[CronJob Overview](dashboards/cronjob-overview.json)** - All CronJobs at a glance
2. **[CronJob Details](dashboards/cronjob-details.json)** - Per-job deep dive

### Alerts

Pre-configured Prometheus alerts for:

- Failed executions (`CronJobFailed`)
- Missed schedules (`CronJobMissedSchedule`)
- Slow execution >1 hour (`CronJobSlowExecution`)
- Low success rate <90% (`CronJobLowSuccessRate`)
- No recent success in 48 hours (`CronJobNoRecentSuccess`)

```bash
# Apply alert rules (Prometheus Operator)
kubectl apply -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/alerts/cronjob-alerts.yaml
```

## Comparison

| Feature | k8s-cronjob-monitor | Cronitor | Healthchecks.io | DIY Prometheus |
|---------|---------------------|----------|-----------------|----------------|
| **Cost** | **Free** | $21-449/mo | $0-80/mo | Free (20hr setup) |
| **Setup time** | **1 command** | Manual per job | Manual per job | Hours of work |
| **Auto-discovery** | **Yes** | No | No | No |
| **Data location** | **Your cluster** | Their SaaS | Their SaaS | Your cluster |
| **Kubernetes-native** | **Yes** | No | No | Manual |
| **Pre-built dashboards** | **Yes** | Yes | No | No |

## Configuration

### Helm Values

```yaml
# values.yaml
image:
  repository: ghcr.io/kubeshield/k8s-cronjob-monitor
  tag: v1.0.0

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

metrics:
  port: 8080
  serviceMonitor:
    enabled: true
    interval: 15s

logging:
  level: info  # or "debug"
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_PORT` | `:8080` | Port for metrics endpoint |
| `LOG_LEVEL` | `info` | Log level (info, debug) |

## Development

```bash
# Clone repository
git clone https://github.com/kubeshield/k8s-cronjob-monitor.git
cd k8s-cronjob-monitor

# Install dependencies
go mod download

# Run locally (requires kubeconfig)
make run

# Build binary
make build

# Build Docker image
make docker-build
```

## RBAC Permissions

The operator requires minimal read-only permissions:

```yaml
rules:
  - apiGroups: ["batch"]
    resources: ["cronjobs", "jobs"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["get", "list", "watch"]
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Metrics Reference](docs/metrics.md)
- [Troubleshooting](docs/troubleshooting.md)

## Contributing

We love contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

Apache 2.0 - See [LICENSE](LICENSE)

## Related Projects

- **[KubeShield](https://kubeshield.io)** - Kubernetes security and compliance tools

---

**Built with love by [KubeShield](https://kubeshield.io)**
