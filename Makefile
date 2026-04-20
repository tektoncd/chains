MODULE = $(shell env GO111MODULE=on $(GO) list -m)
DATE ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
	cat $(CURDIR)/.version 2> /dev/null || echo v0)
PKGS = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./... ))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
	'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
	$(PKGS))
BIN = $(CURDIR)/.bin

GOLANGCI_VERSION := $(shell yq '.jobs.linting.steps[] | select(.name == "golangci-lint") | .with.version' .github/workflows/ci.yaml)

GO = go
TIMEOUT_UNIT = 5m
TIMEOUT_E2E = 20m
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

export GO111MODULE=on

COMMANDS=$(patsubst cmd/%,%,$(wildcard cmd/*))
BINARIES=$(addprefix bin/,$(COMMANDS))

.PHONY: all
all: fmt $(BINARIES) | $(BIN) ; $(info $(M) building executable…) @ ## Build program binary

$(BIN):
	@mkdir -p $@

$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); \
	env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
	|| ret=$$?; \
	rm -rf $$tmp ; exit $$ret

FORCE:

bin/%: cmd/% FORCE
	$Q $(GO) build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

KO = $(or ${KO_BIN},${KO_BIN},$(BIN)/ko)
$(BIN)/ko: PACKAGE=github.com/google/ko

.PHONY: apply
apply: | $(KO) ; $(info $(M) ko apply -R -f config/) @ ## Apply config to the current cluster
	$Q $(KO) apply -R -f config

.PHONY: resolve
resolve: | $(KO) ; $(info $(M) ko resolve -R -f config/) @ ## Resolve config to the current cluster
	$Q $(KO) resolve --push=false --oci-layout-path=$(BIN)/oci -R -f config

.PHONY: generated
generated: | vendor ; $(info $(M) update generated files) ## Update generated files
	$Q ./hack/update-codegen.sh

.PHONY: vendor
vendor:
	$Q ./hack/update-deps.sh

## Tests

TEST_UNIT_TARGETS := test-unit-verbose test-unit-race test-unit-verbose-and-race
test-unit-verbose: ARGS=-v
test-unit-race: ARGS=-race
test-unit-verbose-and-race: ARGS=-v -race
$(TEST_UNIT_TARGETS): test-unit
.PHONY: $(TEST_UNIT_TARGETS) test-unit
test-unit: ## Run unit tests
	$(GO) test -timeout $(TIMEOUT_UNIT) $(ARGS) ./...

TEST_E2E_TARGETS := test-e2e-short test-e2e-verbose test-e2e-race
test-e2e-short: ARGS=-short
test-e2e-verbose: ARGS=-v
test-e2e-race: ARGS=-race
$(TEST_E2E_TARGETS): test-e2e
.PHONY: $(TEST_E2E_TARGETS) test-e2e
test-e2e: ## Run end-to-end tests
	$(GO) test -timeout $(TIMEOUT_E2E) -tags e2e $(ARGS) ./test/...

.PHONY: test-yamls
test-yamls: ## Run yaml tests
	./test/e2e-tests-yaml.sh --run-tests

.PHONY: check tests
check tests: test-unit test-e2e test-yamls

## Linters

GOLANGCILINT = $(BIN)/golangci-lint-$(GOLANGCI_VERSION)
$(GOLANGCILINT): ; $(info $(M) getting golangci-lint $(GOLANGCI_VERSION))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN) $(GOLANGCI_VERSION)
	mv $(BIN)/golangci-lint $(BIN)/golangci-lint-$(GOLANGCI_VERSION)

.PHONY: golangci-lint
golangci-lint: | $(GOLANGCILINT) ; $(info $(M) running golangci-lint…) @ ## Run golangci-lint
	$Q $(GOLANGCILINT) config verify
	$Q $(GOLANGCILINT) run --modules-download-mode=vendor --max-issues-per-linter=0 --max-same-issues=0 --timeout 5m

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: | $(BIN) ; $(info $(M) building goimports…)
	GOBIN=$(BIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: goimports
goimports: | $(GOIMPORTS) ; $(info $(M) running goimports…) ## Run goimports
	$Q $(GOIMPORTS) -l -e -w pkg cmd test

.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

.PHONY: yamllint
YAMLLINT := $(shell find . -path ./vendor -prune -o -type f -regex ".*y[a]ml" -print)
yamllint: | $(BIN) ; $(info $(M) running yamllint…) ## Run yamllint
	yamllint -c .yamllint $(YAMLLINT)

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…) @ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
