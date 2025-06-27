# Makefile for New Relic Grafana Plugin

.PHONY: test coverage clean

# Default target
all: test

# Run tests without coverage
test:
	go test ./... -v

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test ./... -v -coverprofile=coverage.out
	@echo "\nFunction coverage summary:"
	@go tool cover -func=coverage.out
	@echo "\nGenerating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html

# Run tests with coverage and display report
coverage-html-report:
	@echo "Running tests with coverage..."
	@go test ./... -v -coverprofile=coverage.out
	@echo "\nFunction coverage summary:"
	@go tool cover -func=coverage.out
	@echo "\nGenerating and opening HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html && open coverage.html

# Clean up generated files
clean:
	rm -f coverage.out coverage.html