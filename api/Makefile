# Binary name
BINARY_NAME=led-sound-light-control

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build directory
BUILD_DIR=bin

# Main package path
MAIN_PATH=.

# Deployment configuration
DEPLOY_HOST=bbb1.wegman
DEPLOY_PATH=~/led-sound-light-control/
BBB_BINARY=$(BINARY_NAME)-bbb

# PRU Configuration
PRU_DIR=../pru
PRU_FIRMWARE=$(PRU_DIR)/gen/pru0_ir_detector.out
PRU_INSTALL_PATH=/lib/firmware/am335x-pru0-fw

.PHONY: all build test clean run install deps fmt vet lint build-bbb deploy help build-pru clean-pru deploy-pru deploy-all build-pru-utils deploy-pru-utils

# Default target
all: deps fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean: clean-pru
	@echo "Cleaning Go artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(MAIN_PATH)

# Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(MAIN_PATH)
	@echo "Install complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
	fi

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Multi-platform build complete"

# Build for BeagleBone Black (ARMv7)
build-bbb:
	@echo "Building for BeagleBone Black (ARM v7)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/$(BBB_BINARY) $(MAIN_PATH)
	@echo "BeagleBone Black build complete: $(BUILD_DIR)/$(BBB_BINARY)"

# Deploy to BeagleBone Black (Go binary only)
deploy: build-bbb
	@echo "Deploying Go binary to $(DEPLOY_HOST):$(DEPLOY_PATH)..."
	@echo "Stopping service..."
	@ssh $(DEPLOY_HOST) "sudo systemctl stop led-sound-light-control.service 2>/dev/null || true"
	@echo "Copying binary..."
	@scp $(BUILD_DIR)/$(BBB_BINARY) $(DEPLOY_HOST):$(DEPLOY_PATH)$(BINARY_NAME)
	@echo "Starting service..."
	@ssh $(DEPLOY_HOST) "sudo systemctl start led-sound-light-control.service"
	@echo "Go binary deployment complete"

# Build PRU firmware (local build - requires TI PRU compiler)
build-pru:
	@echo "Building PRU firmware locally..."
	@if ! command -v clpru > /dev/null 2>&1; then \
		echo "ERROR: TI PRU compiler (clpru) not found."; \
		echo ""; \
		echo "Options:"; \
		echo "  1. Use 'make deploy-pru' to build on BeagleBone (recommended)"; \
		echo "  2. Install TI PRU compiler locally:"; \
		echo "     - Download from: https://www.ti.com/tool/PRU-CGT"; \
		echo "     - Or on Debian: sudo apt-get install ti-pru-cgt-installer"; \
		exit 1; \
	fi
	$(MAKE) -C $(PRU_DIR)
	@echo "PRU firmware build complete: $(PRU_FIRMWARE)"

# Clean PRU build artifacts
clean-pru:
	@echo "Cleaning PRU artifacts..."
	@$(MAKE) -C $(PRU_DIR) clean 2>/dev/null || true

