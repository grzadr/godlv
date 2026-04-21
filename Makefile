.PHONY: all test test-verbose test-coverage coverage-html lint build clean generate run

# Binary name and output directory
BINARY_NAME=godlv
BIN_DIR=bin
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out

# Default target
all: build

# Build the binary (depends on tests and linting passing)
build: clean generate lint test
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Binary created at $(BIN_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -cover -coverprofile=$(COVERAGE_FILE) ./...
	@echo ""
	@echo "Coverage by function:"
	@go tool cover -func=$(COVERAGE_FILE)

# Generate HTML coverage report and open in browser
coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

generate:
	go generate ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR) $(COVERAGE_DIR)
	@echo "Clean complete"

run: build
	$(BIN_DIR)/./$(BINARY_NAME) -h
