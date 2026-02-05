package logprocessing

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status     string            `json:"status"`
	Uptime     string            `json:"uptime,omitempty"`
	Components map[string]string `json:"components,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// Global variables for health status
var (
	healthStatus      HealthStatus
	healthMutex       sync.RWMutex
	startupTime       = time.Now()
	lastProcessedTime time.Time
)

// Initialize health status
func init() {
	healthStatus = HealthStatus{
		Status: "starting",
		Components: map[string]string{
			"service": "initializing",
		},
	}
	lastProcessedTime = time.Now()
}

// SetServiceReady updates the service status to ready
func SetServiceReady() {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	if healthStatus.Components == nil {
		healthStatus.Components = make(map[string]string)
	}
	healthStatus.Status = "healthy"
	healthStatus.Components["service"] = "running"
}

// UpdateHealthStatus updates the health status of a component
func UpdateHealthStatus(component, status string, err error) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	if healthStatus.Components == nil {
		healthStatus.Components = make(map[string]string)
	}

	healthStatus.Components[component] = status
	if err != nil {
		healthStatus.Status = "error"
		healthStatus.Error = err.Error()
	} else if healthStatus.Status != "error" {
		healthStatus.Status = "healthy"
	}
}

// UpdateLastProcessedTime updates the timestamp of the last processed log line
func UpdateLastProcessedTime() {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	lastProcessedTime = time.Now()
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	healthMutex.RLock()
	status := healthStatus
	lastProcessed := lastProcessedTime
	healthMutex.RUnlock()

	// Create a response copy to avoid concurrent map writes
	response := HealthStatus{
		Status:     status.Status,
		Uptime:     time.Since(startupTime).Round(time.Second).String(),
		Components: make(map[string]string),
		Error:      status.Error,
	}

	// Safely copy the components
	for k, v := range status.Components {
		response.Components[k] = v
	}

	// Check if we're processing logs
	if time.Since(lastProcessed) > 5*time.Minute {
		response.Components["log_processing"] = "stale"
		if response.Status == "healthy" {
			response.Status = "degraded"
			response.Error = "No logs processed in the last 5 minutes"
		}
	} else {
		response.Components["log_processing"] = "active"
	}

	w.Header().Set("Content-Type", "application/json")
	if response.Status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
}
