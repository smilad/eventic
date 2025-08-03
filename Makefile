.PHONY: test build clean run-basic run-advanced install lint

# Build the package
build:
	go build -o bin/sse .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	go test -bench=. ./...

# Install dependencies
install:
	go mod download
	go mod tidy

# Lint the code
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run basic example
run-basic:
	cd examples/basic && go run main.go

# Run advanced example
run-advanced:
	cd examples/advanced && go run main.go

# Build examples
build-examples:
	cd examples/basic && go build -o ../../bin/basic-example main.go
	cd examples/advanced && go build -o ../../bin/advanced-example main.go

# Run all tests and checks
check: install lint test

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the package"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  bench          - Run benchmarks"
	@echo "  install        - Install dependencies"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  run-basic      - Run basic example"
	@echo "  run-advanced   - Run advanced example"
	@echo "  build-examples - Build example applications"
	@echo "  check          - Run all tests and checks"
	@echo "  help           - Show this help" 