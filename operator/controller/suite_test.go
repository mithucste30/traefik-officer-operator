package controller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	traefikofficerv1alpha1 "github.com/mithucste30/traefik-officer-operator/operator/api/v1alpha1"
	"github.com/mithucste30/traefik-officer-operator/shared"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	cancelCtx context.CancelFunc
	ctx       context.Context
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	ctx, cancelCtx = context.WithCancel(context.Background())

	// Get the root directory of the project
	projectDir, err := os.Getwd()
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	// Navigate to the operator directory (operator/controller -> operator)
	operatorDir := filepath.Join(projectDir, "..")

	// Setup the test environment with CRD directory
	crdPath := filepath.Join(operatorDir, "crd", "bases")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: false,
	}

	// Start the test environment
	var err2 error
	cfg, err2 = testEnv.Start()
	Expect(err2).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Add the CRD types to the scheme
	err = networkingv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = traefikofficerv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create a client for the test environment
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Debug: Check what's registered in the scheme
	gvk := schema.GroupVersionKind{Group: "traefikofficer.io", Version: "v1alpha1", Kind: "UrlPerformance"}
	GinkgoWriter.Printf("Checking if %v is registered in scheme...\n", gvk)

	// List all registered types for traefikofficer.io
	types := k8sClient.Scheme().AllKnownTypes()
	traefikofficerTypes := make([]string, 0)
	for registeredGVK := range types {
		if registeredGVK.Group == "traefikofficer.io" {
			traefikofficerTypes = append(traefikofficerTypes, registeredGVK.String())
		}
	}
	GinkgoWriter.Printf("Registered traefikofficer.io types: %v\n", traefikofficerTypes)
	GinkgoWriter.Printf("Total registered types: %d\n", len(types))

	// Check if UrlPerformance is in the list
	found := false
	for _, t := range traefikofficerTypes {
		if t == gvk.String() {
			found = true
			break
		}
	}
	GinkgoWriter.Printf("UrlPerformance found in scheme: %v\n", found)

	By("CRDs installed successfully")

	// Verify CRD is installed in API server
	By("verifying CRD is installed in API server")
	crdList := &v1.CustomResourceDefinitionList{}
	err = k8sClient.List(ctx, crdList)
	Expect(err).NotTo(HaveOccurred(), "Should be able to list CRDs")

	crdFound := false
	for _, installedCrd := range crdList.Items {
		if installedCrd.Name == "urlperformances.traefikofficer.io" {
			crdFound = true
			GinkgoWriter.Printf("Found CRD: %s\n", installedCrd.Name)
			break
		}
	}
	Expect(crdFound).To(BeTrue(), "CRD urlperformances.traefikofficer.io should be installed")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if testEnv != nil {
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
	cancelCtx()
})

// createTestConfigManager creates a ConfigManager for testing
func createTestConfigManager() *ConfigManager {
	return NewConfigManager()
}

// createTestRuntimeConfig creates a RuntimeConfig for testing
func createTestRuntimeConfig() *shared.RuntimeConfig {
	return &shared.RuntimeConfig{
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
}

// waitForResource waits for a resource to be created/updated
func waitForResource(ctx context.Context, client client.Client, key client.ObjectKey, obj client.Object) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for resource")
		default:
			err := client.Get(ctx, key, obj)
			if err == nil {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// skipIfEnvtestNotConfigured skips the test if envtest is not configured
func skipIfEnvtestNotConfigured() {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" && testEnv == nil {
		Skip("Skipping test as envtest is not configured. Set KUBEBUILDER_ASSETS or run with envtest.")
	}
}

// getTestServerURL returns the test server URL for webhook tests
func getTestServerURL() string {
	u, _ := url.Parse(cfg.Host)
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u.String()
}

// getTestServerCert returns the test server certificate for webhook tests
func getTestServerCert() string {
	if cfg.CAData != nil {
		return string(cfg.CAData)
	}
	return ""
}

// getTestServerTLSConfig returns TLS config for webhook tests
func getTestServerTLSConfig() *tls.Config {
	if cfg.CAData == nil {
		return &tls.Config{InsecureSkipVerify: true}
	}
	certPool := createCertPool()
	certPool.AppendCertsFromPEM([]byte(getTestServerCert()))
	return &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}
}

// createCertPool creates a certificate pool
func createCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	return pool
}
