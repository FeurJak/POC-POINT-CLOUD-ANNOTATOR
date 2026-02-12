.PHONY: all build test run clean docker-build docker-up docker-down help

# Default target
all: build

# Build the Go binary
build:
	@echo "Building Go binary..."
	@cd backend && go build -o ../bin/server ./cmd/main.go
	@echo "Build complete: bin/server"

# Run unit tests
test:
	@echo "Running tests..."
	@cd backend && go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@cd backend && go test ./... -coverprofile=coverage.out
	@cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

# Run locally (handler mode with local postgres/redis)
run-handler:
	@echo "Starting handler service..."
	DATABASE_URL=postgres://postgres:postgres@localhost:5432/annotations?sslmode=disable \
	REDIS_URL=redis://localhost:6379 \
	./bin/server -role handler -port 8081

# Run locally (gateway mode)
run-gateway:
	@echo "Starting gateway service..."
	HANDLER_URL=http://localhost:8081 \
	./bin/server -role gateway -port 8080

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker compose build

# Start all services
docker-up:
	@echo "Starting all services..."
	docker compose up -d
	@echo ""
	@echo "Services started:"
	@echo "  - Frontend:  http://localhost:3000"
	@echo "  - Gateway:   http://localhost:8080"
	@echo "  - Postgres:  localhost:5432"
	@echo "  - Redis:     localhost:6379"

# Stop all services
docker-down:
	@echo "Stopping all services..."
	docker compose down

# Stop and remove volumes
docker-clean:
	@echo "Stopping services and removing volumes..."
	docker compose down -v

# View logs
docker-logs:
	docker compose logs -f

# View logs for specific service
docker-logs-%:
	docker compose logs -f $*

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f backend/coverage.out backend/coverage.html
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@cd backend && go mod download
	@echo "Dependencies installed"

# Lint code
lint:
	@echo "Linting Go code..."
	@cd backend && go vet ./...
	@echo "Lint complete"

# Format code
fmt:
	@echo "Formatting Go code..."
	@cd backend && go fmt ./...
	@echo "Format complete"

# Help
help:
	@echo "Point Cloud Annotator - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build the Go binary"
	@echo "  make test           - Run unit tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Lint the Go code"
	@echo "  make fmt            - Format the Go code"
	@echo "  make deps           - Install Go dependencies"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Local Running (requires local postgres/redis):"
	@echo "  make run-handler    - Run the handler service locally"
	@echo "  make run-gateway    - Run the gateway service locally"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make docker-up      - Start all services"
	@echo "  make docker-down    - Stop all services"
	@echo "  make docker-clean   - Stop services and remove volumes"
	@echo "  make docker-logs    - View all service logs"
	@echo "  make docker-logs-X  - View logs for service X (e.g., docker-logs-handler)"
