# Testing Guide — traefik-officer-operator

> **Purpose**: This document is a complete specification for an AI coding agent to implement
> a full test suite for the `traefik-officer-operator` project.
> Read this entirely before writing any code.

---

## Project Overview

**Repo**: https://github.com/mithucste30/traefik-officer-operator  
**Module**: `github.com/mithucste30/traefik-officer-operator` (verify in `go.mod`)  
**What it does**: A Kubernetes operator that watches `UrlPerformance` CRDs, reads Traefik
access logs from k8s pod logs or files, parses router names to extract namespace + ingress,
applies path filtering, and exposes Prometheus metrics at `/metrics`.

**Key CRD**: `UrlPerformance` (`traefikofficer.io/v1alpha1`)  
**Key directories**:
- `operator/` — controller/reconciler logic
- `pkg/` — core business logic (parsing, filtering, metrics)
- `shared/` — shared types/interfaces
- `examples/urlperformances/` — example CRD manifests
- `test/` — existing test scaffolding (check what already exists here first)
- `config/crd/bases/` — generated CRD YAML manifests (needed for envtest)

---

## Pre-conditions for the Agent

Before writing any test, the agent MUST:

1. Read `go.mod` to get the exact module name and Go version
2. Read all files in `operator/` to understand the reconciler structure
3. Read all files in `pkg/` to understand business logic boundaries
4. Read all files in `shared/` to understand types and interfaces
5. Check `test/` for any existing tests to avoid duplication
6. Confirm that `config/crd/bases/` contains generated CRD YAML — if not, run:
   ```bash
   make generate
   make manifests
   ```

---

## Testing Stack

| Tool | Purpose |
|------|---------|
| `sigs.k8s.io/controller-runtime/pkg/envtest` | Spin up real etcd + kube-apiserver for integration tests |
| `sigs.k8s.io/controller-runtime/pkg/client/fake` | Fake k8s client for pure unit tests of reconciler |
| `github.com/onsi/ginkgo/v2` | BDD test framework (standard for kubebuilder projects) |
| `github.com/onsi/gomega` | Assertion library paired with Ginkgo |
| `github.com/stretchr/testify` | Optional — for simpler unit tests outside Ginkgo |
| `sigs.k8s.io/controller-runtime/tools/setup-envtest` | Download envtest binaries (etcd, kube-apiserver) |

Install missing dependencies:
```bash
go get sigs.k8s.io/controller-runtime
go get github.com/onsi/ginkgo/v2
go get github.com/onsi/gomega
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

---

## Layer 1 — Unit Tests (`pkg/` and `shared/`)

These tests have **zero Kubernetes dependency**. They test pure Go logic.

### 1.1 Router Name Parser

The operator parses Traefik router names into `(namespace, ingressName)` pairs.
There are two formats — Standard Ingress and IngressRoute CRD.

**File to create**: `pkg/parser/router_name_parser_test.go` (or wherever the parser lives)

**Test cases the agent MUST cover**:

```
Input (Standard Ingress):
  "websecure-monitoring-grafana-operator-grafana-ingress-grafana-non-production-kahf-co@kubernetes"
Expected:
  namespace = "monitoring"
  ingress   = "grafana-operator-grafana-ingress"

Input (IngressRoute CRD):
  "mahfil-dev-mahfil-api-server-ingressroute-http-a457d08d5820f79b3e08@kubernetescrd"
Expected:
  namespace = "mahfil-dev"
  ingress   = "mahfil-api-server-ingressroute-http"

Input (unknown suffix):
  "someprefix-some-name@unknownprovider"
