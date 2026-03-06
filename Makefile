# Image URL to use for building/pushing image targets
IMG ?= guidedtraffic/dex-operator:latest

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

# Go commands
GOCMD = go
GOTEST = $(GOCMD) test
GOFMT = gofmt

# Coverage directory
COVERAGE_DIR = coverage

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands like 'source' to be used
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	@echo "Formatting code..."
	$(GOFMT) -s -w .

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run linting.
	@echo "Running static analysis..."
	go vet ./...
	$(GOFMT) -l .
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))
	golangci-lint run --timeout=5m

.PHONY: cyclo
cyclo: ## Run cyclomatic complexity analysis.
	@echo "Running cyclomatic complexity analysis (threshold: $(CYCLO_THRESHOLD))..."
	@which gocyclo > /dev/null || (echo "Installing gocyclo..." && go install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION))
	@gocyclo -over $(CYCLO_THRESHOLD) -ignore "_test.go" . && echo "✅ All functions are below complexity threshold $(CYCLO_THRESHOLD)" || (echo "❌ Functions above complexity threshold $(CYCLO_THRESHOLD) found!" && gocyclo -over $(CYCLO_THRESHOLD) -ignore "_test.go" . && exit 1)

.PHONY: cyclo-report
cyclo-report: ## Show full cyclomatic complexity report (including tests).
	@echo "Cyclomatic complexity report (sorted by complexity):"
	@which gocyclo > /dev/null || (echo "Installing gocyclo..." && go install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION))
	@gocyclo -top 20 .

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: envtest ## Run unit tests only.
	@echo "Running unit tests..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -v -short ./...

.PHONY: test-unit-coverage
test-unit-coverage: envtest ## Run unit tests with coverage.
	@echo "Running unit tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -v -short -coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic ./...

.PHONY: test-integration
test-integration: envtest ## Run integration tests only.
	@echo "Running integration tests..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -v -tags=integration -count=1 -timeout=60m ./test/integration/...

.PHONY: test-integration-coverage
test-integration-coverage: envtest ## Run integration tests with coverage.
	@echo "Running integration tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -v -tags=integration -count=1 -timeout=60m -coverprofile=$(COVERAGE_DIR)/integration.out -covermode=atomic -coverpkg=./... ./test/integration/...

.PHONY: test-e2e
test-e2e: ## Run E2E tests against a running Kind cluster.
	@echo "Running E2E tests..."
	$(GOTEST) -v -tags=e2e -count=1 -timeout=30m ./test/e2e/...

.PHONY: test-e2e-helm
test-e2e-helm: build ## Run Helm migration E2E test (requires running Kind cluster with operator).
	@echo "Running Helm migration E2E test..."
	MANAGER_BINARY=./bin/manager $(GOTEST) -v -tags=e2e,e2e_helm -count=1 -timeout=10m -run TestE2E_Migrate ./test/e2e/...

.PHONY: kind-create
kind-create: ## Create a Kind cluster for local testing.
	@echo "Creating Kind cluster..."
	@echo 'kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nname: dex-operator-test\nnodes:\n- role: control-plane' | sed 's/\\n/\n/g' > /tmp/kind-config.yaml
	kind create cluster --config /tmp/kind-config.yaml --wait 120s

.PHONY: kind-delete
kind-delete: ## Delete the Kind test cluster.
	@echo "Deleting Kind cluster..."
	-kind delete cluster --name dex-operator-test

.PHONY: kind-load
kind-load: docker-build ## Build and load the operator image into Kind.
	@echo "Loading image into Kind cluster..."
	kind load docker-image ${IMG} --name dex-operator-test

.PHONY: cert-manager-install
cert-manager-install: ## Install cert-manager into the cluster.
	@echo "Installing cert-manager..."
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml
	@echo "Waiting for cert-manager to be ready..."
	kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s
	kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s
	kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=120s
	@echo "Creating self-signed ClusterIssuer..."
	@sleep 5
	@kubectl apply -f test/e2e/testdata/cert-manager-issuer.yaml

E2E_IMG ?= dex-operator:test

KUBECTL_IMG ?= guidedtraffic/kubectl:1.33

.PHONY: e2e-local
e2e-local: kind-create cert-manager-install ## Run full E2E test locally with Kind.
	@echo "Building E2E image..."
	docker build -f Containerfile -t $(E2E_IMG) .
	@echo "Pulling kubectl image for CRD upgrade hook..."
	docker pull $(KUBECTL_IMG)
	@echo "Loading images into Kind cluster..."
	kind load docker-image $(E2E_IMG) --name dex-operator-test
	kind load docker-image $(KUBECTL_IMG) --name dex-operator-test
	@echo "Installing operator via Helm (pre-install hook deploys CRDs)..."
	helm install dex-operator deploy/helm/dex-operator \
		--namespace dex-operator-system \
		--create-namespace \
		--values test/e2e/helm-values.yaml \
		--wait \
		--timeout 300s
	@echo "Running E2E tests..."
	$(MAKE) test-e2e
	@echo "Cleaning up..."
	$(MAKE) kind-delete

