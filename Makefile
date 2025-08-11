# Makefile for AI Government Consultant

# Variables
BINARY_NAME=ai-government-consultant
DOCKER_IMAGE=ai-government-consultant
DOCKER_TAG=latest
GO_VERSION=1.23

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
.PHONY: dev
dev: ## Start development environment with hot reload
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile dev up --build

.PHONY: dev-down
dev-down: ## Stop development environment
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

.PHONY: dev-logs
dev-logs: ## Show development logs
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

# Build targets
.PHONY: build
build: ## Build the application binary
	go build -o bin/$(BINARY_NAME) ./cmd/server

.PHONY: build-docker
build-docker: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: build-dev-docker
build-dev-docker: ## Build development Docker image
	docker build -f Dockerfile.dev -t $(DOCKER_IMAGE):dev .

# Run targets
.PHONY: run
run: ## Run the application locally
	go run ./cmd/server

.PHONY: run-docker
run-docker: ## Run the application in Docker
	docker-compose up --build

# Test targets
.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-race
test-race: ## Run tests with race detection
	go test -v -race ./...

# Code quality targets
.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

# Database targets
.PHONY: db-up
db-up: ## Start database services only
	docker-compose up -d mongodb redis

.PHONY: db-down
db-down: ## Stop database services
	docker-compose down mongodb redis

.PHONY: db-reset
db-reset: ## Reset database (WARNING: This will delete all data)
	docker-compose down -v
	docker-compose up -d mongodb redis

# Utility targets
.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out coverage.html
	rm -f build-errors.log
	docker system prune -f

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod verify

.PHONY: update-deps
update-deps: ## Update dependencies
	go get -u ./...
	go mod tidy

# Production targets
.PHONY: prod-up
prod-up: ## Start production environment
	docker-compose up -d

.PHONY: prod-down
prod-down: ## Stop production environment
	docker-compose down

.PHONY: prod-logs
prod-logs: ## Show production logs
	docker-compose logs -f

# Security targets
.PHONY: security-scan
security-scan: ## Run security scan with gosec
	gosec ./...

.PHONY: vuln-check
vuln-check: ## Check for vulnerabilities
	go list -json -m all | nancy sleuth

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	godoc -http=:6060

# Install development tools
.PHONY: install-tools
install-tools: ## Install development tools
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest