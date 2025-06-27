# AI Agent Framework Makefile

.PHONY: help build test clean docker-build docker-run k8s-deploy k8s-clean dev

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the Go binaries"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  k8s-deploy    - Deploy to Kubernetes"
	@echo "  k8s-clean     - Clean Kubernetes deployment"
	@echo "  dev           - Run in development mode"

# Build Go binaries
build:
	@echo "Building Go binaries..."
	go build -o bin/agent ./cmd/agent/main.go
	go build -o bin/cli ./cmd/cli/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	docker system prune -f

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t ai-agent-framework:latest .

# Run with Docker Compose
docker-run: docker-build
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started. Agent available at http://localhost:8080"
	@echo "Use 'docker-compose logs -f' to view logs"

# Stop Docker Compose
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

# Deploy to Kubernetes
k8s-deploy: docker-build
	@echo "Deploying to Kubernetes..."
	# Load image into Minikube (for local development)
	minikube image load ai-agent-framework:latest
	# Apply Kubernetes manifests
	kubectl apply -f deployments/k8s/namespace.yaml
	kubectl apply -f deployments/k8s/configmap.yaml
	kubectl apply -f deployments/k8s/ollama-deployment.yaml
	kubectl apply -f deployments/k8s/agent-deployment.yaml
	@echo "Deployment complete. Checking status..."
	kubectl get pods -n ai-agent

# Clean Kubernetes deployment
k8s-clean:
	@echo "Cleaning Kubernetes deployment..."
	kubectl delete -f deployments/k8s/ --ignore-not-found=true
	kubectl delete namespace ai-agent --ignore-not-found=true

# Development mode
dev:
	@echo "Starting in development mode..."
	@echo "Make sure Ollama is running on localhost:11434"
	go run ./cmd/agent/main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate mocks (if using mockery)
mocks:
	@echo "Generating mocks..."
	mockery --all --output=mocks

# Run CLI
cli-status:
	go run ./cmd/cli/main.go status

cli-plan:
	go run ./cmd/cli/main.go plan "$(GOAL)"

# Quick start for development
quickstart: deps build
	@echo "Quick start complete!"
	@echo "Run 'make dev' to start the agent"
	@echo "Or 'make docker-run' to start with Docker"
