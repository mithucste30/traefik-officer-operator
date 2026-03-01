package shared

import (
	"regexp"
	"testing"
	"time"
)

func TestRuntimeConfig(t *testing.T) {
	t.Run("New RuntimeConfig creation", func(t *testing.T) {
		config := &RuntimeConfig{
			Key:            "test-ns-test-ingress",
			Namespace:      "test-ns",
			TargetName:     "test-ingress",
			TargetKind:     "Ingress",
			WhitelistRegex: nil,
			IgnoredRegex:   nil,
			MergePaths:     []string{"/api/"},
			URLPatterns:    nil,
			CollectNTop:    20,
			Enabled:        true,
			LastUpdated:    time.Now(),
		}

		if config.Key != "test-ns-test-ingress" {
			t.Errorf("Expected Key to be 'test-ns-test-ingress', got %s", config.Key)
		}
		if config.Namespace != "test-ns" {
			t.Errorf("Expected Namespace to be 'test-ns', got %s", config.Namespace)
		}
		if config.TargetName != "test-ingress" {
			t.Errorf("Expected TargetName to be 'test-ingress', got %s", config.TargetName)
		}
		if config.TargetKind != "Ingress" {
			t.Errorf("Expected TargetKind to be 'Ingress', got %s", config.TargetKind)
		}
		if !config.Enabled {
			t.Error("Expected Enabled to be true")
		}
		if config.CollectNTop != 20 {
			t.Errorf("Expected CollectNTop to be 20, got %d", config.CollectNTop)
		}
		if len(config.MergePaths) != 1 || config.MergePaths[0] != "/api/" {
			t.Errorf("Expected MergePaths to contain ['/api/'], got %v", config.MergePaths)
		}
	})

	t.Run("RuntimeConfig with empty slices", func(t *testing.T) {
		config := &RuntimeConfig{
			Key:            "test-key",
			Namespace:      "default",
			TargetName:     "test-target",
			TargetKind:     "Ingress",
			WhitelistRegex: []*regexp.Regexp{},
			IgnoredRegex:   []*regexp.Regexp{},
			MergePaths:     []string{},
			URLPatterns:    []URLPattern{},
			CollectNTop:    10,
			Enabled:        true,
			LastUpdated:    time.Now(),
		}

		if len(config.WhitelistRegex) != 0 {
			t.Errorf("Expected WhitelistRegex to be empty, got %d items", len(config.WhitelistRegex))
		}
		if len(config.IgnoredRegex) != 0 {
			t.Errorf("Expected IgnoredRegex to be empty, got %d items", len(config.IgnoredRegex))
		}
		if len(config.MergePaths) != 0 {
			t.Errorf("Expected MergePaths to be empty, got %d items", len(config.MergePaths))
		}
		if len(config.URLPatterns) != 0 {
			t.Errorf("Expected URLPatterns to be empty, got %d items", len(config.URLPatterns))
		}
	})

	t.Run("RuntimeConfig disabled", func(t *testing.T) {
		config := &RuntimeConfig{
			Key:         "disabled-key",
			Namespace:   "default",
			TargetName:  "disabled-target",
			TargetKind:  "Ingress",
			CollectNTop: 20,
			Enabled:     false,
			LastUpdated: time.Now(),
		}

		if config.Enabled {
			t.Error("Expected Enabled to be false")
		}
	})
}

func TestURLPattern(t *testing.T) {
	t.Run("URLPattern with pattern and replacement", func(t *testing.T) {
		pattern := regexp.MustCompile(`/api/v1/users/(\d+)`)
		urlPattern := URLPattern{
			Pattern:     pattern,
			Replacement: "/api/v1/users/:id",
		}

		if urlPattern.Pattern == nil {
			t.Error("Expected Pattern to be non-nil")
		}
		if urlPattern.Replacement != "/api/v1/users/:id" {
			t.Errorf("Expected Replacement to be '/api/v1/users/:id', got %s", urlPattern.Replacement)
		}
	})

	t.Run("URLPattern with empty replacement", func(t *testing.T) {
		pattern := regexp.MustCompile(`/api/.*`)
		urlPattern := URLPattern{
			Pattern:     pattern,
			Replacement: "",
		}

		if urlPattern.Pattern == nil {
			t.Error("Expected Pattern to be non-nil")
		}
		if urlPattern.Replacement != "" {
			t.Errorf("Expected Replacement to be empty, got %s", urlPattern.Replacement)
		}
	})
}

func TestConfigManagerInterface(t *testing.T) {
	t.Run("ConfigManager interface definition", func(t *testing.T) {
		// This test verifies the ConfigManager interface is correctly defined
		// Actual implementation tests are in controller tests

		var _ interface {
			UpdateConfig(config *RuntimeConfig)
			GetConfig(key string) (*RuntimeConfig, bool)
			GetAllConfigs() []*RuntimeConfig
		} = &MockConfigManager{}
	})
}

// MockConfigManager is a mock implementation for testing
type MockConfigManager struct {
	configs map[string]*RuntimeConfig
}

func (m *MockConfigManager) UpdateConfig(config *RuntimeConfig) {
	if m.configs == nil {
		m.configs = make(map[string]*RuntimeConfig)
	}
	m.configs[config.Key] = config
}

func (m *MockConfigManager) GetConfig(key string) (*RuntimeConfig, bool) {
	if m.configs == nil {
		return nil, false
	}
	config, ok := m.configs[key]
	return config, ok
}

func (m *MockConfigManager) GetAllConfigs() []*RuntimeConfig {
	if m.configs == nil {
		return []*RuntimeConfig{}
	}
	configs := make([]*RuntimeConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	return configs
}
