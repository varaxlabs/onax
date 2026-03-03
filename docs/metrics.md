# Metrics Reference

k8s-cronjob-monitor exposes all metrics on the `/metrics` endpoint using the Prometheus exposition format.

## Metric Prefix

All metrics use the `cronjob_monitor_` prefix.

## Labels

All metrics include these labels:

| Label | Description |
|-------|-------------|
| `namespace` | Kubernetes namespace of the CronJob |
| `cronjob` | Name of the CronJob |

## Info Gauge

### cronjob_monitor_info

CronJob metadata information. Always set to `1`.

**Additional Labels:**
| Label | Description |
|-------|-------------|
| `schedule` | Cron schedule expression |
| `suspended` | Whether the CronJob is suspended (`true`/`false`) |

**Example Query:**
```promql
# All monitored CronJobs with their schedules
cronjob_monitor_info
```

## Gauges

### cronjob_monitor_execution_status

Current status of the last CronJob execution.

| Value | Meaning |
|-------|---------|
| 1 | Success |
| 0 | Failed |
| -1 | Unknown |

**Example Query:**
```promql
# All failed CronJobs
cronjob_monitor_execution_status == 0
```

### cronjob_monitor_execution_duration_seconds

Duration of the last CronJob execution in seconds.

**Example Query:**
```promql
# Top 10 slowest CronJobs
topk(10, cronjob_monitor_execution_duration_seconds)
```

### cronjob_monitor_last_success_timestamp

Unix timestamp of the last successful execution.

**Example Query:**
```promql
# Time since last success in hours
(time() - cronjob_monitor_last_success_timestamp) / 3600
```

### cronjob_monitor_last_failure_timestamp

Unix timestamp of the last failed execution.

### cronjob_monitor_next_schedule_timestamp

Expected Unix timestamp of the next scheduled execution.

**Example Query:**
```promql
# CronJobs scheduled to run in the next hour
cronjob_monitor_next_schedule_timestamp < (time() + 3600)
```

### cronjob_monitor_active_jobs

Current number of active Job pods for this CronJob.

**Example Query:**
```promql
# All currently running CronJobs
cronjob_monitor_active_jobs > 0
```

### cronjob_monitor_success_rate

Success rate as a decimal (0-1) over the monitoring period.

**Example Query:**
```promql
# CronJobs with less than 90% success rate
cronjob_monitor_success_rate < 0.9
```

### cronjob_monitor_schedule_delay_seconds

Difference between the scheduled start time and actual start time.

**Example Query:**
```promql
# CronJobs with significant schedule delays
cronjob_monitor_schedule_delay_seconds > 60
```

## Counters

### cronjob_monitor_executions_total

Total number of CronJob executions.

**Additional Label:**
| Label | Description |
|-------|-------------|
| `status` | Execution status: `success` or `failed` |

**Example Queries:**
```promql
# Total failures in the last 24 hours
sum(increase(cronjob_monitor_executions_total{status="failed"}[24h]))

# Execution rate per minute
rate(cronjob_monitor_executions_total[5m])
```

### cronjob_monitor_missed_schedules_total

Total number of missed schedules detected.

**Example Query:**
```promql
# Missed schedules in the last hour
increase(cronjob_monitor_missed_schedules_total[1h])
```

## Useful PromQL Queries

### Dashboard Queries

```promql
# Overall success rate across all CronJobs
avg(cronjob_monitor_success_rate)

# Total number of monitored CronJobs
count(cronjob_monitor_execution_status)

# Failed jobs in last 24 hours
sum(increase(cronjob_monitor_executions_total{status="failed"}[24h]))

# Currently running jobs
sum(cronjob_monitor_active_jobs)
```

### Alerting Queries

```promql
# CronJob hasn't succeeded in 48 hours
(time() - cronjob_monitor_last_success_timestamp) > 172800

# Execution taking over 1 hour
cronjob_monitor_execution_duration_seconds > 3600

# Success rate below 90%
cronjob_monitor_success_rate < 0.9
```

### Analysis Queries

```promql
# Average execution duration by namespace
avg by (namespace) (cronjob_monitor_execution_duration_seconds)

# Total executions per day
sum(increase(cronjob_monitor_executions_total[24h]))

# Failure rate by namespace
sum by (namespace) (rate(cronjob_monitor_executions_total{status="failed"}[1h])) /
sum by (namespace) (rate(cronjob_monitor_executions_total[1h]))
```
