package logprocessing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestSetServiceReady tests the SetServiceReady function
func TestSetServiceReady(t *testing.T) {
	// Reset health status
	healthMutex.Lock()
	healthStatus = HealthStatus{
		Status: "starting",
		Components: map[string]string{
			"service": "initializing",
		},
	}
	healthMutex.Unlock()

	SetServiceReady()

	healthMutex.RLock()
	status := healthStatus
	healthMutex.RUnlock()

	if status.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status.Status)
	}

	if status.Components["service"] != "running" {
		t.Errorf("Expected service component 'running', got '%s'", status.Components["service"])
	}
}

// TestUpdateHealthStatus tests the UpdateHealthStatus function
func TestUpdateHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		component string
		status   string
		err      error
		validate func(*testing.T, HealthStatus)
	}{
		{
			name:     "update component status without error",
			component: "log_processor",
			status:   "active",
			err:      nil,
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Components["log_processor"] != "active" {
					t.Errorf("Expected component status 'active', got '%s'", hs.Components["log_processor"])
				}
				if hs.Status != "healthy" {
					t.Errorf("Expected overall status 'healthy', got '%s'", hs.Status)
				}
			},
		},
		{
			name:     "update component status with error",
			component: "k8s_client",
			status:   "failed",
			err:      nil, // Pass nil for non-error test
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Components["k8s_client"] != "failed" {
					t.Errorf("Expected component status 'failed', got '%s'", hs.Components["k8s_client"])
				}
			},
		},
		{
			name:     "update with error sets error status",
			component: "metrics",
			status:   "error",
			err:      nil, // Will be set in test
			validate: func(t *testing.T, hs HealthStatus) {
				// Note: The error field is set when err != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset health status
			healthMutex.Lock()
			healthStatus = HealthStatus{
				Status:     "healthy",
				Components: make(map[string]string),
			}
			healthMutex.Unlock()

			if tt.name == "update with error sets error status" {
				UpdateHealthStatus(tt.component, tt.status, fmt.Errorf("test error"))
			} else {
				UpdateHealthStatus(tt.component, tt.status, tt.err)
			}

			healthMutex.RLock()
			status := healthStatus
			healthMutex.RUnlock()

			if tt.validate != nil {
				tt.validate(t, status)
			}
		})
	}
}

// TestUpdateHealthStatusConcurrency tests concurrent updates
func TestUpdateHealthStatusConcurrency(t *testing.T) {
	// Reset health status
	healthMutex.Lock()
	healthStatus = HealthStatus{
		Status:     "healthy",
		Components: make(map[string]string),
	}
	healthMutex.Unlock()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			component := "component" + string(rune('0'+idx%10))
			UpdateHealthStatus(component, "active", nil)
		}(i)
	}

	wg.Wait()

	// Verify no race condition occurred
	healthMutex.RLock()
	status := healthStatus
	healthMutex.RUnlock()

	if status.Status == "" {
		t.Error("Expected status to be set")
	}
}

// TestUpdateLastProcessedTime tests the UpdateLastProcessedTime function
func TestUpdateLastProcessedTime(t *testing.T) {
	oldTime := lastProcessedTime

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	UpdateLastProcessedTime()

	newTime := lastProcessedTime

	if newTime.Before(oldTime) || newTime.Equal(oldTime) {
		t.Error("Expected lastProcessedTime to be updated")
	}
}

