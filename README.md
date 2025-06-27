# Local AI Agent Framework

A modular, offline-first AI agent framework built in Go that combines task planning, browser automation, and stateful execution.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Agent Framework                        │
├─────────────────────────────────────────────────────────────────┤
│  CLI/REST API Layer                                             │
│  ┌─────────────┐  ┌─────────────┐                              │
│  │ CLI Client  │  │ REST Server │                              │
│  └─────────────┘  └─────────────┘                              │
├─────────────────────────────────────────────────────────────────┤
│  Core Agent Engine                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Planner   │  │Task Executor│  │LangGraph Eng│            │
│  │     🧠      │  │     🔁      │  │     🧩      │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
├─────────────────────────────────────────────────────────────────┤
│  Execution Layer                                                │
│  ┌─────────────┐  ┌─────────────┐                              │
│  │Browser Agent│  │Memory Store │                              │
│  │     🌐      │  │     💬      │                              │
│  └─────────────┘  └─────────────┘                              │
├─────────────────────────────────────────────────────────────────┤
│  External Dependencies (Local)                                  │
│  ┌─────────────┐  ┌─────────────┐                              │
│  │   Ollama    │  │ Playwright  │                              │
│  │ (LLM Local) │  │ (Browser)   │                              │
│  └─────────────┘  └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Minikube (for K8s deployment)
- Ollama running locally on port 11434

### Local Development
```bash
# Install dependencies
go mod tidy

# Run Ollama (in separate terminal)
ollama serve
ollama pull llama3

# Start the agent framework
go run cmd/agent/main.go

# Or use CLI
go run cmd/cli/main.go plan "Book a flight to NYC"
```

### Docker Deployment
```bash
# Build all services
docker-compose build

# Start the framework
docker-compose up -d
```

### Kubernetes Deployment
```bash
# Deploy to Minikube
kubectl apply -f deployments/k8s/
```

## 📦 Module Structure

### 1. 🧠 Planner (`pkg/planner`)
- Breaks down high-level goals into executable tasks
- Interfaces with local LLM via Ollama
- Generates task dependency graphs

### 2. 🔁 Task Executor (`pkg/executor`)
- Routes tasks to appropriate handlers
- Manages task lifecycle and status
- Supports pluggable task types

### 3. 🌐 Browser Agent (`pkg/browser`)
- Playwright-based browser automation
- Headless and headed modes
- Screenshot and interaction capabilities

### 4. 💬 Memory Store (`pkg/memory`)
- In-memory task state management
- Redis-compatible interface for scaling
- Task history and context preservation

### 5. 🧩 LangGraph Engine (`pkg/langgraph`)
- State machine for task workflows
- Event-driven state transitions
- Channel-based communication

## 🔧 Configuration

Environment variables:
- `OLLAMA_URL`: Ollama API endpoint (default: http://localhost:11434)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `BROWSER_HEADLESS`: Run browser in headless mode (true/false)
- `MEMORY_TYPE`: Memory backend (memory, redis)

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

## 📊 Monitoring

- Structured logging with logrus
- Metrics via Prometheus (optional)
- Health checks on `/health` endpoint
- Task execution tracing

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details
