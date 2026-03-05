package models

import (
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CronJobStatus represents the derived monitoring state for a CronJob.
type CronJobStatus struct {
	Namespace        string
	Name             string
	Schedule         string
	Suspended        bool
	LastExecStatus   float64 // 1=success, 0=failed, -1=unknown
	LastExecDuration float64 // seconds
	LastSuccessTime  float64 // unix timestamp
	LastFailureTime  float64 // unix timestamp
	NextScheduleTime float64 // unix timestamp
	ActiveJobCount   int
	SuccessCount     int64
	FailureCount     int64
	ScheduleDelay    float64 // seconds
}

// ComputeStatus derives monitoring state from a CronJob and its owned Jobs.
// ownedJobs should already be filtered to jobs owned by this CronJob.
func ComputeStatus(cronJob *batchv1.CronJob, ownedJobs []batchv1.Job) *CronJobStatus {
	status := &CronJobStatus{
		Namespace:      cronJob.Namespace,
		Name:           cronJob.Name,
		Schedule:       cronJob.Spec.Schedule,
		Suspended:      cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend,
		LastExecStatus: -1, // unknown until we find a completed job
		ActiveJobCount: len(cronJob.Status.Active),
	}

	if len(ownedJobs) == 0 {
		return status
	}

	// Sort jobs by start time, most recent first
	sort.Slice(ownedJobs, func(i, j int) bool {
		return JobStartTime(&ownedJobs[i]).After(JobStartTime(&ownedJobs[j]))
	})

	// Single pass: counts, last exec status/duration, schedule delay
	foundLastCompleted := false
	foundScheduleDelay := false

	for i := range ownedJobs {
		job := &ownedJobs[i]

		if IsJobSuccessful(job) {
			status.SuccessCount++
			if job.Status.CompletionTime != nil {
				ts := float64(job.Status.CompletionTime.Unix())
				if ts > status.LastSuccessTime {
					status.LastSuccessTime = ts
				}
			}
		} else if IsJobFailed(job) {
			status.FailureCount++
			failTime := jobFailureTime(job)
			if failTime > status.LastFailureTime {
				status.LastFailureTime = failTime
			}
		}

		if !foundLastCompleted && IsJobComplete(job) {
			if IsJobSuccessful(job) {
				status.LastExecStatus = 1
			} else {
				status.LastExecStatus = 0
			}
			if job.Status.CompletionTime != nil && job.Status.StartTime != nil {
				status.LastExecDuration = job.Status.CompletionTime.Sub(job.Status.StartTime.Time).Seconds()
			}
			foundLastCompleted = true
		}

		if !foundScheduleDelay && job.Status.StartTime != nil && cronJob.Status.LastScheduleTime != nil {
			delay := job.Status.StartTime.Sub(cronJob.Status.LastScheduleTime.Time).Seconds()
			if delay >= 0 {
				status.ScheduleDelay = delay
			}
			foundScheduleDelay = true
		}
	}

	return status
}

func jobFailureTime(job *batchv1.Job) float64 {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return float64(c.LastTransitionTime.Unix())
		}
	}
	if job.Status.StartTime != nil {
		return float64(job.Status.StartTime.Unix())
	}
	return float64(job.CreationTimestamp.Unix())
}

// JobStartTime returns the effective start time for sorting.
func JobStartTime(job *batchv1.Job) time.Time {
	if job.Status.StartTime != nil {
		return job.Status.StartTime.Time
	}
	return job.CreationTimestamp.Time
}

// IsJobComplete checks if a job has completed (success or failure).
func IsJobComplete(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsJobSuccessful checks if a completed job was successful.
func IsJobSuccessful(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsJobFailed checks if a job has failed.
func IsJobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// GetCronJobOwner returns the name and UID of the CronJob that owns a Job,
// or empty strings if none found.
func GetCronJobOwner(job *batchv1.Job) (string, types.UID) {
	for _, ref := range job.OwnerReferences {
		if ref.Kind == "CronJob" {
			return ref.Name, ref.UID
		}
	}
	return "", ""
}
