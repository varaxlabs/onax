# k8s-cronjob-monitor Makefile

# Image URL
IMG ?= ghcr.io/kubeshield/k8s-cronjob-monitor:latest

# Get the currently used golang install path
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

# setup-envtest binary
SETUP_ENVTEST ?= $(GOBIN)/setup-envtest
ENVTEST_K8S_VERSION ?= 1.30.0

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: fmt vet envtest ## Run tests
	KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/testbin -p path)" \
		go test ./... -coverprofile cover.out -race
	@echo "Coverage:"
	@go tool cover -func cover.out | tail -1

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	go tool cover -html=cover.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-integration
test-integration: fmt vet envtest ## Run integration tests only
	KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/testbin -p path)" \
		go test ./pkg/controller/... -coverprofile cover.out -race -v

##@ Build

.PHONY: build
build: fmt vet ## Build binary
	go build -o bin/operator cmd/operator/main.go

.PHONY: run
run: fmt vet ## Run operator locally
	go run cmd/operator/main.go

.PHONY: docker-build
docker-build: ## Build docker image
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image
	docker push ${IMG}

##@ Tool Dependencies

.PHONY: envtest
envtest: $(SETUP_ENVTEST) ## Download setup-envtest if necessary
$(SETUP_ENVTEST):
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

##@ Deployment

.PHONY: deploy
deploy: ## Deploy to cluster using Helm
	helm upgrade --install k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
		--namespace monitoring \
		--create-namespace

.PHONY: undeploy
undeploy: ## Remove from cluster
	helm uninstall k8s-cronjob-monitor --namespace monitoring

.PHONY: manifests
manifests: ## Generate raw YAML manifests from Helm
	helm template k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
		--namespace monitoring > deploy/manifests/operator.yaml

##@ Local Testing

.PHONY: kind-create
kind-create: ## Create kind cluster with monitoring stack
	./hack/test-setup.sh create

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	kind delete cluster --name cronjob-monitor-test

.PHONY: import-dashboards
import-dashboards: ## Import Grafana dashboards
	./hack/import-dashboards.sh

.PHONY: test-cronjobs
test-cronjobs: ## Deploy test CronJobs
	kubectl apply -f examples/
