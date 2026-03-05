package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/varaxlabs/varax-monitor/pkg/metrics"
)

func init() {
	// Set up a logger so log.FromContext works in tests
	zap.New()
}

func boolPtr(b bool) *bool { return &b }

func newTestTracker(objs ...runtime.Object) (*ScheduleTracker, *metrics.Collector) {
	scheme := runtime.NewScheme()
	_ = batchv1.AddToScheme(scheme)

	clientObjs := make([]runtime.Object, len(objs))
	copy(clientObjs, objs)

	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(clientObjs...).Build()
	collector := metrics.NewCollector()
	tracker := NewScheduleTracker(c, collector)
	tracker.checkInterval = 100 * time.Millisecond
	return tracker, collector
}

func TestCheckCronJobSchedule_MissedSchedule(t *testing.T) {
	// CronJob that should have run 10 minutes ago with 5 minute grace period
	pastTime := metav1.NewTime(time.Now().Add(-20 * time.Minute))
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "test-cj",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: &pastTime,
		},
	}

	tracker, collector := newTestTracker(cj)
	ctx := context.Background()

	tracker.checkCronJobSchedule(ctx, cj, time.Now())

	// Should detect missed schedule and record it
	mfs, _ := collector.TestRegistry.Gather()
	found := false
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_missed_schedules_total" {
			found = true
			break
		}
	}
	assert.True(t, found, "missed_schedules_total metric should exist")
}

func TestCheckCronJobSchedule_WithinGracePeriod(t *testing.T) {
	// CronJob that was just recently scheduled (within grace period)
	recentTime := metav1.NewTime(time.Now().Add(-2 * time.Minute))
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "test-cj",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: &recentTime,
		},
	}

	tracker, collector := newTestTracker(cj)
	ctx := context.Background()

	tracker.checkCronJobSchedule(ctx, cj, time.Now())

	// Should NOT record missed schedule
	mfs, _ := collector.TestRegistry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_missed_schedules_total" {
			for _, m := range mf.GetMetric() {
				assert.Equal(t, float64(0), m.GetCounter().GetValue(),
					"should not have recorded a missed schedule within grace period")
			}
		}
	}
}

func TestCheckCronJobSchedule_Suspended(t *testing.T) {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			Suspend:  boolPtr(true),
		},
	}

	tracker, collector := newTestTracker(cj)
	ctx := context.Background()

	tracker.checkCronJobSchedule(ctx, cj, time.Now())

	// Suspended CronJobs should not produce any missed schedule metrics
	mfs, _ := collector.TestRegistry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_missed_schedules_total" {
			assert.Empty(t, mf.GetMetric(), "suspended CronJob should not produce missed schedule metrics")
		}
	}
}

func TestCheckCronJobSchedule_InvalidCronExpression(t *testing.T) {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "not-a-valid-cron",
		},
	}

	tracker, _ := newTestTracker(cj)
	ctx := context.Background()

	// Should not panic
	tracker.checkCronJobSchedule(ctx, cj, time.Now())
}

func TestCheckCronJobSchedule_ActiveJobsPrevents(t *testing.T) {
	pastTime := metav1.NewTime(time.Now().Add(-20 * time.Minute))
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "test-cj",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: &pastTime,
			Active: []corev1.ObjectReference{
				{Name: "test-cj-running"},
			},
		},
	}

	tracker, collector := newTestTracker(cj)
	ctx := context.Background()

	tracker.checkCronJobSchedule(ctx, cj, time.Now())

	// Should NOT record missed schedule because there's an active job
	mfs, _ := collector.TestRegistry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_missed_schedules_total" {
			for _, m := range mf.GetMetric() {
				assert.Equal(t, float64(0), m.GetCounter().GetValue(),
					"should not record missed schedule when active jobs exist")
			}
		}
	}
}

func TestStart_CancelledContext(t *testing.T) {
	tracker, _ := newTestTracker()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- tracker.Start(ctx)
	}()

	// Cancel quickly
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("tracker did not stop after context cancellation")
	}
}
