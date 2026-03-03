package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/models"
)

func getGaugeValue(g *prometheus.GaugeVec, labels ...string) float64 {
	m := &io_prometheus_client.Metric{}
	_ = g.WithLabelValues(labels...).Write(m)
	return m.GetGauge().GetValue()
}

func getCounterValue(c *prometheus.CounterVec, labels ...string) float64 {
	m := &io_prometheus_client.Metric{}
	_ = c.WithLabelValues(labels...).Write(m)
	return m.GetCounter().GetValue()
}

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	require.NotNil(t, c)
	require.NotNil(t, c.Registry)
}

func TestUpdateAll_Success(t *testing.T) {
	c := NewCollector()

	status := &models.CronJobStatus{
		Namespace:        "default",
		Name:             "test-cj",
		Schedule:         "*/5 * * * *",
		Suspended:        false,
		LastExecStatus:   1,
		LastExecDuration: 30.5,
		LastSuccessTime:  1700000000,
		LastFailureTime:  0,
		ActiveJobCount:   0,
		SuccessCount:     5,
		FailureCount:     0,
		ScheduleDelay:    2.3,
	}

	c.UpdateAll(status)

	assert.Equal(t, float64(1), getGaugeValue(c.executionStatus, "default", "test-cj"))
	assert.Equal(t, 30.5, getGaugeValue(c.executionDuration, "default", "test-cj"))
	assert.Equal(t, float64(1700000000), getGaugeValue(c.lastSuccessTime, "default", "test-cj"))
	assert.Equal(t, float64(0), getGaugeValue(c.activeJobs, "default", "test-cj"))
	assert.Equal(t, 2.3, getGaugeValue(c.scheduleDelay, "default", "test-cj"))
	assert.Equal(t, float64(1), getGaugeValue(c.successRate, "default", "test-cj"))
	assert.Equal(t, float64(5), getCounterValue(c.executionsTotal, "default", "test-cj", "success"))
}

func TestUpdateAll_Failure(t *testing.T) {
	c := NewCollector()

	status := &models.CronJobStatus{
		Namespace:      "default",
		Name:           "test-cj",
		Schedule:       "*/5 * * * *",
		LastExecStatus: 0,
		SuccessCount:   3,
		FailureCount:   2,
	}

	c.UpdateAll(status)

	assert.Equal(t, float64(0), getGaugeValue(c.executionStatus, "default", "test-cj"))
	assert.InDelta(t, 0.6, getGaugeValue(c.successRate, "default", "test-cj"), 0.01)
	assert.Equal(t, float64(3), getCounterValue(c.executionsTotal, "default", "test-cj", "success"))
	assert.Equal(t, float64(2), getCounterValue(c.executionsTotal, "default", "test-cj", "failed"))
}

func TestUpdateAll_CounterIdempotency(t *testing.T) {
	c := NewCollector()

	// First update: 3 successes
	c.UpdateAll(&models.CronJobStatus{
		Namespace:    "default",
		Name:         "test-cj",
		Schedule:     "*/5 * * * *",
		SuccessCount: 3,
		FailureCount: 1,
	})

	assert.Equal(t, float64(3), getCounterValue(c.executionsTotal, "default", "test-cj", "success"))
	assert.Equal(t, float64(1), getCounterValue(c.executionsTotal, "default", "test-cj", "failed"))

	// Second update with same counts — counters should NOT increase
	c.UpdateAll(&models.CronJobStatus{
		Namespace:    "default",
		Name:         "test-cj",
		Schedule:     "*/5 * * * *",
		SuccessCount: 3,
		FailureCount: 1,
	})

	assert.Equal(t, float64(3), getCounterValue(c.executionsTotal, "default", "test-cj", "success"))
	assert.Equal(t, float64(1), getCounterValue(c.executionsTotal, "default", "test-cj", "failed"))

	// Third update with new counts — only delta applied
	c.UpdateAll(&models.CronJobStatus{
		Namespace:    "default",
		Name:         "test-cj",
		Schedule:     "*/5 * * * *",
		SuccessCount: 5,
		FailureCount: 2,
	})

	assert.Equal(t, float64(5), getCounterValue(c.executionsTotal, "default", "test-cj", "success"))
	assert.Equal(t, float64(2), getCounterValue(c.executionsTotal, "default", "test-cj", "failed"))
}

func TestSetInfo(t *testing.T) {
	c := NewCollector()

	c.SetInfo("default", "test-cj", "*/5 * * * *", false)

	val := getGaugeValue(c.info, "default", "test-cj", "*/5 * * * *", "false")
	assert.Equal(t, float64(1), val)
}

func TestRemoveCronJob(t *testing.T) {
	c := NewCollector()

	c.UpdateAll(&models.CronJobStatus{
		Namespace:    "default",
		Name:         "test-cj",
		Schedule:     "*/5 * * * *",
		SuccessCount: 1,
	})

	c.RemoveCronJob("default", "test-cj")

	// After removal, metric families should be cleaned
	mfs, _ := c.Registry.Gather()
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "cronjob" {
					assert.NotEqual(t, "test-cj", lp.GetValue(),
						"metric %s still has label for removed CronJob", mf.GetName())
				}
			}
		}
	}
}

func TestSuccessRate_NoExecutions(t *testing.T) {
	c := NewCollector()

	c.UpdateAll(&models.CronJobStatus{
		Namespace:    "default",
		Name:         "test-cj",
		Schedule:     "*/5 * * * *",
		SuccessCount: 0,
		FailureCount: 0,
	})

	assert.Equal(t, float64(1), getGaugeValue(c.successRate, "default", "test-cj"))
}

func TestRecordMissedSchedule(t *testing.T) {
	c := NewCollector()

	c.RecordMissedSchedule("default", "test-cj")
	c.RecordMissedSchedule("default", "test-cj")

	assert.Equal(t, float64(2), getCounterValue(c.missedSchedules, "default", "test-cj"))
}

func TestUpdateNextSchedule(t *testing.T) {
	c := NewCollector()

	c.UpdateNextSchedule("default", "test-cj", 1700000000)

	assert.Equal(t, float64(1700000000), getGaugeValue(c.nextScheduleTime, "default", "test-cj"))
}
