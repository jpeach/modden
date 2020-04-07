RM_F := rm -rf
GO := go
GIT := git

export GO111MODULE=on

BIN := modden
SRC := $(BIN).tgz

REPO := github.com/jpeach/modden
SHA := $(shell git rev-parse --short=8 HEAD)
VERSION := $(shell ./hack/tree-version.sh)
BUILDDATE := $(shell TZ=GMT date '+%Y-%m-%dT%R:%S%z')

GO_BUILD_LDFLAGS := \
	-s \
	-w \
	-X $(REPO)/pkg/version.Version=$(VERSION) \
	-X $(REPO)/pkg/version.Sha=$(SHA) \
	-X $(REPO)/pkg/version.BuildDate=$(BUILDDATE)

.PHONY: help
help:
	@echo "$(BIN)"
	@echo
	@echo Targets:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9._-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: ## Build
build: generate
	@$(GO) build -ldflags "$(GO_BUILD_LDFLAGS)" -o $(BIN) .

install: ## Install
install: generate
	@$(GO) install -ldflags "$(GO_BUILD_LDFLAGS)" .

.PHONY: generate
generate: ## Generate build files
generate: pkg/builtin/assets.go

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
	$(RM_F) $(BIN) $(SRC) dist
	$(RM_F) pkg/builtin/assets.go
	$(GO) clean ./...

.PHONY: archive
archive: ## Create a source archive
archive: $(SRC)
$(SRC):
	$(GIT) archive --prefix=$(BIN)/ --format=tgz -o $@ HEAD

.PHONY: release
release:
	# Check there is a token.
	[[ -n "$$GITLAB_TOKEN" ]] || [[ -r ~/.config/goreleaser/github_token ]]
	# Check we are on a tag.
	git describe --exact-match >/dev/null
	# Do a full dry-run.
	SHA=$(SHA) VERSION=$(VERSION) goreleaser release --rm-dist --skip-publish --skip-validate
	SHA=$(SHA) VERSION=$(VERSION) goreleaser release --rm-dist --skip-validate