# Deploy PRU firmware to BeagleBone (builds on target)
deploy-pru:
	@echo "Deploying PRU firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p ~/led-sound-light-control/pru"
	@scp -r $(PRU_DIR)/*.c $(PRU_DIR)/*.h $(PRU_DIR)/*.cmd $(PRU_DIR)/Makefile $(DEPLOY_HOST):~/led-sound-light-control/pru/
	@echo "Building PRU firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd ~/led-sound-light-control/pru && make"
	@echo "Installing and loading firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp ~/led-sound-light-control/pru/gen/pru0_ir_detector.out $(PRU_INSTALL_PATH) && \
	                     echo 'Stopping PRU (if running)...' && \
	                     (echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state > /dev/null 2>&1 || true) && \
	                     sleep 1 && \
	                     echo 'Starting PRU...' && \
	                     echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state > /dev/null && \
	                     sleep 1 && \
	                     echo 'Verifying PRU state...' && \
	                     cat /sys/class/remoteproc/remoteproc1/state"
	@echo "PRU firmware deployed and loaded successfully"

# Build PRU debug utilities for BeagleBone
build-pru-utils:
	@echo "Building PRU utilities for BeagleBone Black (ARM v7)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/pru-debug-bbb cmd/pru-debug.go
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/pru-timing-reader-bbb cmd/pru-timing-reader.go
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/pru-gpio-reader-bbb cmd/pru-gpio-reader.go
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BUILD_DIR)/pru-bits-reader-bbb cmd/pru-bits-reader.go
	@echo "PRU utilities build complete"

# Deploy PRU debug utilities to BeagleBone
deploy-pru-utils: build-pru-utils
	@echo "Deploying PRU utilities to $(DEPLOY_HOST)..."
	@scp $(BUILD_DIR)/pru-debug-bbb $(DEPLOY_HOST):$(DEPLOY_PATH)pru-debug
	@scp $(BUILD_DIR)/pru-timing-reader-bbb $(DEPLOY_HOST):$(DEPLOY_PATH)pru-timing-reader
	@scp $(BUILD_DIR)/pru-gpio-reader-bbb $(DEPLOY_HOST):$(DEPLOY_PATH)pru-gpio-reader
	@scp $(BUILD_DIR)/pru-bits-reader-bbb $(DEPLOY_HOST):$(DEPLOY_PATH)pru-bits-reader
	@echo "PRU utilities deployed successfully"
	@echo "On BeagleBone, run:"
	@echo "  sudo ~/led-sound-light-control/pru-debug         - Monitor PRU status"
	@echo "  sudo ~/led-sound-light-control/pru-timing-reader - Read timing capture data"
	@echo "  sudo ~/led-sound-light-control/pru-gpio-reader   - Read raw GPIO samples"
	@echo "  sudo ~/led-sound-light-control/pru-bits-reader   - Read captured IR bits"

# Deploy everything (PRU firmware + Go binary)
deploy-all: deploy-pru deploy
	@echo ""
	@echo "==================================="
	@echo "Full deployment complete!"
	@echo "==================================="
	@echo "PRU firmware: $(PRU_FIRMWARE) -> $(PRU_INSTALL_PATH)"
	@echo "Go binary: $(BUILD_DIR)/$(BBB_BINARY) -> $(DEPLOY_PATH)$(BINARY_NAME)"
	@echo ""
	@echo "To start the application on BeagleBone:"
	@echo "  ssh $(DEPLOY_HOST)"
	@echo "  cd $(DEPLOY_PATH)"
	@echo "  sudo ./$(BINARY_NAME) serve"

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Go Application:"
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Remove all build artifacts (Go + PRU)"
	@echo "  make run            - Run the application"
	@echo "  make install        - Install binary to \$$GOPATH/bin"
	@echo "  make deps           - Download and tidy dependencies"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run golangci-lint (if installed)"
	@echo "  make build-all      - Build for multiple platforms"
	@echo ""
	@echo "BeagleBone Black Deployment:"
	@echo "  make build-bbb      - Build for BeagleBone Black (ARM v7)"
	@echo "  make deploy         - Build and deploy Go binary only"
	@echo ""
	@echo "PRU Firmware:"
	@echo "  make build-pru      - Build PRU firmware locally (requires clpru)"
	@echo "  make clean-pru      - Remove PRU build artifacts"
	@echo "  make deploy-pru     - Copy source to BeagleBone, build and load firmware"
	@echo "  make deploy-all     - Deploy both PRU firmware and Go binary (recommended)"
	@echo ""
	@echo "PRU Debug Utilities:"
	@echo "  make build-pru-utils  - Build PRU debug utilities for BeagleBone"
	@echo "  make deploy-pru-utils - Build and deploy PRU debug utilities"
	@echo ""
	@echo "Combo:"
	@echo "  make all            - Run deps, fmt, vet, test, and build"
	@echo "  make help           - Display this help message"