.PHONY: test-coverage
test-coverage: test ## Run tests and show coverage report.
	go tool cover -html=cover.out -o coverage.html

.PHONY: coverage
coverage: envtest ## Generate test coverage report.
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"
	@echo "Coverage summary:"
	@grep "total:" $(COVERAGE_DIR)/coverage.txt

.PHONY: coverage-ci
coverage-ci: envtest ## Generate CI coverage report.
	@echo "Generating CI coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt
	@grep "total:" $(COVERAGE_DIR)/coverage.txt
.PHONY: coverage-merge
coverage-merge: ## Merge unit and integration coverage profiles.
	@echo "Merging coverage profiles..."
	@mkdir -p $(COVERAGE_DIR)
	@# Install gocovmerge if not present
	@which gocovmerge > /dev/null || (echo "Installing gocovmerge..." && go install github.com/wadey/gocovmerge@latest)
	@# Merge coverage files
	gocovmerge $(COVERAGE_DIR)/unit.out $(COVERAGE_DIR)/integration.out > $(COVERAGE_DIR)/combined.out
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/combined.out > $(COVERAGE_DIR)/combined.txt
	@echo "Combined coverage:"
	@grep "total:" $(COVERAGE_DIR)/combined.txt

.PHONY: coverage-json
coverage-json: ## Generate coverage badge JSON for shields.io.
	@echo "Generating coverage badge JSON..."
	@mkdir -p .github/badges
	@COVERAGE=$$(grep "total:" $(COVERAGE_DIR)/combined.txt | awk '{print $$3}' | sed 's/%//'); \
	COLOR="red"; \
	if [ $$(echo "$$COVERAGE >= 80" | bc -l) -eq 1 ]; then COLOR="brightgreen"; \
	elif [ $$(echo "$$COVERAGE >= 60" | bc -l) -eq 1 ]; then COLOR="green"; \
	elif [ $$(echo "$$COVERAGE >= 40" | bc -l) -eq 1 ]; then COLOR="yellow"; \
	elif [ $$(echo "$$COVERAGE >= 20" | bc -l) -eq 1 ]; then COLOR="orange"; \
	fi; \
	echo "{\"schemaVersion\":1,\"label\":\"coverage\",\"message\":\"$$COVERAGE%\",\"color\":\"$$COLOR\"}" > .github/badges/coverage.json
	@echo "Badge JSON created at .github/badges/coverage.json"
	@cat .github/badges/coverage.json

##@ Security

.PHONY: gosec
gosec: ## Run gosec security scan.
	@echo "Running gosec security scan..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@v2.22.0)
	GOFLAGS="-buildvcs=false" gosec ./...

.PHONY: vuln
vuln: ## Check for vulnerabilities.
	@echo "Checking for vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	GOFLAGS="-buildvcs=false" govulncheck ./...

##@ Code Generation

HELM_CHART_DIR = deploy/helm/dex-operator
CRD_HELM_TARGET = $(HELM_CHART_DIR)/templates/crd.yaml

.PHONY: sync-helm-crd
sync-helm-crd: ## Sync generated CRDs into Helm chart (single source of truth: config/crd/bases/).
	@echo "Syncing CRDs to Helm chart..."
	@echo "# Generated - do not edit manually. Use 'make sync-helm-crd'." > $(CRD_HELM_TARGET)
	@for crd in config/crd/bases/*.yaml; do \
		if [ -f "$$crd" ]; then \
			echo "---" >> $(CRD_HELM_TARGET); \
			cat "$$crd" >> $(CRD_HELM_TARGET); \
		fi; \
	done
	@echo "CRDs synced to $(CRD_HELM_TARGET)"

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./cmd/main.go --zap-log-level=debug

.PHONY: generate-all
generate-all: manifests generate sync-helm-crd ## Regenerate CRD, DeepCopy code, and sync Helm chart. Run this after any api/v1/ type change.

.PHONY: docker-build
docker-build: generate-all ## Build docker image with the manager (regenerates CRD and DeepCopy first).
	docker build -f Containerfile -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for cross-platform support.
	- docker buildx create --name project-builder
	docker buildx use project-builder
	docker buildx build -f Containerfile --push --platform=linux/amd64,linux/arm64 --tag ${IMG} .
	docker buildx rm project-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | kubectl apply -f -

.PHONY: uninstall
uninstall: kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/rbac | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.3.0
CONTROLLER_GEN_VERSION ?= v0.17.3
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v2.1.6
GOCYCLO_VERSION ?= v0.6.0

# Cyclomatic complexity threshold (recommended: 10-15)
CYCLO_THRESHOLD ?= 15

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_GEN_VERSION))

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=dex-operator-role crd paths="./..." output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac
	$(MAKE) helm-sync-crds

HELM_CRD_DIR = deploy/helm/dex-operator/files/crds

.PHONY: helm-sync-crds
helm-sync-crds: ## Sync CRD manifests from config/crd/bases into the Helm chart's files/crds/ directory.
	@mkdir -p $(HELM_CRD_DIR)
	@cp config/crd/bases/*.yaml $(HELM_CRD_DIR)/
	@echo "Synced CRDs → $(HELM_CRD_DIR)"

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and target path
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
}
endef
