RM_F := rm -rf
GO := go
GIT := git

export GO111MODULE=on

BIN := modden
SRC := $(BIN).tgz

REPO := github.com/jpeach/modden
SHA := $(shell git rev-parse HEAD)
REVISION := $(shell git rev-parse --symbolic HEAD)

.PHONY: help
help:
	@echo "$(BIN)"
	@echo
	@echo Targets:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9._-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: ## Build
build: pkg/builtin/assets.go
	@$(GO) build \
		-ldflags "-X $(REPO)/pkg/version.Revision=$(REVISION)" \
		-ldflags "-X $(REPO)/pkg/version.Sha=$(SHA)" \
		-o $(BIN) .

install: ## Install
install: pkg/builtin/assets.go
	@$(GO) install \
		-ldflags "-X $(REPO)/pkg/version.Revision=$(REVISION)" \
		-ldflags "-X $(REPO)/pkg/version.Sha=$(SHA)" \
		.

pkg/builtin/assets.go: $(wildcard pkg/builtin/*.rego) $(wildcard pkg/builtin/*.yaml)
	./hack/go-bindata.sh -pkg builtin -o $@ $^

.PHONY: check
check: ## Run tests
check: check-tests check-lint check-tidy

.PHONY: check-tests
check-tests: ## Run tests
	@$(GO) test -cover -v ./...

.PHONY: check-tidy
check-tidy: ## Run linters
	@$(GO) mod tidy

.PHONY: check-lint
check-lint: ## Run linters
	@if command -v golangci-lint > /dev/null 2>&1 ; then \
		golangci-lint run --exclude-use-default=false ; \
	else \
		docker run \
			--rm \
			--volume $$(pwd):/app \
			--workdir /app \
			--env GO111MODULE \
			golangci/golangci-lint:v1.23.7 \
			golangci-lint run --exclude-use-default=false ; \
	fi

.PHONY: clean
clean: ## Remove output files
	$(RM_F) $(BIN) $(SRC)
	$(RM_F) pkg/builtin/assets.go
	$(GO) clean ./...

.PHONY: archive
archive: ## Create a source archive
archive: $(SRC)
$(SRC):
	$(GIT) archive --prefix=$(BIN)/ --format=tgz -o $@ HEAD
