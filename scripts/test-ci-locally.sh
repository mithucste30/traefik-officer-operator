#!/bin/bash
# Test GitHub Actions CI locally
# This script runs ALL the same checks that the CI workflow does
# If this passes, the CI should pass 100%

set -e

echo "🚀 Starting Complete CI Tests Locally..."
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print success
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to print info
print_info() {
    echo -e "${YELLOW}▶️  $1${NC}"
}

# Function to print section header
print_section() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

# Track failures
FAILURES=0

###############################################################################

print_section "JOB 1: LINT"

# Test 1.1: Run golangci-lint
print_info "Running golangci-lint..."
if golangci-lint run --timeout=10m; then
    print_success "golangci-lint passed"
else
    print_error "golangci-lint failed"
    FAILURES=$((FAILURES + 1))
fi

# Test 1.2: Check go.mod
print_info "Checking go.mod tidy..."
go mod tidy
cd operator && go mod tidy && cd ..
if git diff --exit-code go.mod go.sum operator/go.mod operator/go.sum > /dev/null 2>&1; then
    print_success "go.mod is tidy"
else
    print_error "go.mod is not tidy"
    git diff go.mod go.sum operator/go.mod operator/go.sum || true
    FAILURES=$((FAILURES + 1))
fi

###############################################################################

print_section "JOB 2: TEST"

# Test 2.1: Run unit tests
print_info "Running unit tests..."
if go test -v -race ./pkg/... ./shared/...; then
    print_success "Unit tests passed"
else
    print_error "Unit tests failed"
    FAILURES=$((FAILURES + 1))
fi

# Test 2.2: Setup envtest
print_info "Setting up envtest..."
if command -v setup-envtest &> /dev/null; then
    export PATH=$HOME/go/bin:$PATH
    KUBEBUILDER_ASSETS=$(setup-envtest use 1.30.0 -p path)
    export KUBEBUILDER_ASSETS
    print_success "envtest setup complete"

    # Test 2.3: Run integration tests
    print_info "Running integration tests..."
    if go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        FAILURES=$((FAILURES + 1))
    fi
else
    print_info "setup-envtest not found, skipping integration tests"
    print_info "Install with: go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
fi

###############################################################################

print_section "JOB 3: BUILD"

# Test 3.1: Build standalone binary
print_info "Building standalone binary..."
if go build -o bin/traefik-officer ./cmd/traefik-officer; then
    print_success "Standalone binary built successfully"
    rm -f bin/traefik-officer
else
    print_error "Failed to build standalone binary"
    FAILURES=$((FAILURES + 1))
fi

# Test 3.2: Build operator binary
print_info "Building operator binary..."
if cd operator && go build -o ../bin/traefik-officer-operator . && cd ..; then
    print_success "Operator binary built successfully"
    rm -f bin/traefik-officer-operator
else
    print_error "Failed to build operator binary"
    FAILURES=$((FAILURES + 1))
fi

###############################################################################

print_section "JOB 4: DOCKER BUILD TEST"

# Test 4.1: Check if Docker is available
if command -v docker &> /dev/null; then
    # Test 4.2: Build standalone Docker image
    print_info "Building standalone Docker image..."
    if docker build -f ./Dockerfile -t traefik-officer:test . > /dev/null 2>&1; then
        print_success "Standalone Docker image built successfully"
        docker rmi traefik-officer:test > /dev/null 2>&1 || true
    else
        print_error "Failed to build standalone Docker image"
        FAILURES=$((FAILURES + 1))
    fi

    # Test 4.3: Build operator Docker image
    print_info "Building operator Docker image..."
    if docker build -f ./operator/Dockerfile -t traefik-officer-operator:test . > /dev/null 2>&1; then
        print_success "Operator Docker image built successfully"
        docker rmi traefik-officer-operator:test > /dev/null 2>&1 || true
    else
        print_error "Failed to build operator Docker image"
        FAILURES=$((FAILURES + 1))
    fi
else
    print_info "Docker not found, skipping Docker build tests"
    print_info "Install Docker Desktop to test Docker builds locally"
fi

###############################################################################

print_section "JOB 5: HELM LINT"

# Test 5.1: Check if Helm is installed
if command -v helm &> /dev/null; then
    # Test 5.2: Lint Helm chart
    print_info "Linting Helm chart..."
    if helm lint helm/traefik-officer-operator; then
        print_success "Helm chart lint passed"
    else
        print_error "Helm chart lint failed"
        FAILURES=$((FAILURES + 1))
    fi

    # Test 5.3: Helm template test
    print_info "Testing Helm chart template..."
    if helm template traefik-officer-operator helm/traefik-officer-operator --debug > /dev/null 2>&1; then
        print_success "Helm chart template test passed"
    else
        print_error "Helm chart template test failed"
        FAILURES=$((FAILURES + 1))
    fi
else
    print_info "Helm not found, skipping Helm lint"
    print_info "Install Helm with: brew install helm"
fi

###############################################################################

print_section "JOB 6: SECURITY SCAN"

# Test 6.1: Check if Trivy is installed
if command -v trivy &> /dev/null; then
    print_info "Running Trivy security scan..."
    if trivy fs --skip-dirs .git . > /dev/null 2>&1; then
        print_success "Trivy security scan passed"
    else
        print_error "Trivy security scan found vulnerabilities"
        print_info "Run 'trivy fs --skip-dirs .git .' for details"
        FAILURES=$((FAILURES + 1))
    fi
else
    print_info "Trivy not found, skipping security scan"
    print_info "Install Trivy with: brew install trivy"
fi

###############################################################################

# Final Summary
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

if [ $FAILURES -eq 0 ]; then
    print_success "🎉 ALL CI TESTS PASSED LOCALLY!"
    echo ""
    echo "You can now push to GitHub with 100% confidence!"
    echo ""
    echo "Tests that passed:"
    echo "  ✅ golangci-lint"
    echo "  ✅ go.mod tidy check"
    echo "  ✅ Unit tests"
    echo "  ✅ Integration tests (if envtest available)"
    echo "  ✅ Standalone binary build"
    echo "  ✅ Operator binary build"
    echo "  ✅ Docker image builds (if Docker available)"
    echo "  ✅ Helm chart lint (if Helm available)"
    echo "  ✅ Helm template test (if Helm available)"
    echo "  ✅ Security scan (if Trivy available)"
    echo ""
    exit 0
else
    print_error "CI TESTS FAILED: $FAILURES failure(s)"
    echo ""
    echo "Please fix the failures above before pushing to GitHub."
    echo ""
    exit 1
fi
