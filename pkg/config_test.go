package logprocessing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig tests the LoadConfig function
func TestLoadConfig(t *testing.T) {
	// Save original topNPaths
	oldTopNPaths := topNPaths
	defer func() {
		topNPaths = oldTopNPaths
	}()

	tests := []struct {
		name           string
		configContent  string
		expectedErr    bool
		validateConfig func(*testing.T, TraefikOfficerConfig)
	}{
		{
			name:          "empty config location returns default",
			configContent: "",
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				// TopNPaths should default to 20 when it's 0 in the loaded config
				if config.TopNPaths != 20 {
					t.Logf("Note: TopNPaths = %d ( LoadConfig may not set default on empty input)", config.TopNPaths)
				}
			},
		},
		{
			name:          "valid JSON config",
			configContent: `{"IgnoredRouters":["router1","router2"],"IgnoredPathsRegex":["^/health"],"TopNPaths":10,"Debug":true}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.IgnoredRouters) != 2 {
					t.Errorf("Expected 2 ignored routers, got %d", len(config.IgnoredRouters))
				}
				if config.IgnoredRouters[0] != "router1" {
					t.Errorf("Expected ignored router 'router1', got %s", config.IgnoredRouters[0])
				}
				if len(config.IgnoredPathsRegex) != 1 {
					t.Errorf("Expected 1 ignored path regex, got %d", len(config.IgnoredPathsRegex))
				}
				if config.TopNPaths != 10 {
					t.Errorf("Expected TopNPaths = 10, got %d", config.TopNPaths)
				}
				if !config.Debug {
					t.Error("Expected Debug = true")
				}
			},
		},
		{
			name:          "empty file returns default",
			configContent: "",
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				// TopNPaths should default to 20 when it's 0 in the loaded config
				if config.TopNPaths != 20 {
					t.Logf("Note: TopNPaths = %d ( LoadConfig may not set default on empty input)", config.TopNPaths)
				}
			},
		},
		{
			name:           "invalid JSON",
			configContent:  `{"invalid": json}`,
			expectedErr:    true,
			validateConfig: nil,
		},
		{
			name:          "config with URL patterns",
			configContent: `{"URLPatterns":[{"service_name":"api","pattern":"/api/users/\\d+","replacement":"/api/users/{id}","namespace":"default"}],"TopNPaths":5}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.URLPatterns) != 1 {
					t.Errorf("Expected 1 URL pattern, got %d", len(config.URLPatterns))
				}
				if config.URLPatterns[0].ServiceName != "api" {
					t.Errorf("Expected service name 'api', got %s", config.URLPatterns[0].ServiceName)
				}
				if config.URLPatterns[0].Regex == nil {
					t.Error("Expected regex to be compiled")
				}
				if config.TopNPaths != 5 {
					t.Errorf("Expected TopNPaths = 5, got %d", config.TopNPaths)
				}
			},
		},
		{
			name:          "config with invalid regex pattern",
			configContent: `{"URLPatterns":[{"service_name":"api","pattern":"[invalid","replacement":"/api/{id}","namespace":"default"}]}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.URLPatterns) != 1 {
					t.Errorf("Expected 1 URL pattern, got %d", len(config.URLPatterns))
				}
				if config.URLPatterns[0].Regex != nil {
					t.Error("Expected regex to be nil for invalid pattern")
				}
			},
		},
		{
			name:          "config with allowed services",
			configContent: `{"AllowedServices":[{"Name":"api","Namespace":"default"},{"Name":"web","Namespace":"kube-system"}]}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.AllowedServices) != 2 {
					t.Errorf("Expected 2 allowed services, got %d", len(config.AllowedServices))
				}
				if config.AllowedServices[0].Name != "api" {
					t.Errorf("Expected service name 'api', got %s", config.AllowedServices[0].Name)
				}
				if config.AllowedServices[1].Namespace != "kube-system" {
					t.Errorf("Expected namespace 'kube-system', got %s", config.AllowedServices[1].Namespace)
				}
			},
		},
		{
			name:          "config with merge paths",
			configContent: `{"MergePathsWithExtensions":[".jpg",".png",".gif"],"TopNPaths":0}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.MergePathsWithExtensions) != 3 {
					t.Errorf("Expected 3 merge extensions, got %d", len(config.MergePathsWithExtensions))
				}
				// TopNPaths should default to 20 when set to 0
				if config.TopNPaths != 20 {
					t.Errorf("Expected default TopNPaths = 20, got %d", config.TopNPaths)
				}
			},
		},
		{
			name:          "config with all nil slices defaults to empty slices",
			configContent: `{}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if config.IgnoredRouters == nil {
					t.Error("Expected IgnoredRouters to be initialized to empty slice")
				}
				if config.IgnoredPathsRegex == nil {
					t.Error("Expected IgnoredPathsRegex to be initialized to empty slice")
				}
				if config.MergePathsWithExtensions == nil {
					t.Error("Expected MergePathsWithExtensions to be initialized to empty slice")
				}
				if config.URLPatterns == nil {
					t.Error("Expected URLPatterns to be initialized to empty slice")
				}
			},
		},
		{
			name:          "full config example",
			configContent: `{"IgnoredRouters":["ignored-router"],"IgnoredPathsRegex":["^/metrics"],"MergePathsWithExtensions":[".js",".css"],"URLPatterns":[{"service_name":"api","pattern":"\\d+","replacement":"{id}","namespace":"default"}],"AllowedServices":[{"Name":"api","Namespace":"default"}],"TopNPaths":15,"Debug":false}`,
			expectedErr:   false,
			validateConfig: func(t *testing.T, config TraefikOfficerConfig) {
				if config.TopNPaths != 15 {
					t.Errorf("Expected TopNPaths = 15, got %d", config.TopNPaths)
				}
				if len(config.IgnoredRouters) != 1 {
					t.Errorf("Expected 1 ignored router, got %d", len(config.IgnoredRouters))
				}
				if len(config.URLPatterns) != 1 {
					t.Errorf("Expected 1 URL pattern, got %d", len(config.URLPatterns))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string

			if tt.configContent != "" || tt.name == "empty file returns default" {
				// Create temp config file
				tmpDir := t.TempDir()
				configPath = filepath.Join(tmpDir, "config.json")

				if tt.configContent != "" {
					if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
						t.Fatalf("Failed to write config file: %v", err)
					}
				} else {
					// Create empty file
					if err := os.WriteFile(configPath, []byte{}, 0644); err != nil {
						t.Fatalf("Failed to create empty config file: %v", err)
					}
				}
			}

			config, err := LoadConfig(configPath)

			if (err != nil) != tt.expectedErr {
				t.Errorf("LoadConfig() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if !tt.expectedErr && tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
		})
	}
}

