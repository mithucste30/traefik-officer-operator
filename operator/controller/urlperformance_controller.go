package controller

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	traefikofficerv1alpha1 "github.com/0xvox/traefik-officer/operator/api/v1alpha1"
)

// URLPattern represents a compiled URL pattern (shared with pkg/metrics.go)
type URLPattern struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// UrlPerformanceReconciler reconciles a UrlPerformance object
type UrlPerformanceReconciler struct {
	client.Client
	Log      log.Log
	Scheme   *runtime.Scheme
	// ConfigManager is used to update the runtime configuration
	ConfigManager *ConfigManager
}

// ConfigManager manages dynamic configuration from CRDs
type ConfigManager struct {
	// Map of configKey -> UrlPerformance config
	configs map[string]*RuntimeConfig
	mu      sync.RWMutex
}

// RuntimeConfig represents the configuration for a specific UrlPerformance
type RuntimeConfig struct {
	// Key: namespace-ingressName or namespace-ingressRouteName
	Key               string
	Namespace         string
	TargetName        string
	TargetKind        string
	WhitelistRegex    []*regexp.Regexp
	IgnoredRegex      []*regexp.Regexp
	MergePaths        []string
	URLPatterns       []URLPattern
	CollectNTop       int
	Enabled           bool
	lastUpdated       time.Time
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]*RuntimeConfig),
	}
}

// UpdateConfig updates or removes a configuration
func (cm *ConfigManager) UpdateConfig(config *RuntimeConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !config.Enabled {
		// Remove disabled configs
		delete(cm.configs, config.Key)
		logger.Infof("Removed config for %s (disabled)", config.Key)
		return
	}

	cm.configs[config.Key] = config
	logger.Infof("Updated config for %s (whitelist: %d, ignored: %d, top: %d)",
		config.Key,
		len(config.WhitelistRegex),
		len(config.IgnoredRegex),
		config.CollectNTop)
}

// GetConfig retrieves configuration for a specific key
func (cm *ConfigManager) GetConfig(key string) (*RuntimeConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, ok := cm.configs[key]
	return config, ok
}

// GetAllConfigs returns all active configurations
func (cm *ConfigManager) GetAllConfigs() []*RuntimeConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*RuntimeConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configs = append(configs, config)
	}
	return configs
}

//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=traefik.io,resources=ingressroutes,verbs=get;list;watch

