# Gin Template — common dev tasks.
# Works on bash (Linux/macOS) and Git Bash on Windows. For pure PowerShell use the
# corresponding `go` commands directly.

APP_NAME ?= gin-template
PKG      := ./...
BIN_DIR  := bin
BINARY   := $(BIN_DIR)/$(APP_NAME)
MAIN_PKG := ./cmd/api

GO       := go

.PHONY: help tidy run dev build test cover lint fmt vet swag migrate-up migrate-down migrate-status migrate-create clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## Sync go.mod and download dependencies
	$(GO) mod tidy

run: ## Run the API directly
	$(GO) run $(MAIN_PKG)

dev: ## Run with hot reload (requires air)
	air

build: ## Compile binary into ./bin
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) $(MAIN_PKG)

test: ## Run unit tests
	$(GO) test -race -count=1 $(PKG)

cover: ## Run tests with coverage report
	$(GO) test -race -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint (must be installed separately)
	golangci-lint run

fmt: ## Format code
	$(GO) fmt $(PKG)

vet: ## Run go vet
	$(GO) vet $(PKG)

swag: ## Regenerate Swagger docs from annotations
	swag init -g cmd/api/main.go -o docs

migrate-up: ## Apply all up migrations
	goose -dir migrations $(DB_DRIVER) "$(DB_DSN)" up

migrate-down: ## Roll back the last migration
	goose -dir migrations $(DB_DRIVER) "$(DB_DSN)" down

migrate-status: ## Show migration status
	goose -dir migrations $(DB_DRIVER) "$(DB_DSN)" status

migrate-create: ## Create a new SQL migration: `make migrate-create name=add_orders`
	goose -dir migrations create $(name) sql

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) tmp coverage.out coverage.html
