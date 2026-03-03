# Configuration Reference

## Helm Values

The following table lists the configurable parameters of the k8s-cronjob-monitor chart.

### General

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Container image repository | `ghcr.io/kubeshield/k8s-cronjob-monitor` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full chart name | `""` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create a service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` (auto-generated) |

### Metrics

| Parameter | Description | Default |
|-----------|-------------|---------|
| `metrics.port` | Metrics endpoint port | `8080` |
| `metrics.path` | Metrics endpoint path | `/metrics` |
| `metrics.serviceMonitor.enabled` | Enable ServiceMonitor | `true` |
| `metrics.serviceMonitor.interval` | Scrape interval | `15s` |
| `metrics.serviceMonitor.labels` | Additional ServiceMonitor labels | `{}` |
| `metrics.serviceMonitor.namespace` | ServiceMonitor namespace | `""` (release namespace) |

### Namespace Filtering

| Parameter | Description | Default |
|-----------|-------------|---------|
| `namespaces.include` | Only monitor CronJobs in these namespaces | `[]` (all) |
| `namespaces.exclude` | Exclude CronJobs in these namespaces | `[]` |

### Scheduling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `scheduling.gracePeriod` | Grace period before reporting missed schedule | `5m` |
| `scheduling.checkInterval` | How often to check for missed schedules | `1m` |

### Logging

| Parameter | Description | Default |
|-----------|-------------|---------|
| `logging.level` | Log level (info, debug) | `info` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |

### Pod Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext.runAsNonRoot` | Run as non-root | `true` |
| `podSecurityContext.runAsUser` | Run as user ID | `65532` |
| `podSecurityContext.fsGroup` | FS group ID | `65532` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

## Environment Variables

These can be set via Helm values or directly in the deployment:

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_PORT` | Port for the metrics endpoint | `:8080` |
| `LOG_LEVEL` | Logging verbosity | `info` |

## Example Configurations

### Minimal
```yaml
# Use all defaults - monitors all namespaces
helm install k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
  --namespace monitoring --create-namespace
```

### Production
```yaml
image:
  tag: v1.0.0

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

namespaces:
  include:
    - production
    - staging

scheduling:
  gracePeriod: 10m

metrics:
  serviceMonitor:
    enabled: true
    interval: 30s
```
