package main

import (
	"net/http"
	"os"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/varaxlabs/varax-monitor/pkg/controller"
	"github.com/varaxlabs/varax-monitor/pkg/metrics"
	"github.com/varaxlabs/varax-monitor/pkg/watcher"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
}

func main() {
	opts := zap.Options{Development: false}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := ctrl.Log.WithName("setup")

	metricsCollector := metrics.NewCollector()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: ":8080",
			ExtraHandlers: map[string]http.Handler{
				"/custom-metrics": metrics.NewMetricsHandler(metricsCollector),
			},
		},
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		logger.Error(err, "unable to create manager")
		os.Exit(1)
	}

	// Register our custom metrics with controller-runtime's default Prometheus registry
	metricsCollector.RegisterWithPrometheus()

	// Setup CronJob reconciler
	reconciler := &controller.CronJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Collector: metricsCollector,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}

	// Setup schedule tracker as a runnable
	if err := watcher.SetupScheduleTracker(mgr, metricsCollector); err != nil {
		logger.Error(err, "unable to set up schedule tracker")
		os.Exit(1)
	}

	// Health and ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	logger.Info("Starting varax-monitor")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
}