Expected:
  error or empty strings (define behavior, don't panic)

Input (empty string):
  ""
Expected:
  error, no panic

Input (malformed, no @ separator):
  "completelyinvalidroutername"
Expected:
  error, no panic
```

### 1.2 Path Filtering Logic

The operator supports `whitelistPathsRegex` and `ignoredPathsRegex` (blacklist).

**Test cases**:

```
Whitelist = ["^/api/"]
  "/api/users"       → included (true)
  "/api/v2/orders"   → included (true)
  "/static/app.js"   → excluded (false)
  "/health"          → excluded (false)

Blacklist = ["\\.css$", "\\.js$", "^/static/"]
  "/style.css"       → ignored (true)
  "/app.js"          → ignored (true)
  "/static/img.png"  → ignored (true)
  "/api/users"       → not ignored (false)

Both whitelist AND blacklist set (blacklist takes priority):
  whitelist = ["^/api/"], blacklist = ["\\.css$"]
  "/api/style.css"   → ignored due to blacklist
  "/api/users"       → included

Invalid regex in whitelist → constructor/compile step returns error, does not panic

Empty whitelist (nil/[]) → all paths pass through (no filtering)
Empty blacklist (nil/[]) → no paths are ignored
```

### 1.3 URL Normalization / Merge Paths

The operator reduces metric cardinality by merging paths (e.g., `/api/users/123` → `/api/users/{id}`).

**Test cases**:

```
mergePathsWithExtensions = ["/api/"]
  "/api/users/123"     → "/api/users/{id}" or "/api/" (check actual behavior)
  "/api/orders/abc-def" → normalized form
  "/other/path"         → unchanged

No merge config → all paths pass through unchanged

Multiple merge patterns → each applied in order
```

### 1.4 Top-N Path Tracking

The operator tracks top N paths by latency via `collectNTop`.

**Test cases**:

```
collectNTop = 3, insert 5 paths with different latencies
  → only top 3 by latency are retained

collectNTop = 0 or unset → behavior is defined (either off, or all tracked)

Equal latency tie → deterministic result (define and test)

Thread safety: concurrent writes to the top-N tracker → no data race
  (run with go test -race)
```

### 1.5 Log Line Parsing

The operator parses Traefik access log lines into structured data.

**Test cases** (use real Traefik access log format):

```
Valid log line (JSON format):
  {"RouterName":"websecure-monitoring-grafana@kubernetes","RequestPath":"/api/v1","Duration":0.045,"DownstreamStatus":200}
Expected:
  RouterName parsed
  Path = "/api/v1"
  Duration = 45ms
  StatusCode = 200

Valid log line (Common Log Format if supported):
  test with actual format from pkg/ code

Malformed log line → error returned, no panic

Empty line → handled gracefully

Missing RouterName field → handled, metric emitted with empty/unknown labels
```

---

## Layer 2 — Controller Integration Tests (envtest)

These tests run a real kube-apiserver + etcd locally via `envtest`.
The controller runs in a goroutine and reconciles against this real API server.

### 2.1 Test Suite Bootstrap

**File to create**: `operator/suite_test.go`

```go
package operator_test

import (
    "context"
    "path/filepath"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "k8s.io/client-go/rest"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"

    // import your API types and scheme
)

var (
    cfg       *rest.Config
    k8sClient client.Client
    testEnv   *envtest.Environment
    ctx       context.Context
    cancel    context.CancelFunc
)

func TestOperator(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Operator Suite")
}

var _ = BeforeSuite(func() {
    ctrl.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
    ctx, cancel = context.WithCancel(context.TODO())

    testEnv = &envtest.Environment{
        CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
        ErrorIfCRDPathMissing: true,
    }

    var err error
    cfg, err = testEnv.Start()
    Expect(err).NotTo(HaveOccurred())

    // Register scheme, create client, start manager + reconciler here
    // Inject a FAKE LogSource (see Section 2.3) so no real Traefik is needed

    go func() {
        defer GinkgoRecover()
        Expect(mgr.Start(ctx)).To(Succeed())
    }()
})

var _ = AfterSuite(func() {
    cancel()
    Expect(testEnv.Stop()).To(Succeed())
})
```

### 2.2 Reconciler Test Scenarios

**File to create**: `operator/reconciler_test.go`

The agent must implement ALL of the following scenarios:

#### Scenario A — Create CR → reconciler picks it up

```
Given: a UrlPerformance CR with valid spec (enabled=true, valid regexes)
When:  created via k8sClient.Create()
Then:
  - reconciler does not error
  - CR status is updated (e.g., status.ready=true or status.phase="Active")
  - poll with Eventually(timeout=10s, interval=250ms)
```

#### Scenario B — Disabled CR is ignored

```
Given: a UrlPerformance CR with spec.enabled=false
When:  created via k8sClient.Create()
Then:
  - reconciler picks it up but does NOT start log monitoring
  - status reflects "Disabled" or similar
  - no goroutine leak (verify via fake log source not being called)
```

#### Scenario C — Update CR → reconciler re-reconciles

```
Given: a UrlPerformance CR that has already been reconciled (scenario A)
When:  spec.collectNTop is changed (e.g., 10 → 20) via k8sClient.Update()
Then:
  - reconciler re-reconciles
  - new config is applied (observable via fake log source or status field)
  - old config is discarded
```

#### Scenario D — Delete CR → cleanup

```
Given: a UrlPerformance CR that has already been reconciled
When:  deleted via k8sClient.Delete()
Then:
  - reconciler runs finalizer/cleanup logic
  - associated monitoring goroutine (if any) is stopped
  - no orphaned state remains
```

#### Scenario E — Invalid regex in spec

```
Given: a UrlPerformance CR with spec.whitelistPathsRegex = ["[invalid"]
When:  created
Then:
  - reconciler does not crash
  - CR status.phase = "Error" or equivalent
  - status.message contains a human-readable error about invalid regex
```

#### Scenario F — Multiple CRs are independent

```
Given: two UrlPerformance CRs in different namespaces, each with different regexes
When:  both are created
Then:
  - each is reconciled independently
  - config of CR-A does not bleed into CR-B
  - deleting CR-A does not affect CR-B
```

#### Scenario G — CR targeting non-existent Ingress

```
Given: a UrlPerformance CR with spec.targetRef.name = "does-not-exist"
When:  created
Then:
  - reconciler handles gracefully (no panic)
  - status reflects warning or "Pending" state
  - operator requeues to retry (does not give up permanently)
```

### 2.3 Fake Log Source — CRITICAL

The operator reads Traefik logs from k8s pod logs. In tests, this MUST be mocked.

The agent must:

1. **Identify the log source abstraction** — look in `operator/` and `shared/` for an interface
   like `LogReader`, `LogSource`, `LogStreamer`, or similar.

2. **If an interface already exists** — implement a fake:
   ```go
   type fakeLogSource struct {
       lines []string
       done  chan struct{}
   }

   func (f *fakeLogSource) Stream(ctx context.Context) (<-chan string, error) {
       ch := make(chan string, len(f.lines))
       for _, line := range f.lines {
           ch <- line
       }
       close(ch)
       return ch, nil
   }
   ```

3. **If NO interface exists** — the agent must REFACTOR the operator to introduce one:
   - Extract the log reading behavior into a `LogSource` interface in `shared/` or `operator/`
   - Wire the real k8s log reader via this interface in `cmd/`
   - Inject the fake in tests via constructor parameter or functional option

   This refactor is REQUIRED to make the controller testable. Do not skip it.

---

## Layer 3 — Metrics Endpoint Tests

These tests verify the Prometheus metrics exposed at `/metrics`.

**File to create**: `operator/metrics_test.go` (or `pkg/metrics/metrics_test.go`)

### 3.1 Metric Cardinality and Labels

```
Given: fake log lines are streamed for a UrlPerformance CR targeting namespace="monitoring", ingress="grafana"
When:  the fake log source emits:
  - 3 requests to /api/v1 (200 OK, ~50ms each)
  - 1 request to /api/v1 (500 error, ~100ms)
  - 2 requests to /static/app.js (200 OK) — this path is blacklisted
Then at /metrics:
  - traefik_officer_requests_total{namespace="monitoring",ingress="grafana"} >= 4
  - traefik_officer_endpoint_requests_total{response_code="500",...} == 1
  - /static/app.js does NOT appear in any metric labels (filtered out)
  - namespace and ingress labels are present on ALL metrics
```

### 3.2 Histogram Quantiles

```
Given: 100 fake requests with varying latencies (use a distribution: 50% < 100ms, 95% < 500ms, 99% < 1s)
Then:
  - traefik_officer_request_duration_seconds_bucket is present
  - histogram_quantile(0.95,...) is calculable from buckets
```

### 3.3 Metric Reset on CR Delete

```
Given: a CR has accumulated metrics
When:  the CR is deleted
Then:
  - metrics for that namespace/ingress are removed (or zeroed)
  - /metrics endpoint no longer exposes stale labels for deleted CR
```

---

## File Structure the Agent Should Create

```
operator/
  suite_test.go          ← envtest bootstrap (BeforeSuite/AfterSuite)
  reconciler_test.go     ← Scenarios A–G
  metrics_test.go        ← Metric label/cardinality tests

pkg/
  parser/
    router_name_parser_test.go   ← Unit tests for router name parsing
  filter/
    path_filter_test.go          ← Unit tests for whitelist/blacklist
  normalizer/
    url_normalizer_test.go       ← Unit tests for URL normalization
  topn/
    topn_tracker_test.go         ← Unit tests for Top-N tracking
  logparser/
    log_line_parser_test.go      ← Unit tests for log line parsing

shared/
  fake_log_source_test.go        ← OR wherever fakeLogSource lives
```

> Note: Adjust package paths to match the actual directory structure found in the repo.

---

## Running the Tests

### Setup envtest binaries

```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
export KUBEBUILDER_ASSETS=$(setup-envtest use 1.29.x --bin-dir /usr/local/kubebuilder/bin -p path)
```

### Run all tests

```bash
go test ./... -v -race
```

### Run only unit tests (no envtest)

```bash
go test ./pkg/... -v -race
```

### Run only integration tests

```bash
go test ./operator/... -v -race
```

### Run with Ginkgo CLI (optional, better output)

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
ginkgo -v -race ./...
```

### Run with coverage

```bash
go test ./... -coverprofile=coverage.out -race
go tool cover -html=coverage.out
```

---

## CI Integration (GitHub Actions)

Add to `.github/workflows/ci.yml`:

```yaml
- name: Install setup-envtest
  run: go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

- name: Set KUBEBUILDER_ASSETS
  run: echo "KUBEBUILDER_ASSETS=$(setup-envtest use 1.29.x -p path)" >> $GITHUB_ENV

- name: Run tests
  run: go test ./... -v -race -coverprofile=coverage.out

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    files: coverage.out
```

---

## Acceptance Criteria

The test suite is considered complete when:

- [ ] All unit tests in `pkg/` pass with `go test -race`
- [ ] All envtest integration tests pass (Scenarios A–G)
- [ ] Metrics tests verify label correctness
- [ ] `go test -race ./...` produces zero race condition warnings
- [ ] `make test` succeeds (update Makefile target if needed)
- [ ] Code coverage is ≥ 70% across `pkg/` and `operator/`
- [ ] CI pipeline runs tests automatically on every PR

---

## Common Pitfalls

- **KUBEBUILDER_ASSETS not set** → envtest panics on start. Always export it.
- **CRD YAML missing** → `ErrorIfCRDPathMissing: true` will fail loudly. Run `make manifests` first.
- **Asserting immediately after Create()** → reconciliation is async. Always use `Eventually()`.
- **Forgetting -race flag** → the log streaming goroutines are highly concurrent; race conditions won't surface without it.
- **Not canceling context in AfterSuite** → manager goroutine leaks between test runs.
- **Real log source in tests** → if the reconciler tries to connect to a real k8s cluster for pod logs during envtest, tests will hang or fail. The fake log source injection (Section 2.3) is mandatory.
- **Scheme not registered** → if your CRD types aren't added to the scheme, `k8sClient.Create()` will return "no kind is registered" errors. Register all types in `BeforeSuite`.
