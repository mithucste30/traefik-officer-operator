package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// OperatorModeConfig manages configurations from CRD when running in operator mode
type OperatorModeConfig struct {
	configManager *ConfigManager
	mu            sync.RWMutex
	enabled       bool
}

// ConfigManager interface for getting runtime configurations
type ConfigManager interface {
	GetConfig(key string) (*RuntimeConfig, bool)
	GetAllConfigs() []*RuntimeConfig
}

// RuntimeConfig represents the configuration for a specific UrlPerformance
type RuntimeConfig struct {
	Key            string
	Namespace      string
	TargetName     string
	TargetKind     string
	WhitelistRegex []*regexp.Regexp
	IgnoredRegex   []*regexp.Regexp
	MergePaths     []string
	URLPatterns    []URLPattern
	CollectNTop    int
	Enabled        bool
	lastUpdated    interface{} // time.Time
}

var operatorConfig = &OperatorModeConfig{
	enabled: false,
}

// SetOperatorMode enables operator mode and sets the config manager
func SetOperatorMode(enabled bool, cm ConfigManager) {
	operatorConfig.mu.Lock()
	defer operatorConfig.mu.Unlock()

	operatorConfig.enabled = enabled
	if cm != nil {
		operatorConfig.configManager = cm
	}

	logger.Infof("Operator mode set to: %v", enabled)
}

// IsOperatorMode returns whether operator mode is enabled
func IsOperatorMode() bool {
	operatorConfig.mu.RLock()
	defer operatorConfig.mu.RUnlock()
	return operatorConfig.enabled
}

// ShouldProcessRouter checks if a router should be processed based on CRD configs
func ShouldProcessRouter(routerName string) (bool, *RuntimeConfig) {
	if !IsOperatorMode() {
		// Not in operator mode - use legacy config file approach
		return true, nil
	}

	operatorConfig.mu.RLock()
	cm := operatorConfig.configManager
	operatorConfig.mu.RUnlock()

	if cm == nil {
		logger.Warn("Operator mode enabled but no config manager available")
		return false, nil
	}

	// Parse router name to extract namespace and target name
	namespace, targetName, targetKind := parseRouterName(routerName)
	if namespace == "" || targetName == "" {
		logger.Debugf("Could not parse router name: %s", routerName)
		return false, nil
	}

	// Build config key
	configKey := fmt.Sprintf("%s-%s", namespace, targetName)

	// Get configuration
	config, exists := cm.GetConfig(configKey)
	if !exists {
		logger.Debugf("No configuration found for: %s (router: %s)", configKey, routerName)
		return false, nil
	}

	if !config.Enabled {
		logger.Debugf("Configuration disabled for: %s", configKey)
		return false, nil
	}

	// Verify target kind matches
	if targetKind != config.TargetKind {
		logger.Debugf("Target kind mismatch for %s: got %s, expected %s", configKey, targetKind, config.TargetKind)
		return false, nil
	}

	return true, config
}

// parseRouterName parses the router name from Traefik logs
func parseRouterName(routerName string) (namespace, targetName, targetKind string) {
	// Remove provider suffix
	if idx := strings.Index(routerName, "@"); idx != -1 {
		provider := routerName[idx+1:]
		routerName = routerName[:idx]

		if provider == "kubernetes" {
			targetKind = "Ingress"
		} else if provider == "kubernetescrd" {
			targetKind = "IngressRoute"
		}
	}

	parts := strings.Split(routerName, "-")

	if targetKind == "IngressRoute" {
		// Format: namespace-resourceName-hash
		// Example: mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08
		if len(parts) >= 2 {
			namespace = parts[0]
			// Find where the hash starts (typically a hex string)
			for i := len(parts) - 1; i >= 1; i-- {
				if isHexString(parts[i]) && len(parts[i]) >= 12 {
					targetName = strings.Join(parts[1:i], "-")
					break
				}
			}
		}
	} else if targetKind == "Ingress" {
		// Format: entrypoint-namespace-ingressName-hostname-derived-hash
		// Example: websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co
		if len(parts) >= 3 {
			namespace = parts[1]
			// Find where the hash starts
			for i := len(parts) - 1; i >= 2; i-- {
				if isHexString(parts[i]) && len(parts[i]) >= 12 {
					targetName = strings.Join(parts[2:i], "-")
					break
				}
			}
		}
	}

	return namespace, targetName, targetKind
}

// isHexString checks if a string is a hexadecimal string
func isHexString(s string) bool {
	matched, _ := regexp.MatchString("^[0-9a-f]{12,}$", s)
	return matched
}

// GetRouterLabels extracts labels for metrics from router name
func GetRouterLabels(routerName string) map[string]string {
	labels := make(map[string]string)

	namespace, targetName, targetKind := parseRouterName(routerName)

	if namespace != "" {
		labels["namespace"] = namespace
	}
	if targetName != "" {
		labels["ingress"] = targetName
	}
	if targetKind != "" {
		labels["target_kind"] = targetKind
	}

	return labels
}

// ApplyOperatorConfigToLog applies operator configuration to log processing
func ApplyOperatorConfigToLog(entry *traefikLogConfig, runtimeConfig *RuntimeConfig) bool {
	if runtimeConfig == nil {
		return true
	}

	// Check ignored paths first
	for _, regex := range runtimeConfig.IgnoredRegex {
		if regex != nil && regex.MatchString(entry.RequestPath) {
			logger.Debugf("Path %s matches ignore pattern for %s",
				entry.RequestPath, runtimeConfig.Key)
			return false
		}
	}

	// Check whitelist paths (if specified)
	if len(runtimeConfig.WhitelistRegex) > 0 {
		matched := false
		for _, regex := range runtimeConfig.WhitelistRegex {
			if regex != nil && regex.MatchString(entry.RequestPath) {
				matched = true
				break
			}
		}
		if !matched {
			logger.Debugf("Path %s does not match any whitelist pattern for %s",
				entry.RequestPath, runtimeConfig.Key)
			return false
		}
	}

	return true
}

// MergePathsWithOperatorConfig applies path merging based on operator config
func MergePathsWithOperatorConfig(path string, runtimeConfig *RuntimeConfig) string {
	if runtimeConfig == nil || len(runtimeConfig.MergePaths) == 0 {
		return path
	}

	// Check if path starts with any of the merge prefixes
	for _, prefix := range runtimeConfig.MergePaths {
		if strings.HasPrefix(path, prefix) {
			// Remove query parameters and path parameters
			normalized := path
			re1 := regexp.MustCompile(`/\d+(/|$|\?)`)
			normalized = re1.ReplaceAllString(normalized, "/{id}$1")

			re2 := regexp.MustCompile(`\?.*`)
			normalized = re2.ReplaceAllString(normalized, "")
			return normalized
		}
	}

	return path
}

// GetURLPatternsFromConfig returns URL patterns from runtime config
func GetURLPatternsFromConfig(runtimeConfig *RuntimeConfig) []URLPattern {
	if runtimeConfig == nil {
		return nil
	}

	return runtimeConfig.URLPatterns
}
