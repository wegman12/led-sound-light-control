# Raw I2S Capture Utility Makefile

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean

# Binary names
BINARY := raw_i2s_capture
BINARY_ARM := raw_i2s_capture_arm

# Deployment configuration
DEPLOY_HOST := bbb1.wegman
DEPLOY_PATH := ~/led-sound-light-control/

.PHONY: all clean build build-arm deploy

# Default target: build for ARM
all: build-arm

# Build for local testing (amd64)
build:
	@echo "Building raw_i2s_capture for local testing..."
	$(GOBUILD) -o $(BINARY) .
	@echo "Built: $(BINARY)"

# Build for BeagleBone Black (ARM v7)
build-arm:
	@echo "Building raw_i2s_capture for BeagleBone Black..."
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BINARY_ARM) .
	@echo "Built: $(BINARY_ARM)"

# Deploy to BeagleBone
deploy: build-arm
	@echo "Deploying raw_i2s_capture to $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp $(BINARY_ARM) $(DEPLOY_HOST):$(DEPLOY_PATH)$(BINARY)
	@echo "Deployed to $(DEPLOY_HOST):$(DEPLOY_PATH)$(BINARY)"

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY) $(BINARY_ARM)
