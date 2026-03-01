# Integration Test Issue - Scheme Registration Problem

**Date**: 2026-02-23
**Status**: Under Investigation - TypeMeta Not Being Preserved in client.Create()

## Problem Summary

Integration tests fail with error:
```
no kind is registered for the type v1alpha1.UrlPerformance in scheme "pkg/runtime/scheme.go:111"
```

**Root Cause**: The `client.Create()` method receives an object with correct TypeMeta, but the client's internal scheme doesn't recognize the type, causing the operation to fail.

## Investigation Progress

### ✅ Verified Working:
1. **CRD file exists** at `crd/bases/traefikofficer.io_urlperformances.yaml`
2. **CRD path fixed** in suite_test.go from `helm/traefik-officer-operator/crd` to `crd/bases`
3. **Scheme registration** in suite_test.go:
   ```go
   err = traefikofficerv1alpha1.AddToScheme(scheme.Scheme)
   Expect(err).NotTo(HaveOccurred())
   ```
4. **Client creation** with scheme:
   ```go
   k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
   ```
5. **Reconciler uses correct scheme**:
   ```go
   reconciler = &UrlPerformanceReconciler{
       Client:        k8sClient,
       Scheme:        scheme.Scheme,  // Changed from k8sClient.Scheme()
       ConfigManager: configManager,
   }
   ```
6. **UrlPerformance objects have TypeMeta set**:
   ```go
   testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
       TypeMeta: metav1.TypeMeta{
           Kind:       "UrlPerformance",
           APIVersion: "traefikofficer.io/v1alpha1",
       },
       ObjectMeta: metav1.ObjectMeta{
           Name:      "test-urlperf-a",
           Namespace: testNamespace,
       },
       ...
   }
   ```
7. **Debug output confirms TypeMeta is correct before Create()**:
   ```
   DEBUG: About to create UrlPerformance: Kind=UrlPerformance, APIVersion=traefikofficer.io/v1alpha1, Name=test-urlperf-a
   ```

### ❌ The Problem:
**Error shows empty GVK**:
```
gvk: {Group: "", Version: "", Kind: ""}
```

This indicates that somewhere between the test calling `k8sClient.Create()` and the scheme looking up the type, the TypeMeta information is being lost.

### 🔍 Key Finding:
The error references `"pkg/runtime/scheme.go:111"` which is the **client's internal scheme**, NOT the global `scheme.Scheme` we passed to `client.New()`.

This suggests that `client.New()` creates its own internal scheme that doesn't inherit all the types from the passed scheme, OR the envtest is using a different scheme internally.

## Possible Solutions

### Option 1: Directly Install CRD via Client
Instead of relying on envtest's CRDDirectoryPaths, manually install the CRD:
```go
// In BeforeSuite, after creating client
crdFile := filepath.Join(crdPath, "traefikofficer.io_urlperformances.yaml")
crdYAML, _ := os.ReadFile(crdFile)
crd := &apiextensionsv1.CustomResourceDefinition{}
scheme.Scheme.Decode(crdYAML, &crd)
err = k8sClient.Create(ctx, crd)
Expect(err).NotTo(HaveOccurred())
```

### Option 2: Use Scheme.Builder in Client Creation
The client.New() might need the scheme builder, not the scheme itself:
```go
import "sigs.k8s.io/controller-runtime/pkg/scheme"

builder := scheme.Builder{GroupVersion: traefikofficerv1alpha1.GroupVersion}
k8sClient, err = client.New(cfg, client.Options{
    Scheme: builder.Build()
})
```

### Option 3: Verify envtest CRD Installation
Add check to confirm CRD is actually in the API server:
```go
crdList := &apiextensionsv1.CustomResourceDefinitionList{}
err = k8sClient.List(ctx, crdList)
Expect(err).NotTo(HaveOccurred())
found := false
for _, crd := range crdList.Items {
    if crd.Name == "urlperformances.traefikofficer.io" {
        found = true
        break
    }
}
Expect(found).To(BeTrue(), "CRD should be installed")
```

### Option 4: Check Client Scheme
Verify the client is actually using the scheme we passed:
```go
// After creating client
GinkgoWriter.Printf("Client scheme: %p\n", k8sClient.Scheme())
GinkgoWriter.Printf("Global scheme: %p\n", scheme.Scheme)

// Check if UrlPerformance is in client's scheme
types := k8sClient.Scheme().AllKnownTypes()
for gvk := range types {
    if gvk.Group == "traefikofficer.io" {
        GinkgoWriter.Printf("Found in client scheme: %v\n", gvk)
    }
}
```

## Next Steps

1. **Add debug output to verify CRD is installed in API server** (Option 3)
2. **Check if client scheme matches global scheme** (Option 4)
3. **Try manually installing CRD** (Option 1)
4. **Consider using webhook or different client creation approach**

## Files Modified

- `operator/controller/suite_test.go`: Added debug output, fixed CRD path
- `operator/controller/urlperformance_controller_test.go`: Added TypeMeta to all UrlPerformance creations (8 instances)
- `TESTING_STATUS.md`: Created comprehensive testing status documentation

## Test Commands

```bash
# Run integration tests
export KUBEBUILDER_ASSETS="/Users/kahf/Library/Application Support/io.kubebuilder.envtest/k8s/1.30.0-darwin-amd64"
go test -v ./operator/controller/...
```

## References

- controller-runtime client scheme: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client
- envtest documentation: https://book.kubebuilder.io/reference/envtest.html
- Similar issues: Search for "no kind is registered client.New scheme"
