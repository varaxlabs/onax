package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsHandler creates an HTTP handler that serves metrics from the collector's registry.
func NewMetricsHandler(collector *Collector) http.Handler {
	return promhttp.HandlerFor(collector.Registry, promhttp.HandlerOpts{})
}

// HandlerFor creates an HTTP handler from an arbitrary registry.
func HandlerFor(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
