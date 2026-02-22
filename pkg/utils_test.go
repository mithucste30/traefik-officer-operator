package logprocessing

import (
	"regexp"
	"testing"
)

// TestCheckWhiteListStrict tests exact string matching in whitelist
func TestCheckWhiteListStrict(t *testing.T) {
	tests := []struct {
		name        string
		str         string
		matchStrings []string
		expected    bool
	}{
		{
			name:        "exact match found",
			str:         "/api/users",
			matchStrings: []string{"/api/users", "/api/orders"},
			expected:    true,
		},
		{
			name:        "exact match not found",
			str:         "/api/products",
			matchStrings: []string{"/api/users", "/api/orders"},
			expected:    false,
		},
		{
			name:        "substring should not match",
			str:         "/api/users/123",
			matchStrings: []string{"/api/users"},
			expected:    false,
		},
		{
			name:        "empty match list",
			str:         "/api/users",
			matchStrings: []string{},
			expected:    false,
		},
		{
			name:        "nil match list",
			str:         "/api/users",
			matchStrings: nil,
			expected:    false,
		},
		{
			name:        "empty string",
			str:         "",
			matchStrings: []string{""},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkWhiteListStrict(tt.str, tt.matchStrings)
			if result != tt.expected {
				t.Errorf("checkWhiteListStrict() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCheckWhiteList tests substring matching in whitelist
func TestCheckWhiteList(t *testing.T) {
	tests := []struct {
		name        string
		str         string
		matchStrings []string
		expected    bool
	}{
		{
			name:        "substring match found",
			str:         "/api/users/123",
			matchStrings: []string{"/api/users", "/api/orders"},
			expected:    true,
		},
		{
			name:        "exact match",
			str:         "/api/users",
			matchStrings: []string{"/api/users", "/api/orders"},
			expected:    true,
		},
		{
			name:        "no match",
			str:         "/api/products",
			matchStrings: []string{"/users", "/orders"},
			expected:    false,
		},
		{
			name:        "empty match list",
			str:         "/api/users",
			matchStrings: []string{},
			expected:    false,
		},
		{
			name:        "nil match list",
			str:         "/api/users",
			matchStrings: nil,
			expected:    false,
		},
		{
			name:        "empty string matches empty in list",
			str:         "",
			matchStrings: []string{""},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkWhiteList(tt.str, tt.matchStrings)
			if result != tt.expected {
				t.Errorf("checkWhiteList() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMergePaths tests path merging based on prefix matching
func TestMergePaths(t *testing.T) {
	tests := []struct {
		name         string
		str          string
		matchStrings []string
		expected     string
	}{
		{
			name:         "matches first prefix",
			str:          "/api/users/123",
			matchStrings: []string{"/api/", "/static/"},
			expected:     "/api/",
		},
		{
			name:         "matches second prefix",
			str:          "/static/css/style.css",
			matchStrings: []string{"/api/", "/static/"},
			expected:     "/static/",
		},
		{
			name:         "no match returns original",
			str:          "/health",
			matchStrings: []string{"/api/", "/static/"},
			expected:     "/health",
		},
		{
			name:         "empty match list returns original",
			str:          "/api/users",
			matchStrings: []string{},
			expected:     "/api/users",
		},
		{
			name:         "nil match list returns original",
			str:          "/api/users",
			matchStrings: nil,
			expected:     "/api/users",
		},
		{
			name:         "longest prefix first",
			str:          "/api/v2/users",
			matchStrings: []string{"/api/v2/", "/api/"},
			expected:     "/api/v2/",
		},
		{
			name:         "empty string",
			str:          "",
			matchStrings: []string{"/api/"},
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergePaths(tt.str, tt.matchStrings)
			if result != tt.expected {
				t.Errorf("mergePaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCheckMatches tests regex pattern matching
func TestCheckMatches(t *testing.T) {
	tests := []struct {
		name             string
		str              string
		matchExpressions []string
		expected         bool
	}{
		{
			name:             "matches regex pattern",
			str:              "/api/users",
			matchExpressions: []string{"^/api/", "^/static/"},
			expected:         true,
		},
		{
			name:             "matches second pattern",
			str:              "/static/css",
			matchExpressions: []string{"^/api/", "^/static/"},
			expected:         true,
		},
		{
			name:             "no pattern matches",
			str:              "/health",
			matchExpressions: []string{"^/api/", "^/static/"},
			expected:         false,
		},
		{
			name:             "invalid regex is skipped",
			str:              "/api/users",
			matchExpressions: []string{"[invalid", "^/api/"},
			expected:         true, // Second pattern should match
		},
		{
			name:             "empty pattern list",
			str:              "/api/users",
			matchExpressions: []string{},
			expected:         false,
		},
		{
			name:             "nil pattern list",
			str:              "/api/users",
			matchExpressions: nil,
			expected:         false,
		},
		{
			name:             "complex regex pattern",
			str:              "/api/users/123",
			matchExpressions: []string{"^/api/\\w+/\\d+$"},
			expected:         true,
		},
		{
			name:             "UUID pattern match",
			str:              "/api/users/550e8400-e29b-41d4-a716-446655440000",
			matchExpressions: []string{"/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"},
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkMatches(tt.str, tt.matchExpressions)
			if result != tt.expected {
				t.Errorf("checkMatches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseJSON tests JSON log line parsing
func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
		check   func(*testing.T, traefikLogConfig)
	}{
		{
			name: "valid JSON log line",
			line: `{"ClientHost":"192.168.1.1","RouterName":"test-router","RequestMethod":"GET","RequestPath":"/api/users","OriginStatus":200,"Duration":45000000}`,
			wantErr: false,
			check: func(t *testing.T, log traefikLogConfig) {
				if log.ClientHost != "192.168.1.1" {
					t.Errorf("ClientHost = %v, want 192.168.1.1", log.ClientHost)
				}
				if log.RouterName != "test-router" {
					t.Errorf("RouterName = %v, want test-router", log.RouterName)
				}
				if log.RequestMethod != "GET" {
					t.Errorf("RequestMethod = %v, want GET", log.RequestMethod)
				}
				if log.RequestPath != "/api/users" {
					t.Errorf("RequestPath = %v, want /api/users", log.RequestPath)
				}
				if log.OriginStatus != 200 {
					t.Errorf("OriginStatus = %v, want 200", log.OriginStatus)
				}
				if log.Duration != 45.0 { // Converted from nanoseconds to milliseconds
					t.Errorf("Duration = %v, want 45.0", log.Duration)
				}
			},
		},
		{
			name:    "invalid JSON",
			line:    `not valid json`,
			wantErr: true,
		},
		{
			name:    "empty string",
			line:    "",
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			line:    `{"ClientHost":"192.168.1.1"`,
			wantErr: true,
		},
		{
			name: "valid JSON with all fields",
			line: `{"ClientHost":"10.0.0.1","StartUTC":"2024-01-01T12:00:00Z","RouterName":"websecure-api@kubernetes","RequestMethod":"POST","RequestPath":"/api/orders","RequestProtocol":"HTTP/1.1","OriginStatus":201,"OriginContentSize":1024,"RequestCount":1,"Duration":123456789,"Overhead":5000000}`,
			wantErr: false,
			check: func(t *testing.T, log traefikLogConfig) {
				if log.ClientHost != "10.0.0.1" {
					t.Errorf("ClientHost = %v, want 10.0.0.1", log.ClientHost)
				}
				if log.Overhead != 5.0 { // Converted from nanoseconds to milliseconds
					t.Errorf("Overhead = %v, want 5.0", log.Overhead)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSON(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestIsAccessLogLine tests access log line detection
func TestIsAccessLogLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "IPv4 at start",
			line:     "192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234",
			expected: true,
		},
		{
			name:     "IPv6 at start",
			line:     "2001:db8::1 - - [01/Jan/2024:12:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234",
			expected: true,
		},
		{
			name:     "pod name prefix with IP",
			line:     "[traefik-abc123] 192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234",
			expected: true,
		},
		{
			name:     "common log format timestamp",
			line:     "- - - [01/Jan/2024:12:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234",
			expected: true,
		},
		{
			name:     "empty string",
			line:     "",
			expected: false,
		},
		{
			name:     "not an access log",
			line:     "This is just a regular log message without IP address",
			expected: false,
		},
		{
			name:     "application error log",
			line:     "[ERROR] Failed to connect to database",
			expected: false,
		},
		{
			name:     "valid JSON log (should return false - not common log format)",
			line:     `{"level":"info","msg":"request received"}`,
			expected: false,
		},
		{
			name:     "IP with no spaces",
			line:     "127.0.0.1 some other text",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAccessLogLine(tt.line)
			if result != tt.expected {
				t.Errorf("isAccessLogLine() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestExtractServiceName tests router name parsing
func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name       string
		routerName string
		expected   string
	}{
		{
			name:       "standard ingress with @kubernetes",
			routerName: "websecure-monitoring-grafana-operator-grafana-ingress-grafana@kubernetes",
			expected:   "websecure-monitoring-grafana", // Takes first 3 parts
		},
		{
			name:       "IngressRoute CRD with @kubernetescrd",
			routerName: "mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08@kubernetescrd",
			expected:   "mahfil-api", // Finds 'api' pattern with previous part
		},
		{
			name:       "short router name",
			routerName: "api@kubernetes",
			expected:   "api",
		},
		{
			name:       "no @ symbol",
			routerName: "simple-router-name",
			expected:   "simple-router", // Returns first two parts when len < 3
		},
		{
			name:       "service pattern with api",
			routerName: "websecure-myapp-api-service-abc123@kubernetes",
			expected:   "myapp-api",
		},
		{
			name:       "service pattern with web",
			routerName: "websecure-myapp-web-service-abc123@kubernetes",
			expected:   "myapp-web",
		},
		{
			name:       "very short name",
			routerName: "abc",
			expected:   "abc",
		},
		{
			name:       "empty string",
			routerName: "",
			expected:   "",
		},
		{
			name:       "three parts with api",
			routerName: "namespace-service-api-rest@kubernetes",
			expected:   "service", // Finds 'api' with i=0, returns just 'api'
		},
		{
			name:       "four parts",
			routerName: "ns-app-api-server-route@kubernetes",
			expected:   "app-api", // Finds 'api' pattern with previous part
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.routerName)
			if result != tt.expected {
				t.Errorf("extractServiceName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeURL tests URL normalization with patterns
func TestNormalizeURL(t *testing.T) {
	patterns := []URLPattern{
		{
			ServiceName: "api-service",
			Pattern:     `/api/users/\d+`,
			Replacement: "/api/users/{id}",
			Namespace:   "default",
			Regex:       regexp.MustCompile(`/api/users/\d+`),
		},
	}

	tests := []struct {
		name        string
		serviceName string
		path        string
		urlPatterns []URLPattern
		expected    string
	}{
		{
			name:        "service-specific pattern matches",
			serviceName: "default-api-service",
			path:        "/api/users/123",
			urlPatterns: patterns,
			expected:    "/api/users/{id}",
		},
		{
			name:        "service-specific pattern does not match",
			serviceName: "default-api-service",
			path:        "/api/orders",
			urlPatterns: patterns,
			expected:    "/api/orders",
		},
		{
			name:        "default numeric ID replacement",
			serviceName: "other-service",
			path:        "/api/products/456",
			urlPatterns: []URLPattern{},
			expected:    "/api/products/{id}",
		},
		{
			name:        "default UUID replacement",
			serviceName: "other-service",
			path:        "/api/users/550e8400-e29b-41d4-a716-446655440000",
			urlPatterns: []URLPattern{},
			expected:    "/api/users/{uuid}",
		},
		{
			name:        "default token replacement",
			serviceName: "other-service",
			path:        "/api/auth/abc123def456ghi789jkl012mno345pqr",
			urlPatterns: []URLPattern{},
			expected:    "/api/auth/{token}",
		},
		{
			name:        "query params replacement",
			serviceName: "other-service",
			path:        "/api/search?q=test&limit=10",
			urlPatterns: []URLPattern{},
			expected:    "/api/search?{query_params}",
		},
		{
			name:        "complex path with multiple replacements",
			serviceName: "other-service",
			path:        "/api/users/550e8400-e29b-41d4-a716-446655440000/posts?sort=desc",
			urlPatterns: []URLPattern{},
			expected:    "/api/users/{uuid}/posts?{query_params}",
		},
		{
			name:        "path with trailing slash and ID",
			serviceName: "other-service",
			path:        "/api/orders/999/",
			urlPatterns: []URLPattern{},
			expected:    "/api/orders/{id}/",
		},
		{
			name:        "no replacement needed",
			serviceName: "other-service",
			path:        "/health",
			urlPatterns: []URLPattern{},
			expected:    "/health",
		},
		{
			name:        "empty path",
			serviceName: "other-service",
			path:        "",
			urlPatterns: []URLPattern{},
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile regexes for URLPatterns
			for i := range tt.urlPatterns {
				if tt.urlPatterns[i].Pattern != "" {
					re, err := regexp.Compile(tt.urlPatterns[i].Pattern)
					if err != nil {
						t.Fatalf("Failed to compile pattern: %v", err)
					}
					tt.urlPatterns[i].Regex = re
				}
			}
			result := normalizeURL(tt.serviceName, tt.path, tt.urlPatterns)
			if result != tt.expected {
				t.Errorf("normalizeURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBuildServiceName tests service name construction
func TestBuildServiceName(t *testing.T) {
	tests := []struct {
		testName  string
		namespace string
		serviceName string
		separator string
		expected  string
	}{
		{
			testName:  "both non-empty",
			namespace: "default",
			serviceName: "api",
			separator: "-",
			expected:  "default-api",
		},
		{
			testName:  "empty namespace",
			namespace: "",
			serviceName: "api",
			separator: "-",
			expected:  "api",
		},
		{
			testName:  "empty name",
			namespace: "default",
			serviceName: "",
			separator: "-",
			expected:  "default",
		},
		{
			testName:  "both empty",
			namespace: "",
			serviceName: "",
			separator: "-",
			expected:  "",
		},
		{
			testName:  "different separator",
			namespace: "kube-system",
			serviceName: "coredns",
			separator: "/",
			expected:  "kube-system/coredns",
		},
		{
			testName:  "whitespace trimming",
			namespace: "  default  ",
			serviceName: "  api  ",
			separator: "-",
			expected:  "default-api",
		},
		{
			testName:  "separator with whitespace",
			namespace: "default",
			serviceName: "api",
			separator: " - ",
			expected:  "default - api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := BuildServiceName(tt.namespace, tt.serviceName, tt.separator)
			if result != tt.expected {
				t.Errorf("BuildServiceName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestContains tests service slice membership checking
func TestContains(t *testing.T) {
	services := []TraefikService{
		{Name: "api", Namespace: "default"},
		{Name: "web", Namespace: "kube-system"},
		{Name: "db", Namespace: "production"},
	}

	tests := []struct {
		name     string
		slice    []TraefikService
		item     string
		expected bool
	}{
		{
			name:     "service exists in slice",
			slice:    services,
			item:     "default-api",
			expected: true,
		},
		{
			name:     "service does not exist",
			slice:    services,
			item:     "test-api",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []TraefikService{},
			item:     "default-api",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "default-api",
			expected: false,
		},
		{
			name:     "different namespace same name",
			slice:    services,
			item:     "production-api",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStartsWith tests prefix matching in service slice
func TestStartsWith(t *testing.T) {
	services := []TraefikService{
		{Name: "api", Namespace: "default"},
		{Name: "web", Namespace: "kube-system"},
		{Name: "backend", Namespace: "production"},
	}

	tests := []struct {
		name     string
		slice    []TraefikService
		item     string
		expected bool
	}{
		{
			name:     "item starts with service name",
			slice:    services,
			item:     "default-api-v2",
			expected: true,
		},
		{
			name:     "exact match",
			slice:    services,
			item:     "default-api",
			expected: true,
		},
		{
			name:     "no prefix match",
			slice:    services,
			item:     "test-service",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []TraefikService{},
			item:     "default-api",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "default-api",
			expected: false,
		},
		{
			name:     "different namespace",
			slice:    services,
			item:     "production-backend-v2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startsWith(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("startsWith() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCountTotalTopPaths tests counting top paths across services
func TestCountTotalTopPaths(t *testing.T) {
	tests := []struct {
		name     string
		tps      map[string]map[string]bool
		expected int
	}{
		{
			name: "multiple services with paths",
			tps: map[string]map[string]bool{
				"service1": {
					"service1:/api/users": true,
					"service1:/api/orders": true,
				},
				"service2": {
					"service2:/health": true,
				},
			},
			expected: 3,
		},
		{
			name:     "empty map",
			tps:      map[string]map[string]bool{},
			expected: 0,
		},
		{
			name:     "nil map",
			tps:      nil,
			expected: 0,
		},
		{
			name: "service with no paths",
			tps: map[string]map[string]bool{
				"service1": {},
			},
			expected: 0,
		},
		{
			name: "single service single path",
			tps: map[string]map[string]bool{
				"service1": {
					"service1:/api": true,
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countTotalTopPaths(tt.tps)
			if result != tt.expected {
				t.Errorf("countTotalTopPaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}