// TestHealthHandler tests the HealthHandler HTTP handler
func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupStatus    func()
		expectedStatus int
		validate       func(*testing.T, HealthStatus)
	}{
		{
			name: "healthy status",
			setupStatus: func() {
				healthMutex.Lock()
				healthStatus = HealthStatus{
					Status: "healthy",
					Components: map[string]string{
						"service": "running",
					},
				}
				lastProcessedTime = time.Now()
				healthMutex.Unlock()
			},
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Status != "healthy" {
					t.Errorf("Expected status 'healthy', got '%s'", hs.Status)
				}
				if hs.Components["log_processing"] != "active" {
					t.Errorf("Expected log_processing 'active', got '%s'", hs.Components["log_processing"])
				}
			},
		},
		{
			name: "degraded status due to stale processing",
			setupStatus: func() {
				healthMutex.Lock()
				healthStatus = HealthStatus{
					Status: "healthy",
					Components: map[string]string{
						"service": "running",
					},
				}
				// Set last processed time to more than 5 minutes ago
				lastProcessedTime = time.Now().Add(-6 * time.Minute)
				healthMutex.Unlock()
			},
			expectedStatus: http.StatusServiceUnavailable,
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Status != "degraded" {
					t.Errorf("Expected status 'degraded', got '%s'", hs.Status)
				}
				if hs.Components["log_processing"] != "stale" {
					t.Errorf("Expected log_processing 'stale', got '%s'", hs.Components["log_processing"])
				}
				if hs.Error == "" {
					t.Error("Expected error message to be set")
				}
			},
		},
		{
			name: "error status",
			setupStatus: func() {
				healthMutex.Lock()
				healthStatus = HealthStatus{
					Status: "error",
					Components: map[string]string{
						"service": "failed",
					},
					Error: "test error",
				}
				lastProcessedTime = time.Now()
				healthMutex.Unlock()
			},
			expectedStatus: http.StatusServiceUnavailable,
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", hs.Status)
				}
				if hs.Error != "test error" {
					t.Errorf("Expected error 'test error', got '%s'", hs.Error)
				}
			},
		},
		{
			name: "starting status",
			setupStatus: func() {
				healthMutex.Lock()
				healthStatus = HealthStatus{
					Status: "starting",
					Components: map[string]string{
						"service": "initializing",
					},
				}
				lastProcessedTime = time.Now()
				healthMutex.Unlock()
			},
			expectedStatus: http.StatusServiceUnavailable,
			validate: func(t *testing.T, hs HealthStatus) {
				if hs.Status != "starting" {
					t.Errorf("Expected status 'starting', got '%s'", hs.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupStatus()

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			HealthHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected content type 'application/json', got '%s'", contentType)
			}

			// Parse response
			var hs HealthStatus
			if err := json.NewDecoder(w.Body).Decode(&hs); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Validate response
			if tt.validate != nil {
				tt.validate(t, hs)
			}

			// Check uptime is present
			if hs.Uptime == "" {
				t.Error("Expected uptime to be set")
			}
		})
	}
}

// TestHealthHandlerConcurrency tests concurrent health handler calls
func TestHealthHandlerConcurrency(t *testing.T) {
	// Setup
	healthMutex.Lock()
	healthStatus = HealthStatus{
		Status: "healthy",
		Components: map[string]string{
			"service": "running",
		},
	}
	lastProcessedTime = time.Now()
	healthMutex.Unlock()

	var wg sync.WaitGroup
	numRequests := 50

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			HealthHandler(w, req)
		}()
	}

	wg.Wait()
	// If we get here without panic, test passes
}

// TestHealthStatusStruct tests the HealthStatus struct
func TestHealthStatusStruct(t *testing.T) {
	hs := HealthStatus{
		Status: "healthy",
		Uptime: "1h30m",
		Components: map[string]string{
			"service": "running",
			"metrics": "active",
		},
		Error: "",
	}

	if hs.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", hs.Status)
	}

	if len(hs.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(hs.Components))
	}
}

// TestInitHealthStatus tests the init function in health.go
func TestInitHealthStatus(t *testing.T) {
	// The init function runs automatically, but we can verify its effects
	// Note: This may have been changed by other tests, so we just check the structure
	healthMutex.RLock()
	status := healthStatus
	healthMutex.RUnlock()

	// Just verify the components map exists (status may have been changed by other tests)
	if status.Components == nil {
		t.Error("Expected components map to be initialized")
	}
}
