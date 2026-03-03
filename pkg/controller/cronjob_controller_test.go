package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	io_prometheus_client "github.com/prometheus/client_model/go"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/metrics"
)

func getGaugeValue(g interface{ Write(*io_prometheus_client.Metric) error }) float64 {
	m := &io_prometheus_client.Metric{}
	_ = g.Write(m)
	return m.GetGauge().GetValue()
}

func newTestReconciler(objs ...runtime.Object) (*CronJobReconciler, *metrics.Collector) {
	scheme := runtime.NewScheme()
	_ = batchv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
	collector := metrics.NewCollector()

	return &CronJobReconciler{
		Client:    c,
		Scheme:    scheme,
		Collector: collector,
	}, collector
}

func TestReconcile_CronJobCreated(t *testing.T) {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj",
			UID:       "cj-uid-1",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	r, collector := newTestReconciler(cj)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-cj"},
	})

	require.NoError(t, err)
	assert.Equal(t, 60*time.Second, result.RequeueAfter)

	// Check that info metric is set
	mfs, _ := collector.Registry.Gather()
	foundInfo := false
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_info" {
			foundInfo = true
			break
		}
	}
	assert.True(t, foundInfo, "info metric should exist after reconcile")
}

func TestReconcile_CronJobWithSuccessfulJob(t *testing.T) {
	cjUID := types.UID("cj-uid-1")
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj",
			UID:       cjUID,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	now := time.Now()
	start := metav1.NewTime(now.Add(-30 * time.Second))
	end := metav1.NewTime(now)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj-123",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "test-cj", UID: cjUID},
			},
		},
		Status: batchv1.JobStatus{
			StartTime:      &start,
			CompletionTime: &end,
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			},
		},
	}

	r, collector := newTestReconciler(cj, job)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-cj"},
	})

	require.NoError(t, err)

	// Verify success metrics
	mfs, _ := collector.Registry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_execution_status" {
			for _, m := range mf.GetMetric() {
				assert.Equal(t, float64(1), m.GetGauge().GetValue())
			}
		}
		if mf.GetName() == "cronjob_monitor_executions_total" {
			for _, m := range mf.GetMetric() {
				for _, l := range m.GetLabel() {
					if l.GetName() == "status" && l.GetValue() == "success" {
						assert.Equal(t, float64(1), m.GetCounter().GetValue())
					}
				}
			}
		}
	}
}

func TestReconcile_CronJobWithFailedJob(t *testing.T) {
	cjUID := types.UID("cj-uid-1")
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj",
			UID:       cjUID,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	now := time.Now()
	start := metav1.NewTime(now.Add(-10 * time.Second))
	end := metav1.NewTime(now)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj-456",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "test-cj", UID: cjUID},
			},
		},
		Status: batchv1.JobStatus{
			StartTime:      &start,
			CompletionTime: &end,
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(now)},
			},
		},
	}

	r, collector := newTestReconciler(cj, job)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-cj"},
	})

	require.NoError(t, err)

	mfs, _ := collector.Registry.Gather()
	for _, mf := range mfs {
		if mf.GetName() == "cronjob_monitor_execution_status" {
			for _, m := range mf.GetMetric() {
				assert.Equal(t, float64(0), m.GetGauge().GetValue())
			}
		}
	}
}

func TestReconcile_CronJobDeleted(t *testing.T) {
	// Create reconciler with no CronJob (simulating deletion)
	r, collector := newTestReconciler()

	// First, set up some metrics
	ts := metrics.TestStatus("default", "test-cj")
	collector.UpdateAll(&ts)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-cj"},
	})

	require.NoError(t, err)

	// Check that metrics were removed
	mfs, _ := collector.Registry.Gather()
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "cronjob" {
					assert.NotEqual(t, "test-cj", l.GetValue(),
						"metric %s should be cleaned up after deletion", mf.GetName())
				}
			}
		}
	}
}

func TestReconcile_JobEventTriggersCronJobReconcile(t *testing.T) {
	cjUID := types.UID("cj-uid-1")

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cj-789",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "test-cj", UID: cjUID},
			},
		},
	}

	r, _ := newTestReconciler()

	requests := r.mapJobToCronJob(context.Background(), job)
	require.Len(t, requests, 1)
	assert.Equal(t, "default", requests[0].Namespace)
	assert.Equal(t, "test-cj", requests[0].Name)
}

func TestMapJobToCronJob_NoOwner(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "standalone-job",
		},
	}

	r, _ := newTestReconciler()

	requests := r.mapJobToCronJob(context.Background(), job)
	assert.Empty(t, requests)
}
