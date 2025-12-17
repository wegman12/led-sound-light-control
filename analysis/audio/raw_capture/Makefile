# Raw ADC Capture Utility Makefile

BINARY := raw_capture
BINARY_ARM := raw_capture_arm

.PHONY: all build build-arm clean deploy test

all: build

# Build for local machine (amd64)
build:
	@echo "Building for local machine..."
	go build -o $(BINARY) .
	@echo "Build complete: $(BINARY)"

# Build for BeagleBone Black (ARM v7)
build-arm:
	@echo "Building for BeagleBone Black (ARM v7)..."
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY_ARM) .
	@echo "Build complete: $(BINARY_ARM)"

# Deploy to BeagleBone
deploy: build-arm
	@echo "Deploying to BeagleBone..."
	@if [ -z "$(HOST)" ]; then \
		echo "ERROR: HOST not specified. Usage: make deploy HOST=bbb2.wegman"; \
		exit 1; \
	fi
	scp $(BINARY_ARM) $(HOST):/tmp/raw_capture
	@echo "Deployed to $(HOST):/tmp/raw_capture"
	@echo ""
	@echo "To use on BeagleBone:"
	@echo "  ssh $(HOST)"
	@echo "  sudo /tmp/raw_capture 10 /tmp/baseline.bin"

# Test locally (requires PRU hardware)
test: build
	@echo "Testing requires root and PRU hardware"
	sudo ./$(BINARY) 5 test_output.bin

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY) $(BINARY_ARM)
	@echo "Clean complete"
