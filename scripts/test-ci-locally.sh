#!/bin/bash
# Test GitHub Actions CI locally
# This script runs all the same checks that the CI workflow does

set -e

echo "🚀 Starting CI Tests Locally..."
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
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

# Test 1: Check go.mod
print_info "Testing go.mod tidy..."
go mod tidy
cd operator && go mod tidy && cd ..
if git diff --exit-code go.mod go.sum operator/go.mod operator/go.sum > /dev/null 2>&1; then
    print_success "go.mod is tidy"
else
    print_error "go.mod is not tidy. Please run 'go mod tidy' and 'cd operator && go mod tidy'"
    exit 1
fi

# Test 2: Run golangci-lint
print_info "Running golangci-lint..."
if golangci-lint run --timeout=10m; then
    print_success "golangci-lint passed"
else
    print_error "golangci-lint failed"
    exit 1
fi

# Test 3: Run unit tests
print_info "Running unit tests..."
if go test -v -race ./pkg/... ./shared/...; then
    print_success "Unit tests passed"
else
    print_error "Unit tests failed"
    exit 1
fi

# Test 4: Setup envtest
print_info "Setting up envtest..."
if command -v setup-envtest &> /dev/null; then
    export PATH=$HOME/go/bin:$PATH
    KUBEBUILDER_ASSETS=$(setup-envtest use 1.30.0 -p path)
    export KUBEBUILDER_ASSETS
    print_success "envtest setup complete"
else
    print_info "setup-envtest not found, skipping integration tests"
    KUBEBUILDER_ASSETS=""
fi

# Test 5: Run integration tests (if envtest is available)
if [ -n "$KUBEBUILDER_ASSETS" ]; then
    print_info "Running integration tests..."
    if go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        exit 1
    fi
fi

# Test 6: Build standalone binary
print_info "Building standalone binary..."
if go build -o bin/traefik-officer ./cmd/traefik-officer; then
    print_success "Standalone binary built successfully"
    rm -f bin/traefik-officer
else
    print_error "Failed to build standalone binary"
    exit 1
fi

# Test 7: Build operator binary
print_info "Building operator binary..."
if go build -o bin/traefik-officer-operator ./cmd/operator; then
    print_success "Operator binary built successfully"
    rm -f bin/traefik-officer-operator
else
    print_error "Failed to build operator binary"
    exit 1
fi

# Summary
echo ""
print_success "🎉 All CI tests passed locally!"
echo ""
echo "Summary of tests passed:"
echo "  ✅ go.mod tidy check"
echo "  ✅ golangci-lint"
echo "  ✅ Unit tests"
echo "  ✅ Integration tests (if envtest available)"
echo "  ✅ Standalone binary build"
echo "  ✅ Operator binary build"
echo ""
echo "You can now push to GitHub with confidence!"
