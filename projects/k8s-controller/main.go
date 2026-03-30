package main

import (
	"flag"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greetingv1 "tour_of_go/projects/k8s-controller/api/v1"
	"tour_of_go/projects/k8s-controller/internal/controller"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(greetingv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8083", "The address the probe endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("main")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics: metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		logger.Error(err, "unable to create manager")
		os.Exit(1)
	}

	if err := (&controller.GreetingReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller")
		os.Exit(1)
	}

	// Health check endpoints — used by K8s liveness/readiness probes
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
}
