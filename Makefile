.PHONY: all test test-race lint vet fmt ci clean tidy

# Default target runs all checks.
all: ci

# Run tests.
test:
go test -v -count=1 ./...

# Run tests with race detector.
test-race:
go test -v -race -count=1 ./...

# Run golangci-lint if available, otherwise fall back to go vet.
lint:
@if command -v golangci-lint >/dev/null 2>&1; then \
golangci-lint run ./...; \
else \
echo "golangci-lint not found, running go vet instead"; \
go vet ./...; \
fi

# Run go vet.
vet:
go vet ./...

# Check formatting.
fmt:
@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

# Tidy dependencies.
tidy:
go mod tidy

# Run all CI checks.
ci: fmt vet test-race

# Generate test coverage report.
coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
@echo "Coverage report: coverage.html"

# Clean build artifacts.
clean:
rm -f coverage.out coverage.html
go clean -testcache
