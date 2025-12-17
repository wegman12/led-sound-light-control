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

.PHONY: all compile compile-remote install deploy deploy-full deploy-uenv clean help

all: compile-remote

# Compile device tree overlay locally (requires dtc installed)
compile: $(DTBO_FILE)

$(DTBO_FILE): $(DTS_FILE)
	@echo "Compiling device tree overlay: $(DTS_FILE) -> $(DTBO_FILE)"
	$(DTC) -O dtb -o $(DTBO_FILE) -b 0 -@ $(DTS_FILE)
	@echo "✓ Compilation complete"

# Compile device tree overlay on remote BeagleBone (recommended)
compile-remote:
	@echo "Compiling on $(DEPLOY_HOST)..."
	scp $(DTS_FILE) $(DEPLOY_HOST):~/
	ssh $(DEPLOY_HOST) "dtc -O dtb -o ~/$(DTBO_FILE) -b 0 -@ ~/$(DTS_FILE)"
	scp $(DEPLOY_HOST):~/$(DTBO_FILE) ./
	@echo "✓ Remote compilation complete"

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

# Deploy compiled overlay to remote BeagleBone (requires local .dtbo file)
deploy:
	@echo "Deploying to $(DEPLOY_HOST)..."
	@if [ ! -f $(DTBO_FILE) ]; then \
		echo "Error: $(DTBO_FILE) not found. Run 'make compile-remote' first."; \
		exit 1; \
	fi
	scp $(DTBO_FILE) $(DEPLOY_HOST):/tmp/
	ssh $(DEPLOY_HOST) "sudo cp /tmp/$(DTBO_FILE) $(FIRMWARE_DIR)/"
	ssh $(DEPLOY_HOST) "sudo cp /tmp/$(DTBO_FILE) $(OVERLAY_PATH)"
	@echo "✓ Deployed to $(DEPLOY_HOST)"
	@echo "  - $(FIRMWARE_DIR)/$(DTBO_FILE)"
	@echo "  - $(OVERLAY_PATH)$(DTBO_FILE)"
	@echo ""
	@echo "Overlay will load automatically on next boot via uEnv.txt"
	@echo "To reboot: ssh $(DEPLOY_HOST) 'sudo reboot'"

# Compile and deploy in one step (recommended)
deploy-full: compile-remote deploy
	@echo ""
	@echo "✓ Full deployment complete!"
	@echo "Reboot BeagleBone to load the updated overlay"

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
	@echo "  compile-remote - Compile device tree on remote BeagleBone (default, recommended)"
	@echo "  compile        - Compile device tree locally (requires dtc installed)"
	@echo "  deploy         - Deploy compiled overlay to BeagleBone"
	@echo "  deploy-full    - Compile remotely and deploy in one step"
	@echo "  deploy-uenv    - Deploy uEnv.txt to BeagleBone (USE WITH CAUTION, requires reboot)"
	@echo "  install        - Install overlay locally (run on BeagleBone)"
	@echo "  clean          - Remove compiled files"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make                                    # Compile remotely"
	@echo "  make deploy-full                        # Compile and deploy"
	@echo "  make deploy-full DEPLOY_HOST=bbb1.wegman  # Deploy to specific host"
	@echo "  make deploy-uenv DEPLOY_HOST=bbb2.wegman  # Deploy uEnv.txt"
	@echo ""
	@echo "Current settings:"
	@echo "  DEPLOY_HOST = $(DEPLOY_HOST)"
	@echo "  KERNEL_VERSION = $(KERNEL_VERSION)"
