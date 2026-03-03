package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func boolPtr(b bool) *bool { return &b }

func newCronJob(ns, name, schedule string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			UID:       types.UID("cj-uid-1"),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
		},
	}
}

func newJob(ns, name string, owner string, ownerUID types.UID) batchv1.Job {
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: owner, UID: ownerUID},
			},
		},
	}
}

func TestComputeStatus_NoJobs(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	status := ComputeStatus(cj, nil)

	assert.Equal(t, "default", status.Namespace)
	assert.Equal(t, "test-cj", status.Name)
	assert.Equal(t, "*/5 * * * *", status.Schedule)
	assert.False(t, status.Suspended)
	assert.Equal(t, float64(-1), status.LastExecStatus)
	assert.Equal(t, int64(0), status.SuccessCount)
	assert.Equal(t, int64(0), status.FailureCount)
}

func TestComputeStatus_SuccessfulJob(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	now := time.Now()
	start := metav1.NewTime(now.Add(-30 * time.Second))
	end := metav1.NewTime(now)

	job := newJob("default", "test-cj-123", "test-cj", "cj-uid-1")
	job.Status = batchv1.JobStatus{
		StartTime:      &start,
		CompletionTime: &end,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
		},
	}

	status := ComputeStatus(cj, []batchv1.Job{job})

	assert.Equal(t, float64(1), status.LastExecStatus)
	assert.InDelta(t, 30.0, status.LastExecDuration, 0.1)
	assert.Equal(t, int64(1), status.SuccessCount)
	assert.Equal(t, int64(0), status.FailureCount)
	assert.Equal(t, float64(end.Unix()), status.LastSuccessTime)
}

func TestComputeStatus_FailedJob(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	now := time.Now()
	start := metav1.NewTime(now.Add(-10 * time.Second))
	end := metav1.NewTime(now)

	job := newJob("default", "test-cj-456", "test-cj", "cj-uid-1")
	job.Status = batchv1.JobStatus{
		StartTime:      &start,
		CompletionTime: &end,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(now)},
		},
	}

	status := ComputeStatus(cj, []batchv1.Job{job})

	assert.Equal(t, float64(0), status.LastExecStatus)
	assert.Equal(t, int64(0), status.SuccessCount)
	assert.Equal(t, int64(1), status.FailureCount)
	assert.Greater(t, status.LastFailureTime, float64(0))
}

func TestComputeStatus_MixedJobs(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	now := time.Now()

	successStart := metav1.NewTime(now.Add(-120 * time.Second))
	successEnd := metav1.NewTime(now.Add(-90 * time.Second))
	successJob := newJob("default", "test-cj-1", "test-cj", "cj-uid-1")
	successJob.Status = batchv1.JobStatus{
		StartTime:      &successStart,
		CompletionTime: &successEnd,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
		},
	}

	failStart := metav1.NewTime(now.Add(-30 * time.Second))
	failEnd := metav1.NewTime(now)
	failJob := newJob("default", "test-cj-2", "test-cj", "cj-uid-1")
	failJob.Status = batchv1.JobStatus{
		StartTime:      &failStart,
		CompletionTime: &failEnd,
		Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(now)},
		},
	}

	status := ComputeStatus(cj, []batchv1.Job{successJob, failJob})

	// Most recent completed job is the failed one
	assert.Equal(t, float64(0), status.LastExecStatus)
	assert.Equal(t, int64(1), status.SuccessCount)
	assert.Equal(t, int64(1), status.FailureCount)
}

func TestComputeStatus_ActiveJobs(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	cj.Status.Active = []corev1.ObjectReference{{Name: "test-cj-active"}}

	status := ComputeStatus(cj, nil)
	assert.Equal(t, 1, status.ActiveJobCount)
}

func TestComputeStatus_Suspended(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	cj.Spec.Suspend = boolPtr(true)

	status := ComputeStatus(cj, nil)
	assert.True(t, status.Suspended)
}

func TestComputeStatus_ScheduleDelay(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")
	now := time.Now()
	scheduledTime := metav1.NewTime(now.Add(-10 * time.Second))
	cj.Status.LastScheduleTime = &scheduledTime

	startTime := metav1.NewTime(now.Add(-5 * time.Second))
	job := newJob("default", "test-cj-1", "test-cj", "cj-uid-1")
	job.Status = batchv1.JobStatus{
		StartTime: &startTime,
	}

	status := ComputeStatus(cj, []batchv1.Job{job})
	assert.InDelta(t, 5.0, status.ScheduleDelay, 0.1)
}

func TestComputeStatus_NeverRan(t *testing.T) {
	cj := newCronJob("default", "test-cj", "*/5 * * * *")

	status := ComputeStatus(cj, []batchv1.Job{})

	assert.Equal(t, float64(-1), status.LastExecStatus)
	assert.Equal(t, float64(0), status.LastExecDuration)
}

func TestIsJobComplete(t *testing.T) {
	tests := []struct {
		name     string
		job      batchv1.Job
		expected bool
	}{
		{
			name: "completed job",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name: "failed job",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name:     "active job",
			job:      batchv1.Job{Status: batchv1.JobStatus{Active: 1}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsJobComplete(&tt.job))
		})
	}
}

func TestIsJobSuccessful(t *testing.T) {
	job := &batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			},
		},
	}
	assert.True(t, IsJobSuccessful(job))

	failedJob := &batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
			},
		},
	}
	assert.False(t, IsJobSuccessful(failedJob))
}

func TestIsJobFailed(t *testing.T) {
	job := &batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
			},
		},
	}
	assert.True(t, IsJobFailed(job))
}

func TestGetCronJobOwner(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "my-cronjob", UID: "uid-123"},
			},
		},
	}

	name, uid := GetCronJobOwner(job)
	require.Equal(t, "my-cronjob", name)
	require.Equal(t, types.UID("uid-123"), uid)

	// No owner
	orphan := &batchv1.Job{}
	name, uid = GetCronJobOwner(orphan)
	assert.Empty(t, name)
	assert.Empty(t, uid)
}
