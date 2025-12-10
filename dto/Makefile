# Device Tree Overlay Makefile
DTC := dtc
SRC := custom-pins.dts
DTBO := custom-pins.dtbo
DEPLOY_HOST := bbb1.wegman
# Get kernel version from BeagleBone and construct overlay path
KERNEL_VERSION := $(shell ssh $(DEPLOY_HOST) "uname -r" 2>/dev/null || echo "5.10.168-ti-r83")
OVERLAY_PATH := /boot/dtbs/$(KERNEL_VERSION)/overlays/

.PHONY: all build deploy deploy-uenv clean help

all: build

# Build the device tree overlay
build: $(DTBO)

$(DTBO): $(SRC)
	@echo "Compiling device tree overlay..."
	$(DTC) -O dtb -o $(DTBO) -b 0 -@ $(SRC)
	@echo "Build complete: $(DTBO)"

# Deploy to BeagleBone Black (build on target)
deploy:
	@echo "Copying source file to $(DEPLOY_HOST)..."
	@scp $(SRC) $(DEPLOY_HOST):~/
	@echo "Building on BeagleBone..."
	@ssh $(DEPLOY_HOST) "dtc -O dtb -o ~/$(DTBO) -b 0 -@ ~/$(SRC)"
	@echo "Installing to $(OVERLAY_PATH)..."
	@ssh $(DEPLOY_HOST) "sudo cp ~/$(DTBO) $(OVERLAY_PATH)"
	@echo "Updating /boot/uEnv.txt to reference $(DTBO)..."
	@ssh $(DEPLOY_HOST) "sudo sed -i 's|uboot_overlay_addr4=.*|uboot_overlay_addr4=$(DTBO)|' /boot/uEnv.txt"
	@echo ""
	@echo "Device tree overlay deployed successfully!"
	@echo ""
	@echo "IMPORTANT: You must REBOOT the BeagleBone for the changes to take effect:"
	@echo "  ssh $(DEPLOY_HOST) 'sudo reboot'"
	@echo ""
	@echo "After reboot, verify the overlay loaded:"
	@echo "  ssh $(DEPLOY_HOST) 'dmesg | grep custom-pins'"

# Deploy uEnv.txt to BeagleBone (USE WITH CAUTION)
deploy-uenv:
	@echo "WARNING: This will replace /boot/uEnv.txt on $(DEPLOY_HOST)"
	@echo "Make sure you have reviewed the changes in uEnv.txt"
	@echo ""
	@echo "Backing up current uEnv.txt..."
	@ssh $(DEPLOY_HOST) "sudo cp /boot/uEnv.txt /boot/uEnv.txt.backup.$$(date +%Y%m%d_%H%M%S)"
	@echo "Deploying uEnv.txt..."
	@scp uEnv.txt $(DEPLOY_HOST):~/
	@ssh $(DEPLOY_HOST) "sudo cp ~/uEnv.txt /boot/uEnv.txt"
	@echo ""
	@echo "uEnv.txt deployed successfully!"
	@echo ""
	@echo "IMPORTANT: You must REBOOT the BeagleBone for changes to take effect:"
	@echo "  ssh $(DEPLOY_HOST) 'sudo reboot'"
	@echo ""
	@echo "Backup saved at: /boot/uEnv.txt.backup.<timestamp>"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(DTBO)
	@echo "Clean complete"

# Help target
help:
	@echo "Available targets:"
	@echo "  make build       - Compile the device tree overlay"
	@echo "  make deploy      - Build and deploy overlay to BeagleBone (requires reboot)"
	@echo "  make deploy-uenv - Deploy uEnv.txt to BeagleBone (USE WITH CAUTION, requires reboot)"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make help        - Show this help message"
