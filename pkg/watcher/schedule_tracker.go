package watcher

import (
	"context"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/varaxlabs/varax-monitor/pkg/metrics"
)

// ScheduleTracker monitors CronJob schedules and detects missed executions.
// Implements manager.Runnable so it can be added to a controller-runtime Manager.
type ScheduleTracker struct {
	client        client.Client
	collector     *metrics.Collector
	gracePeriod   time.Duration
	checkInterval time.Duration

	// Track which nextRun times have already been reported as missed,
	// keyed by "namespace/name". Prevents repeated counter increments.
	reportedMisses map[string]time.Time
	mu             sync.Mutex
}

// NewScheduleTracker creates a new schedule tracker.
func NewScheduleTracker(c client.Client, collector *metrics.Collector) *ScheduleTracker {
	return &ScheduleTracker{
		client:         c,
		collector:      collector,
		gracePeriod:    5 * time.Minute,
		checkInterval:  1 * time.Minute,
		reportedMisses: make(map[string]time.Time),
	}
}

// Start implements manager.Runnable. It runs the tracking loop until the context is cancelled.
func (t *ScheduleTracker) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("schedule-tracker")
	logger.Info("Starting schedule tracker",
		"gracePeriod", t.gracePeriod,
		"checkInterval", t.checkInterval,
	)

	t.checkAllSchedules(ctx)

	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.checkAllSchedules(ctx)
		case <-ctx.Done():
			logger.Info("Schedule tracker stopped")
			return nil
		}
	}
}

// NeedLeaderElection returns true so the tracker only runs on the leader.
func (t *ScheduleTracker) NeedLeaderElection() bool {
	return true
}

func (t *ScheduleTracker) checkAllSchedules(ctx context.Context) {
	logger := log.FromContext(ctx).WithName("schedule-tracker")

	var cronJobList batchv1.CronJobList
	if err := t.client.List(ctx, &cronJobList); err != nil {
		logger.Error(err, "Failed to list CronJobs")
		return
	}

	now := time.Now()
	for i := range cronJobList.Items {
		t.checkCronJobSchedule(ctx, &cronJobList.Items[i], now)
	}
}

func (t *ScheduleTracker) checkCronJobSchedule(ctx context.Context, cronJob *batchv1.CronJob, now time.Time) {
	logger := log.FromContext(ctx).WithName("schedule-tracker")

	// Skip suspended CronJobs
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		return
	}

	schedule, err := ParseSchedule(cronJob.Spec.Schedule)
	if err != nil {
		logger.V(1).Info("Failed to parse cron schedule",
			"namespace", cronJob.Namespace,
			"name", cronJob.Name,
			"schedule", cronJob.Spec.Schedule,
			"error", err,
		)
		return
	}

	// Calculate next run time
	var nextRun time.Time
	if cronJob.Status.LastScheduleTime != nil {
		nextRun = schedule.Next(cronJob.Status.LastScheduleTime.Time)
	} else {
		nextRun = schedule.Next(cronJob.CreationTimestamp.Time)
	}

	// Update next schedule timestamp metric
	t.collector.UpdateNextSchedule(cronJob.Namespace, cronJob.Name, nextRun.Unix())

	// Check for missed schedule
	if now.After(nextRun.Add(t.gracePeriod)) {
		hasActiveJobs := len(cronJob.Status.Active) > 0

		if !hasActiveJobs {
			if cronJob.Status.LastScheduleTime == nil || cronJob.Status.LastScheduleTime.Time.Before(nextRun) {
				key := cronJob.Namespace + "/" + cronJob.Name

				t.mu.Lock()
				lastReported := t.reportedMisses[key]
				alreadyReported := !lastReported.IsZero() && !lastReported.Before(nextRun)
				if !alreadyReported {
					t.reportedMisses[key] = nextRun
				}
				t.mu.Unlock()

				if !alreadyReported {
					logger.Info("Missed schedule detected",
						"namespace", cronJob.Namespace,
						"name", cronJob.Name,
						"schedule", cronJob.Spec.Schedule,
						"expectedRun", nextRun,
						"overdueBy", now.Sub(nextRun),
					)
					t.collector.RecordMissedSchedule(cronJob.Namespace, cronJob.Name)
				}
			}
		}
	}
}

// SetupWithManager registers the ScheduleTracker as a runnable with the manager.
func SetupScheduleTracker(mgr ctrl.Manager, collector *metrics.Collector) error {
	tracker := NewScheduleTracker(mgr.GetClient(), collector)
	return mgr.Add(tracker)
}
