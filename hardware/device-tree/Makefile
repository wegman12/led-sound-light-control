# Device Tree Overlay Makefile
# Compiles and deploys I2S microphone overlay to BeagleBone Black

DTC := dtc
OVERLAY := BB-I2S-MIC-00A0
DTS_FILE := $(OVERLAY).dts
DTBO_FILE := $(OVERLAY).dtbo

# Deployment settings
DEPLOY_HOST ?= bbb2.wegman
FIRMWARE_DIR := /lib/firmware
# Get kernel version from BeagleBone and construct overlay path
KERNEL_VERSION := $(shell ssh $(DEPLOY_HOST) "uname -r" 2>/dev/null || echo "5.10.168-ti-r83")
OVERLAY_PATH := /boot/dtbs/$(KERNEL_VERSION)/overlays/

.PHONY: all compile install deploy deploy-uenv clean help

all: compile

# Compile device tree overlay
compile: $(DTBO_FILE)

$(DTBO_FILE): $(DTS_FILE)
	@echo "Compiling device tree overlay: $(DTS_FILE) -> $(DTBO_FILE)"
	$(DTC) -O dtb -o $(DTBO_FILE) -b 0 -@ $(DTS_FILE)
	@echo "✓ Compilation complete"

# Install locally (run on BeagleBone)
install: $(DTBO_FILE)
	@echo "Installing overlay to $(FIRMWARE_DIR)..."
	sudo cp $(DTBO_FILE) $(FIRMWARE_DIR)/
	@echo "✓ Overlay installed"
	@echo ""
	@echo "To load the overlay:"
	@echo "  sudo sh -c \"echo '$(OVERLAY)' > /sys/devices/platform/bone_capemgr/slots\""
	@echo ""
	@echo "To load automatically on boot, add to /boot/uEnv.txt:"
	@echo "  uboot_overlay_addr4=/lib/firmware/$(DTBO_FILE)"

# Deploy to remote BeagleBone
deploy: compile
	@echo "Deploying to $(DEPLOY_HOST)..."
	scp $(DTBO_FILE) $(DEPLOY_HOST):/tmp/
	ssh $(DEPLOY_HOST) "sudo cp /tmp/$(DTBO_FILE) $(FIRMWARE_DIR)/"
	@echo "✓ Deployed to $(DEPLOY_HOST)"
	@echo ""
	@echo "To load on $(DEPLOY_HOST) (kernel 5.10+):"
	@echo "  ssh $(DEPLOY_HOST) \"sudo mkdir -p /sys/kernel/config/device-tree/overlays/BB-I2S-MIC\""
	@echo "  ssh $(DEPLOY_HOST) \"sudo sh -c 'cat /lib/firmware/$(DTBO_FILE) > /sys/kernel/config/device-tree/overlays/BB-I2S-MIC/dtbo'\""

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
	@echo "✓ uEnv.txt deployed successfully!"
	@echo ""
	@echo "IMPORTANT: You must REBOOT the BeagleBone for changes to take effect:"
	@echo "  ssh $(DEPLOY_HOST) 'sudo reboot'"
	@echo ""
	@echo "Backup saved at: /boot/uEnv.txt.backup.<timestamp>"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(DTBO_FILE)
	@echo "✓ Clean complete"

# Show help
help:
	@echo "Device Tree Overlay Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  compile      - Compile device tree overlay (default)"
	@echo "  install      - Install overlay locally (requires root)"
	@echo "  deploy       - Deploy to remote BeagleBone (DEPLOY_HOST=$(DEPLOY_HOST))"
	@echo "  deploy-uenv  - Deploy uEnv.txt to BeagleBone (USE WITH CAUTION, requires reboot)"
	@echo "  clean        - Remove compiled files"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make"
	@echo "  make deploy DEPLOY_HOST=bbb1.wegman"
	@echo "  make deploy-uenv DEPLOY_HOST=bbb2.wegman"
	@echo "  make install"