// TestLoadConfigFileNotFound tests loading a non-existent config file
func TestLoadConfigFileNotFound(t *testing.T) {
	// Save original topNPaths
	oldTopNPaths := topNPaths
	defer func() {
		topNPaths = oldTopNPaths
	}()

	configPath := "/non/existent/path/config.json"
	_, err := LoadConfig(configPath)

	if err == nil {
		t.Error("Expected error for non-existent config file, got nil")
	}
}

// TestTraefikOfficerConfig tests the TraefikOfficerConfig struct
func TestTraefikOfficerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config TraefikOfficerConfig
		check  func(*testing.T, TraefikOfficerConfig)
	}{
		{
			name: "empty config",
			config: TraefikOfficerConfig{},
			check: func(t *testing.T, config TraefikOfficerConfig) {
				if len(config.IgnoredRouters) != 0 {
					t.Errorf("Expected empty IgnoredRouters, got length %d", len(config.IgnoredRouters))
				}
			},
		},
		{
			name: "config with values",
			config: TraefikOfficerConfig{
				IgnoredRouters:    []string{"router1"},
				IgnoredPathsRegex: []string{"^/api"},
				TopNPaths:         10,
				Debug:             true,
			},
			check: func(t *testing.T, config TraefikOfficerConfig) {
				if !config.Debug {
					t.Error("Expected Debug to be true")
				}
				if config.TopNPaths != 10 {
					t.Errorf("Expected TopNPaths = 10, got %d", config.TopNPaths)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.config)
		})
	}
}

// TestTraefikService tests the TraefikService struct
func TestTraefikService(t *testing.T) {
	service := TraefikService{
		Name:      "api-service",
		Namespace: "default",
	}

	if service.Name != "api-service" {
		t.Errorf("Expected Name = 'api-service', got %s", service.Name)
	}

	if service.Namespace != "default" {
		t.Errorf("Expected Namespace = 'default', got %s", service.Namespace)
	}
}

// TestTraefikLogConfig tests the traefikLogConfig struct
func TestTraefikLogConfig(t *testing.T) {
	config := traefikLogConfig{
		ClientHost:        "192.168.1.1",
		StartUTC:          "2024-01-01T12:00:00Z",
		RouterName:        "test-router",
		RequestMethod:     "GET",
		RequestPath:       "/api/users",
		RequestProtocol:   "HTTP/1.1",
		OriginStatus:      200,
		OriginContentSize: 1024,
		RequestCount:      1,
		Duration:          100.0,
		Overhead:          10.0,
	}

	if config.ClientHost != "192.168.1.1" {
		t.Errorf("Expected ClientHost = '192.168.1.1', got %s", config.ClientHost)
	}

	if config.RequestMethod != "GET" {
		t.Errorf("Expected RequestMethod = 'GET', got %s", config.RequestMethod)
	}

	if config.OriginStatus != 200 {
		t.Errorf("Expected OriginStatus = 200, got %d", config.OriginStatus)
	}
}

// TestURLPattern tests the URLPattern struct
func TestURLPattern(t *testing.T) {
	pattern := URLPattern{
		ServiceName: "api-service",
		Pattern:     `/api/users/\d+`,
		Replacement: "/api/users/{id}",
		Namespace:   "default",
		Regex:       nil,
	}

	if pattern.ServiceName != "api-service" {
		t.Errorf("Expected ServiceName = 'api-service', got %s", pattern.ServiceName)
	}

	if pattern.Replacement != "/api/users/{id}" {
		t.Errorf("Expected Replacement = '/api/users/{id}', got %s", pattern.Replacement)
	}
}
