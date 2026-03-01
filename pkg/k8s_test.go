package logprocessing

import (
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
)

// TestHomeDir tests the homeDir utility function
func TestHomeDir(t *testing.T) {
	// Save original env vars
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	tests := []struct {
		name     string
		setHome  string
		setProfile string
		expected string
	}{
		{
			name:     "HOME environment variable is set",
			setHome:  "/test/home",
			setProfile: "",
			expected: "/test/home",
		},
		{
			name:     "USERPROFILE environment variable is set (Windows)",
			setHome:  "",
			setProfile: "C:\\Users\\test",
			expected: "C:\\Users\\test",
		},
		{
			name:     "no home directory set",
			setHome:  "",
			setProfile: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HOME", tt.setHome)
			os.Setenv("USERPROFILE", tt.setProfile)

			result := homeDir()
			if result != tt.expected {
				t.Errorf("homeDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestK8SConfig tests the K8SConfig struct
func TestK8SConfig(t *testing.T) {
	config := K8SConfig{
		InCluster:     false,
		KubeConfig:    "/path/to/kubeconfig",
		Context:       "test-context",
		Namespace:     "test-namespace",
		ContainerName: "traefik",
		LabelSelector: "app=traefik",
	}

	if config.InCluster {
		t.Error("Expected InCluster to be false")
	}

	if config.KubeConfig != "/path/to/kubeconfig" {
		t.Errorf("Expected KubeConfig '/path/to/kubeconfig', got %s", config.KubeConfig)
	}

	if config.Namespace != "test-namespace" {
		t.Errorf("Expected Namespace 'test-namespace', got %s", config.Namespace)
	}

	if config.LabelSelector != "app=traefik" {
		t.Errorf("Expected LabelSelector 'app=traefik', got %s", config.LabelSelector)
	}
}

// TestKubernetesLogSourceStruct tests the KubernetesLogSource struct
func TestKubernetesLogSourceStruct(t *testing.T) {
	kls := &KubernetesLogSource{
		namespace:     "default",
		containerName: "traefik",
		labelSelector: "app=traefik",
		lines:         make(chan LogLine, 100),
		podStreams:    make(map[string]*podStream),
		stopCh:        make(chan struct{}),
	}

	if kls.namespace != "default" {
		t.Errorf("Expected namespace 'default', got %s", kls.namespace)
	}

	if kls.containerName != "traefik" {
		t.Errorf("Expected container name 'traefik', got %s", kls.containerName)
	}

	if cap(kls.lines) != 100 {
		t.Errorf("Expected lines channel capacity 100, got %d", cap(kls.lines))
	}
}

// TestIsContainerReady tests the isContainerReady helper function
func TestIsContainerReady(t *testing.T) {
	tests := []struct {
		name          string
		pod           *v1.Pod
		containerName string
		expected      bool
	}{
		{
			name: "container is ready",
			pod: &v1.Pod{
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "traefik",
							Ready: true,
						},
					},
				},
			},
			containerName: "traefik",
			expected:      true,
		},
		{
			name: "container is not ready",
			pod: &v1.Pod{
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "traefik",
							Ready: false,
						},
					},
				},
			},
			containerName: "traefik",
			expected:      false,
		},
		{
			name: "container not found",
			pod: &v1.Pod{
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "other-container",
							Ready: true,
						},
					},
				},
			},
			containerName: "traefik",
			expected:      false,
		},
		{
			name:          "no container statuses",
			pod:           &v1.Pod{},
			containerName: "traefik",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContainerReady(tt.pod, tt.containerName)
			if result != tt.expected {
				t.Errorf("isContainerReady() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPodStreamStruct tests the podStream struct
func TestPodStreamStruct(t *testing.T) {
	cancel := func() {}

	ps := &podStream{
		cancelFunc: cancel,
		podName:    "test-pod",
	}

	if ps.podName != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got %s", ps.podName)
	}

	if ps.cancelFunc == nil {
		t.Error("Expected cancelFunc to be set")
	}
}

// TestKubernetesLogSourceClose tests the Close method
func TestKubernetesLogSourceClose(t *testing.T) {
	kls := &KubernetesLogSource{
		lines:      make(chan LogLine, 100),
		podStreams: make(map[string]*podStream),
		stopCh:     make(chan struct{}),
	}

	// Add a mock pod stream
	cancel := func() {}
	kls.podStreams["test-pod"] = &podStream{
		cancelFunc: cancel,
		podName:    "test-pod",
	}

	// Close should not panic
	err := kls.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify stopCh is closed
	select {
	case <-kls.stopCh:
		// Expected
	default:
		t.Error("Expected stopCh to be closed")
	}
}

// TestNewKubernetesConfigErrorPaths tests error scenarios for NewKubernetesConfig
func TestNewKubernetesConfigErrorPaths(t *testing.T) {
	// Save original env vars
	oldHome := os.Getenv("HOME")
	oldKubeConfig := os.Getenv("KUBERNETES_SERVICE_HOST")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("KUBERNETES_SERVICE_HOST", oldKubeConfig)
	}()

	// Unset Kubernetes environment variables
	os.Unsetenv("KUBERNETES_SERVICE_HOST")

	tests := []struct {
		name        string
		config      K8SConfig
		setup       func()
		expectedErr bool
	}{
		{
			name: "invalid kubeconfig path",
			config: K8SConfig{
				InCluster:  false,
				KubeConfig: "/non/existent/kubeconfig",
			},
			expectedErr: true,
		},
		{
			name: "in-cluster config without service host",
			config: K8SConfig{
				InCluster: true,
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			_, err := NewKubernetesConfig(tt.config)

			if (err != nil) != tt.expectedErr {
				t.Errorf("NewKubernetesConfig() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

// TestForcePodResync tests the forcePodResync method
func TestForcePodResync(t *testing.T) {
	kls := &KubernetesLogSource{
		podMutex:    sync.Mutex{},
		lastPodSync: time.Now(),
	}

	// Set lastPodSync to a non-zero time
	kls.lastPodSync = time.Now()

	// Force resync should set lastPodSync to zero time
	kls.forcePodResync()

	if !kls.lastPodSync.IsZero() {
		t.Error("Expected lastPodSync to be zero after force resync")
	}
}

// TestReadLines tests the ReadLines method
func TestReadLines(t *testing.T) {
	lines := make(chan LogLine, 10)
	kls := &KubernetesLogSource{
		lines: lines,
	}

	result := kls.ReadLines()

	if result != lines {
		t.Error("Expected ReadLines to return the lines channel")
	}
}

// TestPodExists tests the podExists method (without actual k8s client)
func TestPodExists(t *testing.T) {
	// This test would require a mock Kubernetes client
	// We skip it since we can't easily create a clientset without a real cluster
	t.Skip("Skipping podExists test - requires Kubernetes clientset")

	kls := &KubernetesLogSource{
		namespace: "default",
	}

	// Without a real clientset, this would fail
	// In a real test, you'd use a mock clientset
	_, err := kls.podExists("test-pod")

	// We expect an error because clientSet is nil
	if err == nil {
		t.Error("Expected error when clientSet is nil")
	}
}

// TestConstants tests the constants defined in k8s.go
func TestConstants(t *testing.T) {
	if maxRetries != 10 {
		t.Errorf("Expected maxRetries = 10, got %d", maxRetries)
	}

	if initialBackoff != 1*time.Second {
		t.Errorf("Expected initialBackoff = 1s, got %v", initialBackoff)
	}

	if maxBackoff != 1*time.Minute {
		t.Errorf("Expected maxBackoff = 1m, got %v", maxBackoff)
	}

	if syncInterval != 10*time.Second {
		t.Errorf("Expected syncInterval = 10s, got %v", syncInterval)
	}

	if podDiscoveryTimeout != 15*time.Second {
		t.Errorf("Expected podDiscoveryTimeout = 15s, got %v", podDiscoveryTimeout)
	}
}

// TestAddKubernetesFlags tests the AddKubernetesFlags function
func TestAddKubernetesFlags(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	config := AddKubernetesFlags(flags)

	if config == nil {
		t.Fatal("Expected config to be returned")
	}

	// Check default values
	if config.Namespace != "ingress-controller" {
		t.Errorf("Expected default namespace 'ingress-controller', got %s", config.Namespace)
	}

	if config.LabelSelector != "app.kubernetes.io/name=traefik" {
		t.Errorf("Expected default label selector 'app.kubernetes.io/name=traefik', got %s", config.LabelSelector)
	}

	if config.ContainerName != "traefik" {
		t.Errorf("Expected default container name 'traefik', got %s", config.ContainerName)
	}

	if config.InCluster {
		t.Error("Expected InCluster to be false by default")
	}
}

// TestKubernetesLogSourceMethods tests various methods of KubernetesLogSource
func TestKubernetesLogSourceMethods(t *testing.T) {
	kls := &KubernetesLogSource{
		namespace:     "test-ns",
		containerName: "test-container",
		labelSelector: "app=test",
		lines:         make(chan LogLine, 100),
		podStreams:    make(map[string]*podStream),
		stopCh:        make(chan struct{}),
	}

	t.Run("ReadLines returns channel", func(t *testing.T) {
		lines := kls.ReadLines()
		if lines == nil {
			t.Error("Expected ReadLines to return a channel")
		}
		if lines != kls.lines {
			t.Error("Expected ReadLines to return the lines channel")
		}
	})

	t.Run("Close without panic", func(t *testing.T) {
		kls2 := &KubernetesLogSource{
			lines:      make(chan LogLine, 100),
			podStreams: make(map[string]*podStream),
			stopCh:     make(chan struct{}),
		}
		// Should not panic
		err := kls2.Close()
		if err != nil {
			t.Errorf("Close() returned unexpected error: %v", err)
		}
	})
}
