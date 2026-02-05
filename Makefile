.PHONY: help build test lint docker docker-push release install deploy clean

# Version variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Docker variables
DOCKER_REGISTRY := ghcr.io
DOCKER_REPO := $(DOCKER_REGISTRY)/0xvox/traefik-officer
IMAGE_STANDALONE := $(DOCKER_REPO):$(VERSION)
IMAGE_OPERATOR := $(DOCKER_REPO)-operator:$(VERSION)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Development
build: ## Build binaries
	@echo "Building standalone binary..."
	cd cmd/traefik-officer && $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../../bin/traefik-officer .
	@echo "Building operator binary..."
	cd operator && $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../bin/traefik-officer-operator .
	@echo "Binaries built successfully!"

build-standalone: ## Build standalone binary only
	cd cmd/traefik-officer && $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../../bin/traefik-officer .

build-operator: ## Build operator binary only
	cd operator && $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../bin/traefik-officer-operator .

test: ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## Run tests with coverage report
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html

lint: ## Run linters
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Running golangci-lint..."
	golangci-lint run --timeout=10m

fmt: ## Format code
	$(GOCMD) fmt ./...

tidy: ## Tidy go.mod
	$(GOMOD) tidy
	cd operator && $(GOMOD) tidy

## Docker
docker: ## Build Docker images
	docker build -f Dockerfile -t $(IMAGE_STANDALONE) .
	docker build -f operator/Dockerfile -t $(IMAGE_OPERATOR) .

docker-standalone: ## Build standalone Docker image
	docker build -f Dockerfile -t $(IMAGE_STANDALONE) .

docker-operator: ## Build operator Docker image
	docker build -f operator/Dockerfile -t $(IMAGE_OPERATOR) .

docker-push: ## Push Docker images to registry
	docker push $(IMAGE_STANDALONE)
	docker push $(IMAGE_OPERATOR)

## Kubernetes
install-crds: ## Install CRDs
	kubectl apply -f operator/crd/bases/

uninstall-crds: ## Uninstall CRDs
	kubectl delete -f operator/crd/bases/

install-operator: ## Install operator via Helm
	helm install traefik-officer-operator ./helm/traefik-officer-operator

upgrade-operator: ## Upgrade operator via Helm
	helm upgrade traefik-officer-operator ./helm/traefik-officer-operator

uninstall-operator: ## Uninstall operator
	helm uninstall traefik-officer-operator

deploy-examples: ## Deploy example UrlPerformance resources
	kubectl apply -f examples/urlperformances/

## Release
release: ## Create a new release (requires VERSION)
	@echo "Creating release $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

release-dry-run: ## Test release with goreleaser (dry-run)
	goreleaser release --skip-publish --clean

chart-package: ## Package Helm chart
	helm package helm/traefik-officer-operator

chart-lint: ## Lint Helm chart
	helm lint helm/traefik-officer-operator

## Clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.txt coverage.html
	docker system prune -f

## Development helpers
run-operator: ## Run operator locally (requires kubectl config)
	cd operator && $(GOBUILD) -o ../bin/traefik-officer-operator . && \
		../bin/traefik-officer-operator --leader-elect=false

run-standalone: ## Run standalone mode
	cd cmd/traefik-officer && $(GOBUILD) -o ../../bin/traefik-officer . && \
		../../bin/traefik-officer --log-file=test/access.log --json-logs

deps: ## Download dependencies
	$(GOMOD) download
	cd operator && $(GOMOD) download
	cd cmd/traefik-officer && $(GOMOD) download

## CI/CD
ci: lint test ## Run CI checks locally

all: build ## Build all binaries