// Reconcile is the main reconciliation loop
func (r *UrlPerformanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling UrlPerformance", "namespace", req.Namespace, "name", req.Name)

	// Fetch the UrlPerformance instance
	instance := &traefikofficerv1alpha1.UrlPerformance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			log.Info("UrlPerformance resource not found. Ignoring since object must be deleted", "name", req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get UrlPerformance")
		return ctrl.Result{}, err
	}

	// Initialize status if needed
	if instance.Status.ObservedGeneration == 0 {
		instance.Status.Phase = traefikofficerv1alpha1.PhasePending
	}

	// Check if disabled
	if !instance.Spec.Enabled {
		return r.handleDisabled(ctx, instance)
	}

	// Verify target exists
	targetNamespace := instance.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = instance.Namespace
	}

	targetExists := false
	var targetErr error

	switch instance.Spec.TargetRef.Kind {
	case "Ingress":
		ingress := &networkingv1.Ingress{}
		targetErr = r.Get(ctx, types.NamespacedName{
			Namespace: targetNamespace,
			Name:      instance.Spec.TargetRef.Name,
		}, ingress)
		targetExists = (targetErr == nil)

	case "IngressRoute":
		ingressRoute := &traefikv1alpha1.IngressRoute{}
		targetErr = r.Get(ctx, types.NamespacedName{
			Namespace: targetNamespace,
			Name:      instance.Spec.TargetRef.Name,
		}, ingressRoute)
		targetExists = (targetErr == nil)
	}

	if !targetExists {
		log.Error(targetErr, "Target resource not found", "kind", instance.Spec.TargetRef.Kind, "namespace", targetNamespace, "name", instance.Spec.TargetRef.Name)
		r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionTargetExists, metav1.ConditionFalse, "NotFound", "Target resource not found")
		instance.Status.Phase = traefikofficerv1alpha1.PhaseError
		return r.updateStatus(ctx, instance)
	}

	// Target exists - proceed with configuration
	r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionTargetExists, metav1.ConditionTrue, "Found", "Target resource found")

	// Build runtime configuration
	configKey := fmt.Sprintf("%s-%s", targetNamespace, instance.Spec.TargetRef.Name)

	// Compile regex patterns
	whitelistRegex := make([]*regexp.Regexp, 0)
	for _, pattern := range instance.Spec.WhitelistPathsRegex {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			log.Error(err, "Invalid whitelist regex pattern", "pattern", pattern)
			r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionConfigGenerated, metav1.ConditionFalse, "InvalidRegex", fmt.Sprintf("Invalid whitelist regex: %s", pattern))
			instance.Status.Phase = traefikofficerv1alpha1.PhaseError
			return r.updateStatus(ctx, instance)
		}
		whitelistRegex = append(whitelistRegex, regex)
	}

	ignoredRegex := make([]*regexp.Regexp, 0)
	for _, pattern := range instance.Spec.IgnoredPathsRegex {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			log.Error(err, "Invalid ignored regex pattern", "pattern", pattern)
			r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionConfigGenerated, metav1.ConditionFalse, "InvalidRegex", fmt.Sprintf("Invalid ignored regex: %s", pattern))
			instance.Status.Phase = traefikofficerv1alpha1.PhaseError
			return r.updateStatus(ctx, instance)
		}
		ignoredRegex = append(ignoredRegex, regex)
	}

	// Convert URL patterns
	urlPatterns := make([]URLPattern, 0)
	for _, pattern := range instance.Spec.URLPatterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			log.Error(err, "Invalid URL pattern regex", "pattern", pattern.Pattern)
			continue
		}
		urlPatterns = append(urlPatterns, URLPattern{
			Pattern:     regex,
			Replacement: pattern.Replacement,
		})
	}

	// Create runtime config
	runtimeConfig := &RuntimeConfig{
		Key:            configKey,
		Namespace:      targetNamespace,
		TargetName:     instance.Spec.TargetRef.Name,
		TargetKind:     instance.Spec.TargetRef.Kind,
		WhitelistRegex: whitelistRegex,
		IgnoredRegex:   ignoredRegex,
		MergePaths:     instance.Spec.MergePathsWithExtensions,
		URLPatterns:    urlPatterns,
		CollectNTop:    instance.Spec.CollectNTop,
		Enabled:        instance.Spec.Enabled,
		lastUpdated:    time.Now(),
	}

	// Update config manager
	if r.ConfigManager != nil {
		r.ConfigManager.UpdateConfig(runtimeConfig)
	}

	// Update status
	r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionConfigGenerated, metav1.ConditionTrue, "Generated", "Configuration generated successfully")
	r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionReady, metav1.ConditionTrue, "Ready", "UrlPerformance is active")
	instance.Status.Phase = traefikofficerv1alpha1.PhaseActive
	instance.Status.ObservedGeneration = instance.Generation

	return r.updateStatus(ctx, instance)
}

// handleDisabled handles disabled UrlPerformance resources
func (r *UrlPerformanceReconciler) handleDisabled(ctx context.Context, instance *traefikofficerv1alpha1.UrlPerformance) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Remove configuration
	configKey := fmt.Sprintf("%s-%s", instance.Spec.TargetRef.Namespace, instance.Spec.TargetRef.Name)
	if r.ConfigManager != nil {
		r.ConfigManager.UpdateConfig(&RuntimeConfig{
			Key:     configKey,
			Enabled: false,
		})
	}

	instance.Status.Phase = traefikofficerv1alpha1.PhaseDisabled
	r.updateCondition(ctx, instance, traefikofficerv1alpha1.ConditionReady, metav1.ConditionFalse, "Disabled", "UrlPerformance is disabled")

	log.Info("UrlPerformance is disabled", "name", instance.Name)
	return r.updateStatus(ctx, instance)
}

// updateCondition updates a condition in the status
func (r *UrlPerformanceReconciler) updateCondition(ctx context.Context, instance *traefikofficerv1alpha1.UrlPerformance, condType traefikofficerv1alpha1.ConditionType, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	newCondition := traefikofficerv1alpha1.Condition{
		Type:               condType,
		Status:             string(status),
		LastTransitionTime: &now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	found := false
	for i, cond := range instance.Status.Conditions {
		if cond.Type == condType {
			if cond.Status != string(status) {
				instance.Status.Conditions[i] = newCondition
			}
			found = true
			break
		}
	}

	if !found {
		instance.Status.Conditions = append(instance.Status.Conditions, newCondition)
	}
}

// updateStatus updates the status subresource
func (r *UrlPerformanceReconciler) updateStatus(ctx context.Context, instance *traefikofficerv1alpha1.UrlPerformance) (ctrl.Result, error) {
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *UrlPerformanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&traefikofficerv1alpha1.UrlPerformance{}).
		// Watch for changes to Ingress resources
		// Watches(&source.Kind{Type: &networkingv1.Ingress{}}, handler.EnqueueRequestsFromMapFunc(r.findObjectsForIngress)).
		// Watch for changes to IngressRoute resources
		// Watches(&source.Kind{Type: &traefikv1alpha1.IngressRoute{}}, handler.EnqueueRequestsFromMapFunc(r.findObjectsForIngressRoute)).
		Complete(r)
}
