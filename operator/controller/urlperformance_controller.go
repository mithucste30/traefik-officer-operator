package controller

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-logr/logr"
	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	traefikofficerv1alpha1 "github.com/mithucste30/traefik-officer-operator/operator/api/v1alpha1"
	"github.com/mithucste30/traefik-officer-operator/shared"
)

// UrlPerformanceReconciler reconciles a UrlPerformance object
type UrlPerformanceReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	ConfigManager *ConfigManager
}

// ConfigManager manages dynamic configuration from CRDs
type ConfigManager struct {
	configs map[string]*shared.RuntimeConfig
	mu      sync.RWMutex
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]*shared.RuntimeConfig),
	}
}

// UpdateConfig updates or removes a configuration
func (cm *ConfigManager) UpdateConfig(config *shared.RuntimeConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !config.Enabled {
		delete(cm.configs, config.Key)
		logger.Infof("Removed config for %s (disabled)", config.Key)
		return
	}

	cm.configs[config.Key] = config
	logger.Infof("Updated config for %s", config.Key)
}

// GetConfig retrieves configuration for a specific key
func (cm *ConfigManager) GetConfig(key string) (*shared.RuntimeConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, ok := cm.configs[key]
	return config, ok
}

// GetAllConfigs returns all active configurations
func (cm *ConfigManager) GetAllConfigs() []*shared.RuntimeConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*shared.RuntimeConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configs = append(configs, config)
	}
	return configs
}

//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=traefikofficer.io,resources=urlperformances/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

// Reconcile is the main reconciliation loop
func (r *UrlPerformanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logr.FromContextOrDiscard(ctx)
	reqLogger.Info("Reconciling UrlPerformance", "namespace", req.Namespace, "name", req.Name)

	// Fetch the UrlPerformance instance
	instance := &traefikofficerv1alpha1.UrlPerformance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("UrlPerformance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
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
	}

	if !targetExists {
		reqLogger.Error(targetErr, "Target resource not found")
		r.updateCondition(ctx, instance, "TargetExists", metav1.ConditionFalse, "NotFound", "Target resource not found")
		instance.Status.Phase = traefikofficerv1alpha1.PhaseError
		return r.updateStatus(ctx, instance)
	}

	r.updateCondition(ctx, instance, "TargetExists", metav1.ConditionTrue, "Found", "Target resource found")

	// Build runtime configuration
	configKey := fmt.Sprintf("%s-%s", targetNamespace, instance.Spec.TargetRef.Name)

	// Compile regex patterns
	whitelistRegex := make([]*regexp.Regexp, 0)
	for _, pattern := range instance.Spec.WhitelistPathsRegex {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			reqLogger.Error(err, "Invalid whitelist regex pattern")
			r.updateCondition(ctx, instance, "ConfigGenerated", metav1.ConditionFalse, "InvalidRegex", "Invalid whitelist regex")
			instance.Status.Phase = traefikofficerv1alpha1.PhaseError
			return r.updateStatus(ctx, instance)
		}
		whitelistRegex = append(whitelistRegex, regex)
	}

	ignoredRegex := make([]*regexp.Regexp, 0)
	for _, pattern := range instance.Spec.IgnoredPathsRegex {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			reqLogger.Error(err, "Invalid ignored regex pattern")
			r.updateCondition(ctx, instance, "ConfigGenerated", metav1.ConditionFalse, "InvalidRegex", "Invalid ignored regex")
			instance.Status.Phase = traefikofficerv1alpha1.PhaseError
			return r.updateStatus(ctx, instance)
		}
		ignoredRegex = append(ignoredRegex, regex)
	}

	// Convert URL patterns
	urlPatterns := make([]shared.URLPattern, 0)
	for _, pattern := range instance.Spec.URLPatterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			reqLogger.Error(err, "Invalid URL pattern regex")
			continue
		}
		urlPatterns = append(urlPatterns, shared.URLPattern{
			Pattern:     regex,
			Replacement: pattern.Replacement,
		})
	}

	// Create runtime config
	runtimeConfig := &shared.RuntimeConfig{
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
		LastUpdated:    time.Now(),
	}

	// Update config manager
	if r.ConfigManager != nil {
		r.ConfigManager.UpdateConfig(runtimeConfig)
	}

	// Update status
	r.updateCondition(ctx, instance, "ConfigGenerated", metav1.ConditionTrue, "Generated", "Configuration generated successfully")
	r.updateCondition(ctx, instance, "Ready", metav1.ConditionTrue, "Ready", "UrlPerformance is active")
	instance.Status.Phase = traefikofficerv1alpha1.PhaseActive
	instance.Status.ObservedGeneration = instance.Generation

	return r.updateStatus(ctx, instance)
}

// handleDisabled handles disabled UrlPerformance resources
func (r *UrlPerformanceReconciler) handleDisabled(ctx context.Context, instance *traefikofficerv1alpha1.UrlPerformance) (ctrl.Result, error) {
	reqLogger := logr.FromContextOrDiscard(ctx)

	// Remove configuration
	configKey := fmt.Sprintf("%s-%s", instance.Spec.TargetRef.Namespace, instance.Spec.TargetRef.Name)
	if r.ConfigManager != nil {
		r.ConfigManager.UpdateConfig(&shared.RuntimeConfig{
			Key:     configKey,
			Enabled: false,
		})
	}

	instance.Status.Phase = traefikofficerv1alpha1.PhaseDisabled
	r.updateCondition(ctx, instance, "Ready", metav1.ConditionFalse, "Disabled", "UrlPerformance is disabled")

	reqLogger.Info("UrlPerformance is disabled")
	return r.updateStatus(ctx, instance)
}

// updateCondition updates a condition in the status
func (r *UrlPerformanceReconciler) updateCondition(ctx context.Context, instance *traefikofficerv1alpha1.UrlPerformance, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	newCondition := traefikofficerv1alpha1.Condition{
		Type:               traefikofficerv1alpha1.ConditionType(condType),
		Status:             string(status),
		LastTransitionTime: &now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	found := false
	for i, cond := range instance.Status.Conditions {
		if string(cond.Type) == condType {
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
		Complete(r)
}
