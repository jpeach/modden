RM_F := rm -rf
GO := go
GIT := git

Additional_Linters := misspell,nakedret

export GO111MODULE=on

BIN := modden
SRC := $(BIN).tgz

.PHONY: help
help:
	@echo "$(BIN)"

.PHONY: build
build: ## Build
	$(GO) build -o $(BIN) ./cmd

.PHONY: check
check: ## Run tests
check: check-tests check-lint check-tidy
	$(GO) test -cover -v ./...

.PHONY: check-tests
check-tests: ## Run tests
	$(GO) test -cover -v ./...

.PHONY: check-tidy
check-tidy: ## Run linters
	$(GO) mod tidy

.PHONY: check-lint
check-lint: ## Run linters
	docker run \
		--rm \
		--volume $$(pwd):/app \
		--workdir /app \
		--env GO111MODULE \
		golangci/golangci-lint:v1.21.0 \
		golangci-lint --enable $(Additional_Linters) run

.PHONY: clean
clean: ## Remove output files
	$(RM_F) $(BIN) $(SRC)
	$(GO) clean ./...

.PHONY: archive
archive: $(SRC)
$(SRC):
	$(GIT) archive --prefix=$(BIN)/ --format=tgz -o $@ HEAD
