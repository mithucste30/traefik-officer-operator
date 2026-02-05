package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	logger "github.com/sirupsen/logrus"

	traefikofficerv1alpha1 "github.com/mithucste30/traefik-officer-operator/operator/api/v1alpha1"
	"github.com/mithucste30/traefik-officer-operator/operator/controller"

	// Import the pkg functions for log processing
	logprocessing "github.com/mithucste30/traefik-officer-operator/pkg"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(traefikofficerv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	// Log processor flags
	var logFile string
	var jsonLogs bool
	var useK8s bool
	var k8sNamespace string
	var k8sContainer string
	var k8sLabelSelector string
	var enableLogProcessor bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// Log processor flags
	flag.StringVar(&logFile, "log-file", "", "Path to Traefik access log file (for file mode)")
	flag.BoolVar(&jsonLogs, "json-logs", false, "Parse logs as JSON instead of common log format")
	flag.BoolVar(&useK8s, "use-k8s", false, "Read logs from Kubernetes pods instead of file")
	flag.StringVar(&k8sNamespace, "k8s-namespace", "traefik", "Kubernetes namespace for Traefik pods")
	flag.StringVar(&k8sContainer, "k8s-container", "traefik", "Container name in Traefik pods")
	flag.StringVar(&k8sLabelSelector, "k8s-label-selector", "app.kubernetes.io/name=traefik", "Label selector for Traefik pods")
	flag.BoolVar(&enableLogProcessor, "enable-log-processor", false, "Enable embedded log processor")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Enable logrus for pkg functions
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "traefik-officer-operator-lock",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create config manager for dynamic configuration
	configManager := controller.NewConfigManager()

	// Enable operator mode in pkg and set config manager
	if enableLogProcessor {
		logprocessing.SetOperatorMode(true, configManager)
		logger.Info("Operator mode enabled in log processor")
	}

	// Setup UrlPerformance controller
	if err = (&controller.UrlPerformanceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("UrlPerformance"),
		Scheme:        mgr.GetScheme(),
		ConfigManager: configManager,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "UrlPerformance")
		os.Exit(1)
	}

	// Add health check endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start log processor if enabled
	if enableLogProcessor {
		go startLogProcessor(logFile, jsonLogs, useK8s, k8sNamespace, k8sContainer, k8sLabelSelector)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// startLogProcessor starts the embedded log processor
func startLogProcessor(logFile string, jsonLogs bool, useK8s bool, k8sNamespace, k8sContainer, k8sLabelSelector string) {
	logger.Info("Starting embedded log processor")

	// Import the log processing setup from pkg
	// This will use the operator mode we enabled earlier

	// TODO: Implement proper log source creation and processing
	// For now, this is a placeholder to show where it would go

	logger.Info("Log processor placeholder - implementation pending")
	logger.Infof("Config: logFile=%s, jsonLogs=%v, useK8s=%v, namespace=%s, container=%s, selector=%s",
		logFile, jsonLogs, useK8s, k8sNamespace, k8sContainer, k8sLabelSelector)

	// The actual implementation would:
	// 1. Create log source (file or k8s)
	// 2. Start processing logs
	// 3. Use IsOperatorMode() to check CRD configs
	// 4. Filter and process based on UrlPerformance CRDs
}
