# PRU1 Audio Sampling Firmware Makefile
# Builds firmware for BeagleBone Black PRU1 and utility applications

# Build directories
GEN_DIR := gen
SRC_DIR := .
SCRIPTS_DIR := scripts

# Deployment configuration
DEPLOY_HOST := bbb1.wegman
DEPLOY_PATH := ~/led-sound-light-control/pru-audio/

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
INCLUDE := --include_path=$(SRC_DIR) \
           --include_path=/usr/share/ti/cgt-pru/include \
           --include_path=/usr/share/ti/cgt-pru/lib \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include/am335x

# Linker command file
LDFILE := am335x_pru.cmd

# Source files
SOURCE := pru1_audio.c
OBJECT := $(GEN_DIR)/pru1_audio.obj

# Output firmware
TARGET := $(GEN_DIR)/pru1_audio.out

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean

# Utility binary names
UTIL_BINARY := audio-util
UTIL_BINARY_BBB := audio-util-bbb

# Phony targets
.PHONY: all clean install help build-util build-util-bbb deploy-util deploy-all

# Default target
all: $(GEN_DIR) $(TARGET)
	@echo "PRU1 audio firmware build complete: $(TARGET)"

# Create output directory
$(GEN_DIR):
	@mkdir -p $(GEN_DIR)

# Compile C source to object file
$(OBJECT): $(SOURCE) resource_table_empty.h | $(GEN_DIR)
	@echo "Compiling PRU1 audio firmware..."
	$(CC) $(INCLUDE) $(CFLAGS) -c $(SOURCE) -o $(OBJECT)

# Link object file to create firmware
$(TARGET): $(OBJECT) $(LDFILE)
	@echo "Linking PRU1 audio firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(TARGET) $(OBJECT) \
		-m $(GEN_DIR)/$(basename $(notdir $(SOURCE))).map $(LDFILE) --library=libc.a

# Clean build artifacts
clean:
	@echo "Cleaning PRU1 audio build artifacts..."
	@rm -rf $(GEN_DIR)
	@cd $(SCRIPTS_DIR) && $(GOCLEAN) && rm -f $(UTIL_BINARY) $(UTIL_BINARY_BBB)
	@echo "Clean complete"

# Install firmware to BeagleBone /lib/firmware
install: $(TARGET)
	@echo "Installing PRU1 audio firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(TARGET) /lib/firmware/am335x-pru1-fw
	@echo "Firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Build utility for local testing (amd64)
build-util:
	@echo "Building audio utility for local testing..."
	@cd $(SCRIPTS_DIR) && $(GOBUILD) -o $(UTIL_BINARY) .
	@echo "Utility built: $(SCRIPTS_DIR)/$(UTIL_BINARY)"

# Build utility for BeagleBone Black (ARM v7)
build-util-bbb:
	@echo "Building audio utility for BeagleBone Black..."
	@cd $(SCRIPTS_DIR) && GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(UTIL_BINARY_BBB) .
	@echo "BeagleBone utility built: $(SCRIPTS_DIR)/$(UTIL_BINARY_BBB)"

# Deploy utility to BeagleBone
deploy-util: build-util-bbb
	@echo "Deploying audio utility to $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp $(SCRIPTS_DIR)/$(UTIL_BINARY_BBB) $(DEPLOY_HOST):$(DEPLOY_PATH)$(UTIL_BINARY)
	@echo "Utility deployed to $(DEPLOY_HOST):$(DEPLOY_PATH)$(UTIL_BINARY)"

# Deploy both firmware and utility to BeagleBone (builds on target)
deploy-all:
	@echo "Deploying PRU1 audio firmware and utility to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make"
	@echo "Installing and loading firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)gen/pru1_audio.out /lib/firmware/am335x-pru1-fw && \
	                     echo 'Stopping PRU1 (if running)...' && \
	                     (echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null 2>&1 || true) && \
	                     sleep 1 && \
	                     echo 'Starting PRU1...' && \
	                     echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null && \
	                     sleep 1 && \
	                     echo 'Verifying PRU1 state...' && \
	                     cat /sys/class/remoteproc/remoteproc2/state"
	@echo ""
	@$(MAKE) deploy-util
	@echo ""
	@echo "==================================="
	@echo "Deployment complete!"
	@echo "==================================="
	@echo "PRU1 firmware: $(DEPLOY_PATH)gen/pru1_audio.out -> /lib/firmware/am335x-pru1-fw"
	@echo "Utility: $(DEPLOY_PATH)$(UTIL_BINARY)"
	@echo ""
	@echo "To test on BeagleBone:"
	@echo "  ssh $(DEPLOY_HOST)"
	@echo "  cd $(DEPLOY_PATH)"
	@echo "  sudo ./$(UTIL_BINARY) test"

# Help target
help:
	@echo "PRU1 Audio Sampling Build System"
	@echo ""
	@echo "Firmware Targets:"
	@echo "  all         - Build PRU1 firmware (default)"
	@echo "  clean       - Remove all build artifacts (firmware + utility)"
	@echo "  install     - Install firmware to /lib/firmware (requires sudo)"
	@echo ""
	@echo "Utility Targets:"
	@echo "  build-util     - Build audio utility for local testing (amd64)"
	@echo "  build-util-bbb - Build audio utility for BeagleBone (ARM v7)"
	@echo "  deploy-util    - Deploy utility to BeagleBone"
	@echo ""
	@echo "Deployment Targets:"
	@echo "  deploy-all  - Deploy firmware and utility to BeagleBone (recommended)"
	@echo ""
	@echo "Help:"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Requirements:"
	@echo "  - TI PRU Code Generation Tools (clpru) for firmware"
	@echo "  - Go 1.23+ for utility"
	@echo "  - Install clpru with: sudo apt-get install ti-pru-cgt-installer"
