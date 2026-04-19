.PHONY: help build run test clean docker-build docker-up docker-down migrate-up migrate-down migrate-status lint fmt docker-dev

# Default target
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start services with Docker Compose"
	@echo "  docker-down   - Stop services"
	@echo "  migrate-up    - Run database migrations up"
	@echo "  migrate-down  - Run database migrations down"
	@echo "  migrate-status - Show migration status"
	@echo "  docker-dev    - Start development environment"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"

# Variables
APP_NAME := devcost-ai
BUILD_DIR := build
DOCKER_IMAGE := $(APP_NAME):latest

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) cmd/server/main.go
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	@go run cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.html -o coverage.html coverage.out
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

# Start services with Docker Compose
docker-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

# Stop services
docker-down:
	@echo "Stopping services..."
	@docker-compose down

# Stop services and remove volumes
docker-clean:
	@echo "Stopping services and removing volumes..."
	@docker-compose down -v

# View logs
logs:
	@echo "Following logs..."
	@docker-compose logs -f

# Start development environment
docker-dev:
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Rebuild and start
docker-rebuild:
	@echo "Rebuilding and starting..."
	@docker-compose up -d --build

# Run database migrations up
migrate-up:
	@echo "Running database migrations up..."
	@go run cmd/migrate/main.go -action up -database "$(DB_HOST):$(DB_PORT)/$(DB_NAME)?user=$(DB_USER)&password=$(DB_PASSWORD)&sslmode=$(DB_SSLMODE)"

# Run database migrations down
migrate-down:
	@echo "Running database migrations down..."
	@go run cmd/migrate/main.go -action down -database "$(DB_HOST):$(DB_PORT)/$(DB_NAME)?user=$(DB_USER)&password=$(DB_PASSWORD)&sslmode=$(DB_SSLMODE)"

# Show migration status
migrate-status:
	@echo "Checking migration status..."
	@go run cmd/migrate/main.go -action version -database "$(DB_HOST):$(DB_PORT)/$(DB_NAME)?user=$(DB_USER)&password=$(DB_PASSWORD)&sslmode=$(DB_SSLMODE)"

# Force migration to specific version
migrate-force:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make migrate-force VERSION=<version>"; \
		exit 1; \
	fi
	@echo "Forcing migration to version $(VERSION)..."
	@go run cmd/migrate/main.go -action force -steps $(VERSION) -database "$(DB_HOST):$(DB_PORT)/$(DB_NAME)?user=$(DB_USER)&password=$(DB_PASSWORD)&sslmode=$(DB_SSLMODE)"

# Create new migration
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=<migration_name>"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)..."
	@migrate create -ext sql -dir migrations $(NAME)

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@cp .env.example .env
	@echo "Development environment ready!"

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@swag init -g cmd/server/main.go -o docs
	@swag generate

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed, installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Pre-commit checks
pre-commit: fmt lint test
	@echo "Running pre-commit checks..."
	@make fmt
	@make lint
	@make test

# All checks before commit
check: fmt lint security test-coverage
	@echo "Running all checks..."
	@make fmt
	@make lint
	@make security
	@make test-coverage
