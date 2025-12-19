# PRU IR Detector Firmware Makefile
# Builds firmware for BeagleBone Black PRU0

# Build directories
GEN_DIR := gen
SRC_DIR := .

# Deployment settings
DEPLOY_HOST ?= bbb2.wegman
REMOTE_DIR := ~/led-sound-light-control/pru
PRU_INSTALL_PATH := /lib/firmware/am335x-pru0-fw

# TI PRU Compiler and Linker
CC := clpru
LD := clpru

# Compiler flags - matching TI example (Ooff = optimization off for debugging)
CFLAGS := -v3 -Ooff --display_error_number --endian=little --hardware_mac=on \
          --obj_directory=$(GEN_DIR) --pp_directory=$(GEN_DIR) -ppd -ppa

# Linker flags
LFLAGS := --reread_libs --warn_sections --stack_size=0x100 --heap_size=0x100

# Library paths for linker
LIB_INCLUDE := -i/usr/share/ti/cgt-pru/lib -i/usr/share/ti/cgt-pru/include

# Include paths (TI PRU Software Support Package)
# Added PRU_SUPPORT/include for rsc_types.h needed by resource_table
INCLUDE := --include_path=$(SRC_DIR) \
           --include_path=/usr/share/ti/cgt-pru/include \
           --include_path=/usr/share/ti/cgt-pru/lib \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include/am335x

# Linker command file (using cookbook version)
LDFILE := am335x_pru.cmd

# Source files - IR Detector with resource table
SOURCE := pru0_ir_detector.c
OBJECT := $(GEN_DIR)/pru0_ir_detector.obj

# Output firmware
TARGET := $(GEN_DIR)/pru0_ir_detector.out

# Test firmware
TEST_SOURCE := test_r31.c
TEST_OBJECT := $(GEN_DIR)/test_r31.obj
TEST_TARGET := $(GEN_DIR)/test_r31.out

# Phony targets
.PHONY: all clean install help test deploy

# Default target
all: $(GEN_DIR) $(TARGET)
	@echo "PRU firmware build complete: $(TARGET)"

# Test target
test: $(GEN_DIR) $(TEST_TARGET)
	@echo "Test firmware build complete: $(TEST_TARGET)"

# Create output directory
$(GEN_DIR):
	@mkdir -p $(GEN_DIR)

# Compile C source to object file
$(OBJECT): $(SOURCE) | $(GEN_DIR)
	@echo "Compiling PRU firmware..."
	$(CC) $(INCLUDE) $(CFLAGS) -c $(SOURCE) -o $(OBJECT)

# Link object file to create firmware (matching TI's proven approach)
$(TARGET): $(OBJECT) $(LDFILE)
	@echo "Linking PRU firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(TARGET) $(OBJECT) -m $(GEN_DIR)/$(basename $(notdir $(SOURCE))).map $(LDFILE) --library=libc.a

# Test firmware compilation
$(TEST_OBJECT): $(TEST_SOURCE) | $(GEN_DIR)
	@echo "Compiling test PRU firmware..."
	$(CC) $(INCLUDE) $(CFLAGS) -c $(TEST_SOURCE) -o $(TEST_OBJECT)

# Test firmware linking
$(TEST_TARGET): $(TEST_OBJECT) $(LDFILE)
	@echo "Linking test PRU firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(TEST_TARGET) $(TEST_OBJECT) -m $(GEN_DIR)/$(basename $(notdir $(TEST_SOURCE))).map $(LDFILE) --library=libc.a

# Clean build artifacts
clean:
	@echo "Cleaning PRU build artifacts..."
	@rm -rf $(GEN_DIR)
	@echo "Clean complete"

# Install firmware to BeagleBone /lib/firmware (run on BeagleBone)
install: $(TARGET)
	@echo "Installing PRU firmware to /lib/firmware/am335x-pru0-fw..."
	@sudo cp $(TARGET) /lib/firmware/am335x-pru0-fw
	@echo "Firmware installed. Reload PRU with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"

# Deploy PRU firmware to BeagleBone (builds on target)
deploy:
	@echo "========================================"
	@echo "Deploying PRU IR Detector Firmware"
	@echo "========================================"
	@echo ""
	@echo "[1/5] Creating directory on $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(REMOTE_DIR)"
	@echo ""
	@echo "[2/5] Copying source files to BeagleBone..."
	@scp *.c *.h *.cmd Makefile README.md $(DEPLOY_HOST):$(REMOTE_DIR)/
	@echo ""
	@echo "[3/5] Building PRU firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(REMOTE_DIR) && make clean && make"
	@echo ""
	@echo "[4/5] Installing and loading firmware..."
	@ssh $(DEPLOY_HOST) "\
		echo 'Stopping PRU...' && \
		(echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state > /dev/null 2>&1 || true) && \
		sleep 1 && \
		echo 'Installing firmware...' && \
		sudo cp $(REMOTE_DIR)/gen/pru0_ir_detector.out $(PRU_INSTALL_PATH) && \
		echo 'Starting PRU...' && \
		echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state > /dev/null && \
		sleep 1"
	@echo ""
	@echo "[5/5] Verifying PRU state..."
	@ssh $(DEPLOY_HOST) "cat /sys/class/remoteproc/remoteproc1/state"
	@echo ""
	@echo "✓ PRU firmware deployed and loaded successfully!"
	@echo ""
	@echo "To verify operation:"
	@echo "  ssh $(DEPLOY_HOST)"
	@echo "  sudo dd if=/dev/mem bs=4 skip=\$$((0x4A311000/4)) count=7 2>/dev/null | od -An -t u4"
	@echo "  (Check status field - should be non-zero if PRU is running)"

# Help target
help:
	@echo "PRU IR Detector Firmware Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build PRU firmware locally (requires clpru)"
	@echo "  test    - Build test firmware (GPIO test)"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Install firmware to /lib/firmware (run on BeagleBone)"
	@echo "  deploy  - Copy source, build on BeagleBone, install and load (recommended)"
	@echo "  help    - Show this help message"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy DEPLOY_HOST=bbb2.wegman  (default)"
	@echo ""
	@echo "Local Build Requirements:"
	@echo "  - TI PRU Code Generation Tools (clpru)"
	@echo "  - Install with: sudo apt-get install ti-pru-cgt-installer"
