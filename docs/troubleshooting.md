# Troubleshooting Guide

## Common Issues

### No Metrics Appearing

**Symptoms:** Prometheus shows no `cronjob_monitor_*` metrics.

**Possible causes:**

1. **Operator not running**
   ```bash
   kubectl get pods -n monitoring -l app.kubernetes.io/name=k8s-cronjob-monitor
   ```
   Check the pod is in `Running` state.

2. **Prometheus not scraping**
   - Check Prometheus targets: Status > Targets
   - Look for `k8s-cronjob-monitor` target
   - If missing, check ServiceMonitor or scrape config

3. **No CronJobs in cluster**
   ```bash
   kubectl get cronjobs --all-namespaces
   ```
   Metrics only appear when CronJobs exist.

4. **RBAC issues**
   ```bash
   kubectl logs -n monitoring -l app.kubernetes.io/name=k8s-cronjob-monitor
   ```
   Look for permission errors.

### CronJob Failed

**Alert:** `CronJobFailed` (severity: warning)

**Runbook:**

1. Check CronJob status:
   ```bash
   kubectl describe cronjob <name> -n <namespace>
   ```

2. Check recent Job logs:
   ```bash
   kubectl get jobs -n <namespace> | grep <cronjob-name>
   kubectl logs job/<job-name> -n <namespace>
   ```

3. Check Pod events:
   ```bash
   kubectl get events -n <namespace> --field-selector involvedObject.kind=Job
   ```

4. Common failure reasons:
   - Image pull errors
   - Resource limits exceeded
   - Command/script errors
   - Missing secrets/configmaps

### Missed Schedule

**Alert:** `CronJobMissedSchedule` (severity: warning)

**Runbook:**

1. Check CronJob events:
   ```bash
   kubectl describe cronjob <name> -n <namespace>
   ```
   Look for "Cannot determine if job needs to be started" or similar.

2. Common causes:
   - Cluster was down during scheduled time
   - CronJob controller issues
   - Starting deadline too short
   - Too many concurrent jobs

3. Fix: Increase `startingDeadlineSeconds`:
   ```yaml
   spec:
     startingDeadlineSeconds: 300
   ```

### Slow Execution

**Alert:** `CronJobSlowExecution` (severity: warning)

**Runbook:**

1. Check current execution duration:
   ```promql
   cronjob_monitor_execution_duration_seconds{cronjob="<name>"}
   ```

2. Common causes:
   - Increased data volume
   - External service slowdown
   - Resource contention
   - Network issues

3. Check Pod resource usage:
   ```bash
   kubectl top pod -n <namespace> | grep <cronjob-name>
   ```

4. Consider setting `activeDeadlineSeconds`:
   ```yaml
   spec:
     jobTemplate:
       spec:
         activeDeadlineSeconds: 3600
   ```

### Low Success Rate

**Alert:** `CronJobLowSuccessRate` (severity: critical)

**Runbook:**

1. Check failure patterns:
   ```promql
   rate(cronjob_monitor_executions_total{cronjob="<name>", status="failed"}[24h])
   ```

2. Review recent failures:
   ```bash
   kubectl get jobs -n <namespace> | grep <cronjob-name>
   # Check failed jobs
   kubectl describe job <failed-job-name> -n <namespace>
   ```

3. Look for patterns:
   - Time-based (certain hours)
   - Resource-based (memory/CPU spikes)
   - External dependency failures

### No Recent Success

**Alert:** `CronJobNoRecentSuccess` (severity: critical)

**Runbook:**

1. Check if CronJob is suspended:
   ```bash
   kubectl get cronjob <name> -n <namespace> -o jsonpath='{.spec.suspend}'
   ```

2. Check concurrency policy:
   ```bash
   kubectl get cronjob <name> -n <namespace> -o jsonpath='{.spec.concurrencyPolicy}'
   ```
   If `Forbid`, a running job may block new ones.

3. Check last schedule time:
   ```promql
   cronjob_monitor_last_success_timestamp{cronjob="<name>"}
   ```

4. Check for missed schedules:
   ```promql
   increase(cronjob_monitor_missed_schedules_total{cronjob="<name>"}[24h])
   ```

5. Check cluster resources:
   ```bash
   kubectl describe nodes | grep -A 5 "Allocated resources"
   ```

## Debug Mode

Enable debug logging for more verbose output:

```bash
helm upgrade k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
  --set logging.level=debug
```

## Getting Help

1. Check [GitHub Issues](https://github.com/kubeshield/k8s-cronjob-monitor/issues)
2. Search existing discussions
3. Open a new issue with:
   - Kubernetes version
   - k8s-cronjob-monitor version
   - Relevant logs
   - Steps to reproduce
