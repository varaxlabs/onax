# Contributing to k8s-cronjob-monitor

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.22+
- Docker
- kind (for local testing)
- Helm 3.0+
- kubectl

### Getting Started

```bash
# Clone the repository
git clone https://github.com/kubeshield/k8s-cronjob-monitor.git
cd k8s-cronjob-monitor

# Install dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Build Docker image
make docker-build
```

### Local Testing

```bash
# Create a kind cluster with Prometheus
make kind-create

# Load the image into kind
kind load docker-image ghcr.io/kubeshield/k8s-cronjob-monitor:latest --name cronjob-monitor-test

# Deploy
make deploy

# Deploy test CronJobs
kubectl apply -f examples/basic-cronjob.yaml

# Check metrics
kubectl port-forward -n monitoring svc/k8s-cronjob-monitor 8080:8080
curl http://localhost:8080/metrics | grep cronjob_monitor
```

## Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Run linting: `make lint`
6. Commit your changes with a descriptive message
7. Push to your fork and open a Pull Request

## Code Style

- Follow standard Go conventions
- Run `go fmt` and `go vet` before committing
- Write tests for new functionality
- Maintain >80% test coverage

## Pull Request Guidelines

- Keep PRs focused on a single change
- Include tests for new features
- Update documentation if needed
- Ensure CI passes

## Reporting Issues

- Use the GitHub issue templates
- Include Kubernetes version, operator version, and relevant logs
- Provide steps to reproduce

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
