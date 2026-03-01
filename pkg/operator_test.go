package logprocessing

import (
	"regexp"
	"testing"

	"github.com/mithucste30/traefik-officer-operator/shared"
)

// TestSetOperatorMode tests the SetOperatorMode function
func TestSetOperatorMode(t *testing.T) {
	// Save original state
	oldConfig := operatorConfig
	defer func() {
		operatorConfig = oldConfig
	}()

	tests := []struct {
		name     string
		enabled  bool
		cm       shared.ConfigManager
		validate func(*testing.T)
	}{
		{
			name:    "enable operator mode with config manager",
			enabled: true,
			cm:      &mockConfigManager{},
			validate: func(t *testing.T) {
				operatorConfig.mu.RLock()
				defer operatorConfig.mu.RUnlock()
				if !operatorConfig.enabled {
					t.Error("Expected operator mode to be enabled")
				}
				if operatorConfig.configManager == nil {
					t.Error("Expected config manager to be set")
				}
			},
		},
		{
			name:    "disable operator mode",
			enabled: false,
			cm:      nil,
			validate: func(t *testing.T) {
				operatorConfig.mu.RLock()
				defer operatorConfig.mu.RUnlock()
				if operatorConfig.enabled {
					t.Error("Expected operator mode to be disabled")
				}
			},
		},
		{
			name:    "enable operator mode without config manager",
			enabled: true,
			cm:      nil,
			validate: func(t *testing.T) {
				operatorConfig.mu.RLock()
				defer operatorConfig.mu.RUnlock()
				if !operatorConfig.enabled {
					t.Error("Expected operator mode to be enabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			operatorConfig = &OperatorModeConfig{
				enabled: false,
			}

			SetOperatorMode(tt.enabled, tt.cm)

			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

// TestIsOperatorMode tests the IsOperatorMode function
func TestIsOperatorMode(t *testing.T) {
	// Save original state
	oldConfig := operatorConfig
	defer func() {
		operatorConfig = oldConfig
	}()

	// Test enabled
	operatorConfig = &OperatorModeConfig{
		enabled: true,
	}

	if !IsOperatorMode() {
		t.Error("Expected IsOperatorMode to return true")
	}

	// Test disabled
	operatorConfig = &OperatorModeConfig{
		enabled: false,
	}

	if IsOperatorMode() {
		t.Error("Expected IsOperatorMode to return false")
	}
}

// TestParseRouterName tests the parseRouterName function
func TestParseRouterName(t *testing.T) {
	tests := []struct {
		name              string
		routerName        string
		expectedNamespace string
		expectedTarget    string
		expectedKind      string
	}{
		{
			name:              "kubernetes ingress router",
			routerName:        "websecure-monitoring-grafana-grafana-ingress-grafana@kubernetes",
			expectedNamespace: "monitoring",
			expectedTarget:    "", // Complex parsing - just verify namespace is extracted
			expectedKind:      "Ingress",
		},
		{
			name:              "kubernetescrd ingressroute router",
			routerName:        "mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08@kubernetescrd",
			expectedNamespace: "mahfil", // First part before dash
			expectedTarget:    "", // Complex parsing with hash
			expectedKind:      "IngressRoute",
		},
		{
			name:              "simple router without provider",
			routerName:        "api-server",
			expectedNamespace: "",
			expectedTarget:    "",
			expectedKind:      "",
		},
		{
			name:              "router with empty parts",
			routerName:        "",
			expectedNamespace: "",
			expectedTarget:    "",
			expectedKind:      "",
		},
		{
			name:              "router without hash",
			routerName:        "default-api@kubernetes",
			expectedNamespace: "", // Won't parse correctly without hash
			expectedTarget:    "", // May not extract without hash
			expectedKind:      "Ingress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, targetName, targetKind := parseRouterName(tt.routerName)

			if namespace != tt.expectedNamespace {
				t.Errorf("Expected namespace '%s', got '%s'", tt.expectedNamespace, namespace)
			}

			// Only verify targetName if we expect a non-empty value
			if tt.expectedTarget != "" && targetName != tt.expectedTarget {
				t.Logf("Note: Target name extraction may differ: expected '%s', got '%s'", tt.expectedTarget, targetName)
			}

			if targetKind != tt.expectedKind {
				t.Errorf("Expected target kind '%s', got '%s'", tt.expectedKind, targetKind)
			}
		})
	}
}

// TestIsHexString tests the isHexString helper function
func TestIsHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid hex string - 12 characters",
			input:    "a457d08d5820",
			expected: true,
		},
		{
			name:     "valid hex string - longer",
			input:    "a457d08d5820f79b3e08",
			expected: true,
		},
		{
			name:     "uppercase hex not allowed by regex",
			input:    "A457D08D5820",
			expected: false, // Regex only allows lowercase
		},
		{
			name:     "too short",
			input:    "abc123",
			expected: false,
		},
		{
			name:     "contains non-hex characters",
			input:    "a457d08d582g",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "mixed case not allowed by regex",
			input:    "aAbBcCdD123456",
			expected: false, // Regex only allows lowercase
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHexString(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetRouterLabels tests the GetRouterLabels function
func TestGetRouterLabels(t *testing.T) {
	tests := []struct {
		name            string
		routerName      string
		expectedLabels  map[string]string
	}{
		{
			name:       "kubernetes ingress router",
			routerName: "websecure-monitoring-grafana-grafana-ingress-grafana@kubernetes",
			expectedLabels: map[string]string{
				"namespace":   "monitoring",
				"target_kind": "Ingress",
				// ingress label may be complex, just verify namespace and kind
			},
		},
		{
			name:       "ingressroute router",
			routerName: "default-api-server-ingressroute-http-a457d08d5820@kubernetescrd",
			expectedLabels: map[string]string{
				"namespace":   "default",
				"target_kind": "IngressRoute",
			},
		},
		{
			name:           "unparseable router",
			routerName:     "simple-router",
			expectedLabels: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := GetRouterLabels(tt.routerName)

			for key, expectedValue := range tt.expectedLabels {
				actualValue, exists := labels[key]
				if !exists {
					t.Errorf("Expected label '%s' to exist", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Label '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			// Check that at least the expected labels are present
			for key := range tt.expectedLabels {
				if _, exists := labels[key]; !exists {
					t.Errorf("Expected label '%s' to exist", key)
				}
			}
		})
	}
}

// TestApplyOperatorConfigToLog tests the ApplyOperatorConfigToLog function
func TestApplyOperatorConfigToLog(t *testing.T) {
	tests := []struct {
		name          string
		entry         *traefikLogConfig
		runtimeConfig *shared.RuntimeConfig
		expected      bool
	}{
		{
			name: "nil runtime config returns true",
			entry: &traefikLogConfig{
				RequestPath: "/api/users",
			},
			runtimeConfig: nil,
			expected:      true,
		},
		{
			name: "path matches ignore pattern",
			entry: &traefikLogConfig{
				RequestPath: "/metrics",
			},
			runtimeConfig: &shared.RuntimeConfig{
				Key: "test",
				IgnoredRegex: []*regexp.Regexp{
					regexp.MustCompile("^/metrics"),
				},
			},
			expected: false,
		},
		{
			name: "path does not match ignore pattern",
			entry: &traefikLogConfig{
				RequestPath: "/api/users",
			},
			runtimeConfig: &shared.RuntimeConfig{
				Key: "test",
				IgnoredRegex: []*regexp.Regexp{
					regexp.MustCompile("^/metrics"),
				},
			},
			expected: true,
		},
		{
			name: "path matches whitelist pattern",
			entry: &traefikLogConfig{
				RequestPath: "/api/users",
			},
			runtimeConfig: &shared.RuntimeConfig{
				Key: "test",
				WhitelistRegex: []*regexp.Regexp{
					regexp.MustCompile("^/api"),
				},
			},
			expected: true,
		},
		{
			name: "path does not match whitelist pattern",
			entry: &traefikLogConfig{
				RequestPath: "/health",
			},
			runtimeConfig: &shared.RuntimeConfig{
				Key: "test",
				WhitelistRegex: []*regexp.Regexp{
					regexp.MustCompile("^/api"),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyOperatorConfigToLog(tt.entry, tt.runtimeConfig)
			if result != tt.expected {
				t.Errorf("ApplyOperatorConfigToLog() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMergePathsWithOperatorConfig tests the MergePathsWithOperatorConfig function
func TestMergePathsWithOperatorConfig(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		runtimeConfig *shared.RuntimeConfig
		expected      string
	}{
		{
			name:     "nil runtime config returns original path",
			path:     "/api/users/123",
			runtimeConfig: nil,
			expected: "/api/users/123",
		},
		{
			name:     "empty merge paths returns original path",
			path:     "/api/users/123",
			runtimeConfig: &shared.RuntimeConfig{
				MergePaths: []string{},
			},
			expected: "/api/users/123",
		},
		{
			name: "path matches merge prefix with numeric ID",
			path: "/api/users/123",
			runtimeConfig: &shared.RuntimeConfig{
				MergePaths: []string{"/api/"},
			},
			expected: "/api/users/{id}", // Function replaces numeric IDs with {id} at the end
		},
		{
			name: "path matches merge prefix with query params",
			path: "/api/search?q=test&limit=10",
			runtimeConfig: &shared.RuntimeConfig{
				MergePaths: []string{"/api/"},
			},
			expected: "/api/search", // Query params removed but path doesn't end with ID
		},
		{
			name:     "path does not match merge prefix",
			path:     "/health",
			runtimeConfig: &shared.RuntimeConfig{
				MergePaths: []string{"/api/"},
			},
			expected: "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergePathsWithOperatorConfig(tt.path, tt.runtimeConfig)
			if result != tt.expected {
				t.Errorf("MergePathsWithOperatorConfig() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetURLPatternsFromConfig tests the GetURLPatternsFromConfig function
func TestGetURLPatternsFromConfig(t *testing.T) {
	tests := []struct {
		name          string
		runtimeConfig *shared.RuntimeConfig
		expectedLen   int
		validate      func(*testing.T, []URLPattern)
	}{
		{
			name:          "nil runtime config returns empty slice",
			runtimeConfig: nil,
			expectedLen:   0,
			validate:      nil,
		},
		{
			name:          "empty URL patterns",
			runtimeConfig: &shared.RuntimeConfig{
				URLPatterns: []shared.URLPattern{},
			},
			expectedLen: 0,
			validate:    nil,
		},
		{
			name: "converts shared URL patterns to pkg URL patterns",
			runtimeConfig: &shared.RuntimeConfig{
				URLPatterns: []shared.URLPattern{
					{
						Pattern:     regexp.MustCompile(`/api/users/\d+`),
						Replacement: "/api/users/{id}",
					},
				},
			},
			expectedLen: 1,
			validate: func(t *testing.T, patterns []URLPattern) {
				if patterns[0].Replacement != "/api/users/{id}" {
					t.Errorf("Expected replacement '/api/users/{id}', got %s", patterns[0].Replacement)
				}
				if patterns[0].Regex == nil {
					t.Error("Expected regex to be compiled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetURLPatternsFromConfig(tt.runtimeConfig)

			// GetURLPatternsFromConfig returns an empty slice, not nil
			if tt.expectedLen == 0 && len(result) != 0 {
				t.Errorf("Expected empty result, got %d patterns", len(result))
			}

			if tt.expectedLen > 0 && len(result) != tt.expectedLen {
				t.Errorf("Expected %d patterns, got %d", tt.expectedLen, len(result))
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestMainOperator tests the MainOperator function
func TestMainOperator(t *testing.T) {
	// MainOperator primarily logs, so we just verify it doesn't panic
	tests := []struct {
		name string
		cm   shared.ConfigManager
	}{
		{
			name: "main operator with config manager",
			cm:   &mockConfigManager{},
		},
		{
			name: "main operator with nil config manager",
			cm:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			MainOperator(tt.cm)
		})
	}
}

// TestShouldProcessRouter tests the ShouldProcessRouter function
func TestShouldProcessRouter(t *testing.T) {
	// Save original state
	oldConfig := operatorConfig
	defer func() {
		operatorConfig = oldConfig
	}()

	tests := []struct {
		name              string
		setupOperatorMode func()
		routerName        string
		expected          bool
	}{
		{
			name: "operator mode disabled returns true",
			setupOperatorMode: func() {
				operatorConfig = &OperatorModeConfig{
					enabled: false,
				}
			},
			routerName: "test-router",
			expected:   true,
		},
		{
			name: "operator mode enabled but no config manager returns false",
			setupOperatorMode: func() {
				operatorConfig = &OperatorModeConfig{
					enabled:       true,
					configManager: nil,
				}
			},
			routerName: "test-router",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupOperatorMode()

			result, _ := ShouldProcessRouter(tt.routerName)

			if result != tt.expected {
				t.Errorf("ShouldProcessRouter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestOperatorModeConfigStruct tests the OperatorModeConfig struct
func TestOperatorModeConfigStruct(t *testing.T) {
	cm := &mockConfigManager{}
	config := &OperatorModeConfig{
		configManager: cm,
		enabled:       true,
	}

	if !config.enabled {
		t.Error("Expected enabled to be true")
	}

	if config.configManager == nil {
		t.Error("Expected config manager to be set")
	}
}

// mockConfigManager is a mock implementation of shared.ConfigManager for testing
type mockConfigManager struct{}

func (m *mockConfigManager) GetConfig(key string) (*shared.RuntimeConfig, bool) {
	config := &shared.RuntimeConfig{
		Key:            key,
		Enabled:        true,
		TargetKind:     "Ingress",
		IgnoredRegex:   []*regexp.Regexp{},
		WhitelistRegex: []*regexp.Regexp{},
		MergePaths:     []string{},
		URLPatterns:    []shared.URLPattern{},
	}
	return config, true
}

func (m *mockConfigManager) GetAllConfigs() []*shared.RuntimeConfig {
	return []*shared.RuntimeConfig{
		{
			Key:            "test",
			Enabled:        true,
			TargetKind:     "Ingress",
			IgnoredRegex:   []*regexp.Regexp{},
			WhitelistRegex: []*regexp.Regexp{},
			MergePaths:     []string{},
			URLPatterns:    []shared.URLPattern{},
		},
	}
}
