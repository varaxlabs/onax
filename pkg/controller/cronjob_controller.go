package controller

import (
	"context"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/metrics"
	"github.com/kubeshield/k8s-cronjob-monitor/pkg/models"
)

// CronJobReconciler reconciles CronJob objects and updates Prometheus metrics.
type CronJobReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Collector *metrics.Collector
}

// Reconcile fetches a CronJob, lists its owned Jobs, computes status, and updates metrics.
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cronJob batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("CronJob deleted, removing metrics", "cronjob", req.NamespacedName)
			r.Collector.RemoveCronJob(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// List all Jobs in the namespace
	var jobList batchv1.JobList
	if err := r.List(ctx, &jobList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	// Filter to jobs owned by this CronJob
	owned := filterOwnedJobs(jobList.Items, cronJob.Name, cronJob.UID)

	// Compute status
	status := models.ComputeStatus(&cronJob, owned)

	// Update metrics
	r.Collector.UpdateAll(status)

	// Requeue after 60s to keep metrics fresh
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager registers the reconciler with the manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(r.mapJobToCronJob)).
		Complete(r)
}

// mapJobToCronJob maps a Job event to a reconcile request for its owning CronJob.
func (r *CronJobReconciler) mapJobToCronJob(ctx context.Context, obj client.Object) []reconcile.Request {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil
	}

	name, _ := models.GetCronJobOwner(job)
	if name == "" {
		return nil
	}

	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Namespace: job.Namespace,
			Name:      name,
		}},
	}
}

// filterOwnedJobs returns jobs owned by the given CronJob, sorted most-recent-first.
func filterOwnedJobs(jobs []batchv1.Job, cronJobName string, cronJobUID types.UID) []batchv1.Job {
	var owned []batchv1.Job
	for _, job := range jobs {
		for _, ref := range job.OwnerReferences {
			if ref.Kind == "CronJob" && ref.Name == cronJobName && ref.UID == cronJobUID {
				owned = append(owned, job)
				break
			}
		}
	}

	sort.Slice(owned, func(i, j int) bool {
		ti := jobStartTime(&owned[i])
		tj := jobStartTime(&owned[j])
		return ti.After(tj)
	})

	return owned
}

func jobStartTime(job *batchv1.Job) time.Time {
	if job.Status.StartTime != nil {
		return job.Status.StartTime.Time
	}
	return job.CreationTimestamp.Time
}
