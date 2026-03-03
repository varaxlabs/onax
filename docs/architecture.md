# Architecture

## Overview

k8s-cronjob-monitor is a Kubernetes operator built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) that monitors CronJob resources and exposes Prometheus metrics.

## Components

```
┌─────────────────────────────────────────────┐
│              controller-runtime Manager     │
│                                             │
│  ┌─────────────────────┐  ┌──────────────┐ │
│  │  CronJobReconciler  │  │  Schedule     │ │
│  │                     │  │  Tracker      │ │
│  │  Watches:           │  │              │ │
│  │  - CronJobs         │  │  Detects     │ │
│  │  - Jobs             │  │  missed      │ │
│  │                     │  │  schedules   │ │
│  └────────┬────────────┘  └──────┬───────┘ │
│           │                      │          │
│           ▼                      ▼          │
│  ┌──────────────────────────────────────┐   │
│  │         Metrics Collector            │   │
│  │                                      │   │
│  │  11 Prometheus metrics:              │   │
│  │  - 8 gauges                          │   │
│  │  - 2 counters                        │   │
│  │  - 1 info gauge                      │   │
│  └──────────────────────────────────────┘   │
│                                             │
│  ┌──────────────┐  ┌────────────────────┐   │
│  │ :8080/metrics│  │ :8081/healthz      │   │
│  │              │  │ :8081/readyz       │   │
│  └──────────────┘  └────────────────────┘   │
└─────────────────────────────────────────────┘
```

## Reconciliation Flow

1. **CronJob Event** (create/update/delete) or **Job Event** triggers reconciliation
2. `CronJobReconciler.Reconcile()` is called with the CronJob's NamespacedName
3. Reconciler fetches the CronJob and lists all Jobs in the namespace
4. Jobs are filtered to only those owned by this CronJob (via OwnerReferences)
5. `models.ComputeStatus()` derives monitoring state from the K8s objects
6. `Collector.UpdateAll()` updates all Prometheus metrics atomically
7. Result is requeued after 60 seconds for freshness

## Job-to-CronJob Mapping

When a Job event occurs, the controller maps it to the owning CronJob via `OwnerReferences`. This ensures that Job completions/failures immediately trigger a CronJob reconciliation and metric update.

## Schedule Tracking

The `ScheduleTracker` runs as a separate goroutine (via `manager.Runnable`) and:
1. Lists all CronJobs every check interval (default: 1 minute)
2. Parses each CronJob's cron schedule
3. Computes the expected next run time
4. If the next run time + grace period has passed without execution, records a missed schedule

## Metrics Design

All metrics use the `cronjob_monitor_` prefix and are registered with an isolated Prometheus registry to avoid test conflicts.

### Counter Idempotency

The collector tracks `lastKnownSuccessCount` and `lastKnownFailureCount` per CronJob. On each reconciliation, only the delta is added to the Prometheus counters. This prevents double-counting when the same Job is observed multiple times.

## RBAC

The operator requires minimal read-only permissions:
- `batch/cronjobs`: get, list, watch
- `batch/jobs`: get, list, watch
- `core/pods`: get, list, watch
- `core/pods/log`: get
