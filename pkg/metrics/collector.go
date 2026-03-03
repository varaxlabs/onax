package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/models"
)

// Collector holds all Prometheus metrics for CronJob monitoring.
type Collector struct {
	Registry *prometheus.Registry

	// Gauges
	executionStatus   *prometheus.GaugeVec
	executionDuration *prometheus.GaugeVec
	lastSuccessTime   *prometheus.GaugeVec
	lastFailureTime   *prometheus.GaugeVec
	nextScheduleTime  *prometheus.GaugeVec
	activeJobs        *prometheus.GaugeVec
	successRate       *prometheus.GaugeVec
	scheduleDelay     *prometheus.GaugeVec

	// Info gauge
	info *prometheus.GaugeVec

	// Counters
	executionsTotal *prometheus.CounterVec
	missedSchedules *prometheus.CounterVec

	// Internal tracking for counter idempotency
	lastKnownSuccess map[string]int64
	lastKnownFailure map[string]int64
	mu               sync.Mutex
}

// NewCollector creates a new metrics collector with an isolated registry.
func NewCollector() *Collector {
	return NewCollectorWithRegistry(prometheus.NewRegistry())
}

// NewCollectorWithRegistry creates a new metrics collector with the given registry.
func NewCollectorWithRegistry(reg *prometheus.Registry) *Collector {
	c := &Collector{
		Registry:         reg,
		lastKnownSuccess: make(map[string]int64),
		lastKnownFailure: make(map[string]int64),
	}

	c.executionStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_execution_status",
			Help: "Current status of last CronJob execution (1=success, 0=failed, -1=unknown)",
		},
		[]string{"namespace", "cronjob"},
	)

	c.executionDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_execution_duration_seconds",
			Help: "Duration of last CronJob execution in seconds",
		},
		[]string{"namespace", "cronjob"},
	)

	c.lastSuccessTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_last_success_timestamp",
			Help: "Timestamp of last successful execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.lastFailureTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_last_failure_timestamp",
			Help: "Timestamp of last failed execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.nextScheduleTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_next_schedule_timestamp",
			Help: "Expected timestamp of next execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.activeJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_active_jobs",
			Help: "Current number of active Job pods",
		},
		[]string{"namespace", "cronjob"},
	)

	c.successRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_success_rate",
			Help: "Success rate (successful / total) over last 24 hours",
		},
		[]string{"namespace", "cronjob"},
	)

	c.scheduleDelay = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_schedule_delay_seconds",
			Help: "Difference between scheduled and actual start time",
		},
		[]string{"namespace", "cronjob"},
	)

	c.info = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_info",
			Help: "CronJob metadata information",
		},
		[]string{"namespace", "cronjob", "schedule", "suspended"},
	)

	c.executionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cronjob_monitor_executions_total",
			Help: "Total number of CronJob executions",
		},
		[]string{"namespace", "cronjob", "status"},
	)

	c.missedSchedules = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cronjob_monitor_missed_schedules_total",
			Help: "Total number of missed schedules",
		},
		[]string{"namespace", "cronjob"},
	)

	reg.MustRegister(
		c.executionStatus,
		c.executionDuration,
		c.lastSuccessTime,
		c.lastFailureTime,
		c.nextScheduleTime,
		c.activeJobs,
		c.successRate,
		c.scheduleDelay,
		c.info,
		c.executionsTotal,
		c.missedSchedules,
	)

	return c
}

// SetInfo sets the info metric for a CronJob.
func (c *Collector) SetInfo(namespace, name, schedule string, suspended bool) {
	suspendedStr := "false"
	if suspended {
		suspendedStr = "true"
	}
	// Delete any previous info labels for this CronJob (schedule/suspended may change)
	c.info.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "cronjob": name})
	c.info.WithLabelValues(namespace, name, schedule, suspendedStr).Set(1)
}

// UpdateAll sets all metrics from a CronJobStatus.
func (c *Collector) UpdateAll(status *models.CronJobStatus) {
	ns, name := status.Namespace, status.Name

	c.executionStatus.WithLabelValues(ns, name).Set(status.LastExecStatus)
	c.executionDuration.WithLabelValues(ns, name).Set(status.LastExecDuration)
	c.lastSuccessTime.WithLabelValues(ns, name).Set(status.LastSuccessTime)
	c.lastFailureTime.WithLabelValues(ns, name).Set(status.LastFailureTime)
	c.activeJobs.WithLabelValues(ns, name).Set(float64(status.ActiveJobCount))
	c.scheduleDelay.WithLabelValues(ns, name).Set(status.ScheduleDelay)

	// Counter idempotency: only increment by the delta
	c.mu.Lock()
	key := ns + "/" + name

	prevSuccess := c.lastKnownSuccess[key]
	if status.SuccessCount > prevSuccess {
		c.executionsTotal.WithLabelValues(ns, name, "success").Add(float64(status.SuccessCount - prevSuccess))
		c.lastKnownSuccess[key] = status.SuccessCount
	}

	prevFailure := c.lastKnownFailure[key]
	if status.FailureCount > prevFailure {
		c.executionsTotal.WithLabelValues(ns, name, "failed").Add(float64(status.FailureCount - prevFailure))
		c.lastKnownFailure[key] = status.FailureCount
	}
	c.mu.Unlock()

	// Success rate
	total := status.SuccessCount + status.FailureCount
	if total == 0 {
		c.successRate.WithLabelValues(ns, name).Set(1.0)
	} else {
		c.successRate.WithLabelValues(ns, name).Set(float64(status.SuccessCount) / float64(total))
	}

	// Info metric
	c.SetInfo(ns, name, status.Schedule, status.Suspended)
}

// UpdateNextSchedule updates the next expected run timestamp.
func (c *Collector) UpdateNextSchedule(namespace, cronjob string, timestamp int64) {
	c.nextScheduleTime.WithLabelValues(namespace, cronjob).Set(float64(timestamp))
}

// RecordMissedSchedule increments the missed schedule counter.
func (c *Collector) RecordMissedSchedule(namespace, cronjob string) {
	c.missedSchedules.WithLabelValues(namespace, cronjob).Inc()
}

// RegisterWithPrometheus registers all metric collectors with the default prometheus registry.
// This is used when integrating with controller-runtime's built-in metrics server.
func (c *Collector) RegisterWithPrometheus() {
	prometheus.MustRegister(
		c.executionStatus,
		c.executionDuration,
		c.lastSuccessTime,
		c.lastFailureTime,
		c.nextScheduleTime,
		c.activeJobs,
		c.successRate,
		c.scheduleDelay,
		c.info,
		c.executionsTotal,
		c.missedSchedules,
	)
}

// TestStatus creates a minimal CronJobStatus for testing.
func TestStatus(namespace, name string) models.CronJobStatus {
	return models.CronJobStatus{
		Namespace:      namespace,
		Name:           name,
		Schedule:       "*/5 * * * *",
		LastExecStatus: 1,
		SuccessCount:   1,
	}
}

// RemoveCronJob cleans up all metrics when a CronJob is deleted.
func (c *Collector) RemoveCronJob(namespace, name string) {
	labels := prometheus.Labels{"namespace": namespace, "cronjob": name}

	c.executionStatus.Delete(labels)
	c.executionDuration.Delete(labels)
	c.lastSuccessTime.Delete(labels)
	c.lastFailureTime.Delete(labels)
	c.nextScheduleTime.Delete(labels)
	c.activeJobs.Delete(labels)
	c.successRate.Delete(labels)
	c.scheduleDelay.Delete(labels)

	c.info.DeletePartialMatch(labels)

	c.executionsTotal.Delete(prometheus.Labels{"namespace": namespace, "cronjob": name, "status": "success"})
	c.executionsTotal.Delete(prometheus.Labels{"namespace": namespace, "cronjob": name, "status": "failed"})
	c.missedSchedules.Delete(labels)

	c.mu.Lock()
	key := namespace + "/" + name
	delete(c.lastKnownSuccess, key)
	delete(c.lastKnownFailure, key)
	c.mu.Unlock()
}
