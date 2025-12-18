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

# Driver settings
KERNEL_DRIVER_PATCH := ../../patches/0002-davinci-mcasp-add-debug-logging.patch
BUILD_DIR_REMOTE := ~/mcasp-debug-build
KERNEL_DIR_REMOTE := ~/repositories/external/linux_kernel/beagleboard

.PHONY: all compile compile-remote install deploy deploy-full deploy-uenv deploy-driver deploy-all clean help

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

# Build and deploy modified kernel driver
deploy-driver:
	@echo "======================================"
	@echo "Building Modified McASP Driver"
	@echo "with Debug Logging"
	@echo "======================================"
	@echo ""
	@# Check if patch file exists
	@if [ ! -f "$(KERNEL_DRIVER_PATCH)" ]; then \
		echo "ERROR: Patch file not found: $(KERNEL_DRIVER_PATCH)"; \
		exit 1; \
	fi
	@echo "[1/8] Copying patch to BeagleBone..."
	scp $(KERNEL_DRIVER_PATCH) $(DEPLOY_HOST):/tmp/mcasp-debug.patch
	@echo "[2/8] Creating build directory..."
	ssh $(DEPLOY_HOST) "mkdir -p $(BUILD_DIR_REMOTE)"
	@echo "[3/8] Copying stock driver source and headers from local kernel tree..."
	@# First reset the local kernel source to ensure we have stock driver
	cd $(KERNEL_DIR_REMOTE) && git checkout sound/soc/ti/davinci-mcasp.c
	@# Create directory structure and copy stock driver and headers
	ssh $(DEPLOY_HOST) "mkdir -p $(BUILD_DIR_REMOTE)/sound/soc/ti"
	scp $(KERNEL_DIR_REMOTE)/sound/soc/ti/davinci-mcasp.c $(DEPLOY_HOST):$(BUILD_DIR_REMOTE)/sound/soc/ti/
	scp $(KERNEL_DIR_REMOTE)/sound/soc/ti/*.h $(DEPLOY_HOST):$(BUILD_DIR_REMOTE)/sound/soc/ti/
	ssh $(DEPLOY_HOST) "ls -lh $(BUILD_DIR_REMOTE)/sound/soc/ti/ | head -10"
	@echo "[4/8] Applying debug patch..."
	ssh $(DEPLOY_HOST) "cd $(BUILD_DIR_REMOTE) && \
		patch -p1 < /tmp/mcasp-debug.patch && \
		echo 'Patch applied successfully'"
	@echo "[5/8] Creating module Makefile..."
	@ssh $(DEPLOY_HOST) "cd $(BUILD_DIR_REMOTE) && \
		printf 'obj-m += snd-soc-davinci-mcasp.o\n' > Makefile && \
		printf 'snd-soc-davinci-mcasp-objs := sound/soc/ti/davinci-mcasp.o\n\n' >> Makefile && \
		printf 'KDIR := /lib/modules/\$$(shell uname -r)/build\n\n' >> Makefile && \
		printf 'all:\n\tmake -C \$$(KDIR) M=\$$(PWD) modules\n\n' >> Makefile && \
		printf 'clean:\n\tmake -C \$$(KDIR) M=\$$(PWD) clean\n' >> Makefile"
	@echo "[6/8] Building kernel module..."
	ssh $(DEPLOY_HOST) "cd $(BUILD_DIR_REMOTE) && make && ls -lh *.ko"
	@echo "[7/8] Installing kernel module..."
	ssh $(DEPLOY_HOST) "cd $(BUILD_DIR_REMOTE) && \
		sudo cp snd-soc-davinci-mcasp.ko /lib/modules/\$$(uname -r)/kernel/sound/soc/ti/ && \
		sudo depmod -a && \
		echo 'Module installed successfully'"
	@echo "[8/8] Removing compressed backup to force uncompressed load..."
	ssh $(DEPLOY_HOST) "sudo rm -f /lib/modules/\$$(uname -r)/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz"
	@echo ""
	@echo "======================================"
	@echo "Driver built and installed!"
	@echo "======================================"
	@echo ""
	@echo "Next: Reboot to load the modified driver"
	@echo "  ssh $(DEPLOY_HOST) 'sudo reboot'"

# Deploy both device tree and driver
deploy-all: deploy-full deploy-driver
	@echo ""
	@echo "======================================"
	@echo "Complete System Deployment Finished!"
	@echo "======================================"
	@echo ""
	@echo "IMPORTANT: Reboot BeagleBone to activate changes"
	@echo "  ssh $(DEPLOY_HOST) 'sudo reboot'"
	@echo ""
	@echo "After reboot, test with:"
	@echo "  1. Enable GPIO1_27: ssh $(DEPLOY_HOST) 'echo 59 | sudo tee /sys/class/gpio/export && sudo sh -c \"echo 1 > /sys/class/gpio/gpio59/value\"'"
	@echo "  2. Load overlay: ssh $(DEPLOY_HOST) 'sudo sh -c \"mkdir -p /sys/kernel/config/device-tree/overlays/BB-I2S-MIC && cat /lib/firmware/$(DTBO_FILE) > /sys/kernel/config/device-tree/overlays/BB-I2S-MIC/dtbo\"'"
	@echo "  3. Test: ssh $(DEPLOY_HOST) 'arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 3 test.wav'"
	@echo "  4. Check: ssh $(DEPLOY_HOST) 'dmesg | grep -i mcasp | tail -30'"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(DTBO_FILE)
	ssh $(DEPLOY_HOST) "rm -rf $(BUILD_DIR_REMOTE)" 2>/dev/null || true
	@echo "✓ Clean complete"

# Show help
help:
	@echo "BeagleBone Black I2S Microphone - Build System"
	@echo ""
	@echo "Device Tree Targets:"
	@echo "  compile-remote - Compile device tree on remote BeagleBone (default, recommended)"
	@echo "  compile        - Compile device tree locally (requires dtc installed)"
	@echo "  deploy         - Deploy compiled overlay to BeagleBone"
	@echo "  deploy-full    - Compile remotely and deploy device tree"
	@echo "  deploy-uenv    - Deploy uEnv.txt to BeagleBone (USE WITH CAUTION, requires reboot)"
	@echo "  install        - Install overlay locally (run on BeagleBone)"
	@echo ""
	@echo "Kernel Driver Targets:"
	@echo "  deploy-driver  - Build and install modified McASP driver with debug logging"
	@echo "  deploy-all     - Deploy both device tree AND driver (complete system)"
	@echo ""
	@echo "Utility Targets:"
	@echo "  clean          - Remove compiled files and remote build artifacts"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make                                      # Compile device tree remotely"
	@echo "  make deploy-full                          # Compile and deploy device tree"
	@echo "  make deploy-driver                        # Build and install driver"
	@echo "  make deploy-all                           # Deploy EVERYTHING (recommended)"
	@echo "  make deploy-all DEPLOY_HOST=bbb1.wegman   # Deploy to specific host"
	@echo ""
	@echo "Current settings:"
	@echo "  DEPLOY_HOST = $(DEPLOY_HOST)"
	@echo "  KERNEL_VERSION = $(KERNEL_VERSION)"
	@echo "  DRIVER_PATCH = $(KERNEL_DRIVER_PATCH)"
