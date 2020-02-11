RM_F := rm -rf
GO := go
GIT := git

Additional_Linters := misspell,nakedret,goimports,misspell,unparam,unconvert,bodyclose,govet

export GO111MODULE=on

BIN := modden
SRC := $(BIN).tgz

.PHONY: help
help:
	@echo "$(BIN)"
	@echo
	@echo Targets:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9._-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: ## Build
	$(GO) build -o $(BIN) .

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
	@docker run \
		--rm \
		--volume $$(pwd):/app \
		--workdir /app \
		--env GO111MODULE \
		golangci/golangci-lint:v1.21.0 \
		golangci-lint run \
			--timeout 10m \
			--enable $(Additional_Linters)

.PHONY: clean
clean: ## Remove output files
	$(RM_F) $(BIN) $(SRC)
	$(GO) clean ./...

.PHONY: archive
archive: ## Create a source archive
archive: $(SRC)
$(SRC):
	$(GIT) archive --prefix=$(BIN)/ --format=tgz -o $@ HEAD
