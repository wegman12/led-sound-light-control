# PRU1 Audio Sampling Firmware Makefile
# Builds I2S/McASP audio firmware for BeagleBone Black PRU1

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

# Source files for I2S audio firmware with 2048-point FFT (uses DDR) - DEFAULT
I2S_2048_SOURCES := pru1_audio_i2s_2048.c fft_2048.c
I2S_2048_OBJECTS := $(GEN_DIR)/pru1_audio_i2s_2048.obj $(GEN_DIR)/fft_2048.obj
I2S_2048_TARGET := $(GEN_DIR)/pru1_audio_i2s_2048.out

# Source files for I2S/McASP audio firmware (1024-point FFT) - ALTERNATIVE
I2S_SOURCES := pru1_audio_i2s.c fft.c
I2S_OBJECTS := $(GEN_DIR)/pru1_audio_i2s.obj $(GEN_DIR)/fft.obj
I2S_TARGET := $(GEN_DIR)/pru1_audio_i2s.out

# Source files for raw I2S capture firmware (no FFT, for debugging)
I2S_RAW_SOURCES := pru1_raw_capture_i2s.c
I2S_RAW_OBJECTS := $(GEN_DIR)/pru1_raw_capture_i2s.obj
I2S_RAW_TARGET := $(GEN_DIR)/pru1_raw_capture_i2s.out

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean

# Utility binary names
UTIL_BINARY := audio-util
UTIL_BINARY_BBB := audio-util-bbb

# Phony targets
.PHONY: all clean install deploy deploy-all help build-util build-util-bbb deploy-util \
        i2s-1024 i2s-1024-install i2s-1024-deploy-all \
        raw raw-install raw-deploy-all

# Default target - build 2048-point I2S firmware
all: $(GEN_DIR) $(I2S_2048_TARGET)
	@echo "PRU1 I2S 2048-point audio firmware build complete: $(I2S_2048_TARGET)"

# Create output directory
$(GEN_DIR):
	@mkdir -p $(GEN_DIR)

# === I2S 2048-point FFT Firmware (DEFAULT) ===

# Compile I2S 2048-point audio source to object file
$(GEN_DIR)/pru1_audio_i2s_2048.obj: pru1_audio_i2s_2048.c resource_table_empty.h fft_2048.h am335x_mcasp.h | $(GEN_DIR)
	@echo "Compiling pru1_audio_i2s_2048.c..."
	$(CC) $(INCLUDE) --include_path=. --include_path=.. $(CFLAGS) -c pru1_audio_i2s_2048.c -o $(GEN_DIR)/pru1_audio_i2s_2048.obj

# Compile 2048-point FFT source to object file
$(GEN_DIR)/fft_2048.obj: fft_2048.c fft_2048.h | $(GEN_DIR)
	@echo "Compiling fft_2048.c..."
	$(CC) $(INCLUDE) --include_path=. --include_path=.. $(CFLAGS) -c fft_2048.c -o $(GEN_DIR)/fft_2048.obj

# Link I2S 2048-point object files to create firmware
$(I2S_2048_TARGET): $(I2S_2048_OBJECTS) $(LDFILE)
	@echo "Linking PRU1 I2S 2048-point audio firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(I2S_2048_TARGET) $(I2S_2048_OBJECTS) \
		-m $(GEN_DIR)/pru1_audio_i2s_2048.map $(LDFILE) --library=libc.a

# Install I2S 2048-point audio firmware to BeagleBone /lib/firmware
install: $(I2S_2048_TARGET)
	@echo "Installing PRU1 I2S 2048-point audio firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(I2S_2048_TARGET) /lib/firmware/am335x-pru1-fw
	@echo "Firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Deploy I2S 2048-point audio firmware to BeagleBone (builds on target) - DEFAULT
deploy-all:
	@echo "Deploying PRU1 I2S 2048-point audio firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@scp ../pru_shared_memory.h ../pru_ddr_memory.h $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 I2S 2048-point audio firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make"
	@echo "Installing and loading I2S 2048-point audio firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)gen/pru1_audio_i2s_2048.out /lib/firmware/am335x-pru1-fw && \
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
	@echo "=============================================="
	@echo "I2S 2048-point Audio Deployment complete!"
	@echo "=============================================="
	@echo "PRU1 firmware: $(DEPLOY_PATH)gen/pru1_audio_i2s_2048.out -> /lib/firmware/am335x-pru1-fw"
	@echo "Utility: $(DEPLOY_PATH)$(UTIL_BINARY)"
	@echo ""
	@echo "IMPORTANT: BB-I2S-PRU-00A0 and BB-PRU-DDR-00A0 overlays must be loaded!"
	@echo ""
	@echo "To test on BeagleBone:"
	@echo "  ssh $(DEPLOY_HOST)"
	@echo "  cd $(DEPLOY_PATH)"
	@echo "  sudo ./$(UTIL_BINARY) test"

# Alias for deploy-all
deploy: deploy-all

# === I2S 1024-point FFT Firmware (ALTERNATIVE) ===

# Build I2S 1024-point audio firmware
i2s-1024: $(GEN_DIR) $(I2S_TARGET)
	@echo "PRU1 I2S 1024-point audio firmware build complete: $(I2S_TARGET)"

# Compile I2S audio source to object file
$(GEN_DIR)/pru1_audio_i2s.obj: pru1_audio_i2s.c resource_table_empty.h fft.h am335x_mcasp.h pru_shared_memory.h | $(GEN_DIR)
	@echo "Compiling pru1_audio_i2s.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c pru1_audio_i2s.c -o $(GEN_DIR)/pru1_audio_i2s.obj

$(GEN_DIR)/fft.obj: fft.c fft.h | $(GEN_DIR)
	@echo "Compiling fft.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c fft.c -o $(GEN_DIR)/fft.obj

# Link I2S audio object files to create firmware
$(I2S_TARGET): $(I2S_OBJECTS) $(LDFILE)
	@echo "Linking PRU1 I2S 1024-point audio firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(I2S_TARGET) $(I2S_OBJECTS) \
		-m $(GEN_DIR)/pru1_audio_i2s.map $(LDFILE) --library=libc.a

# Install I2S 1024-point audio firmware to BeagleBone /lib/firmware
i2s-1024-install: $(I2S_TARGET)
	@echo "Installing PRU1 I2S 1024-point audio firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(I2S_TARGET) /lib/firmware/am335x-pru1-fw
	@echo "I2S 1024-point audio firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Deploy I2S 1024-point audio firmware to BeagleBone (builds on target)
i2s-1024-deploy-all:
	@echo "Deploying PRU1 I2S 1024-point audio firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@scp ../pru_shared_memory.h $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 I2S 1024-point audio firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make i2s-1024"
	@echo "Installing and loading I2S 1024-point audio firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)gen/pru1_audio_i2s.out /lib/firmware/am335x-pru1-fw && \
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
	@echo "===================================="
	@echo "I2S 1024-point Audio Deployment complete!"
	@echo "===================================="
	@echo "PRU1 firmware: $(DEPLOY_PATH)gen/pru1_audio_i2s.out -> /lib/firmware/am335x-pru1-fw"
	@echo "Utility: $(DEPLOY_PATH)$(UTIL_BINARY)"

# === Raw I2S Capture Firmware (for debugging) ===

# Build raw I2S capture firmware
raw: $(GEN_DIR) $(I2S_RAW_TARGET)
	@echo "PRU1 raw I2S capture firmware build complete: $(I2S_RAW_TARGET)"

# Compile raw I2S capture source to object file
$(GEN_DIR)/pru1_raw_capture_i2s.obj: pru1_raw_capture_i2s.c resource_table_empty.h am335x_mcasp.h pru_shared_memory.h | $(GEN_DIR)
	@echo "Compiling pru1_raw_capture_i2s.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c pru1_raw_capture_i2s.c -o $(GEN_DIR)/pru1_raw_capture_i2s.obj

# Link raw I2S capture object file to create firmware
$(I2S_RAW_TARGET): $(I2S_RAW_OBJECTS) $(LDFILE)
	@echo "Linking PRU1 raw I2S capture firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(I2S_RAW_TARGET) $(I2S_RAW_OBJECTS) \
		-m $(GEN_DIR)/pru1_raw_capture_i2s.map $(LDFILE) --library=libc.a

# Install raw I2S capture firmware to BeagleBone /lib/firmware
raw-install: $(I2S_RAW_TARGET)
	@echo "Installing PRU1 raw I2S capture firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(I2S_RAW_TARGET) /lib/firmware/am335x-pru1-fw
	@echo "Raw I2S capture firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Deploy raw I2S capture firmware to BeagleBone (builds on target)
raw-deploy-all:
	@echo "Deploying PRU1 raw I2S capture firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@scp ../pru_shared_memory.h $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 raw I2S capture firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make raw"
	@echo "Installing and loading raw I2S capture firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)gen/pru1_raw_capture_i2s.out /lib/firmware/am335x-pru1-fw && \
	                     echo 'Stopping PRU1 (if running)...' && \
	                     (echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null 2>&1 || true) && \
	                     sleep 1 && \
	                     echo 'Starting PRU1...' && \
	                     echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null && \
	                     sleep 1 && \
	                     echo 'Verifying PRU1 state...' && \
	                     cat /sys/class/remoteproc/remoteproc2/state"
	@echo ""
	@echo "Deploying raw_i2s_capture utility..."
	@cd ../../analysis/audio/raw_i2s_capture && make deploy DEPLOY_HOST=$(DEPLOY_HOST)
	@echo ""
	@echo "========================================"
	@echo "Raw I2S Capture Deployment complete!"
	@echo "========================================"
	@echo "PRU1 firmware: $(DEPLOY_PATH)gen/pru1_raw_capture_i2s.out -> /lib/firmware/am335x-pru1-fw"
	@echo "Capture utility: ~/led-sound-light-control/raw_i2s_capture"
	@echo ""
	@echo "To capture raw samples:"
	@echo "  ssh $(DEPLOY_HOST)"
	@echo "  cd ~/led-sound-light-control"
	@echo "  sudo ./raw_i2s_capture 10 baseline.bin    # 10 seconds of silence"
	@echo "  sudo ./raw_i2s_capture 30 music.bin       # 30 seconds of audio"

# === Utility Targets ===

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

# === Utility Targets ===

# Clean build artifacts
clean:
	@echo "Cleaning PRU1 audio build artifacts..."
	@rm -rf $(GEN_DIR)
	@if [ -d $(SCRIPTS_DIR) ]; then cd $(SCRIPTS_DIR) && $(GOCLEAN) && rm -f $(UTIL_BINARY) $(UTIL_BINARY_BBB); fi
	@echo "Clean complete"

# Help target
help:
	@echo "PRU1 I2S Audio Sampling Build System"
	@echo ""
	@echo "Default Firmware (2048-point FFT with DDR):"
	@echo "  all             - Build I2S 2048-point audio firmware (default)"
	@echo "  install         - Install firmware to /lib/firmware (requires sudo)"
	@echo "  deploy-all      - Build and deploy to BeagleBone (default)"
	@echo "  deploy          - Alias for deploy-all"
	@echo ""
	@echo "Alternative Firmware (1024-point FFT, no DDR):"
	@echo "  i2s-1024            - Build I2S 1024-point audio firmware"
	@echo "  i2s-1024-install    - Install 1024-point firmware"
	@echo "  i2s-1024-deploy-all - Deploy 1024-point firmware to BeagleBone"
	@echo ""
	@echo "Raw Capture (for debugging):"
	@echo "  raw             - Build raw I2S capture firmware"
	@echo "  raw-install     - Install raw capture firmware"
	@echo "  raw-deploy-all  - Deploy raw capture firmware to BeagleBone"
	@echo ""
	@echo "Utility Targets:"
	@echo "  build-util     - Build audio utility for local testing (amd64)"
	@echo "  build-util-bbb - Build audio utility for BeagleBone (ARM v7)"
	@echo "  deploy-util    - Deploy utility to BeagleBone"
	@echo ""
	@echo "Other:"
	@echo "  clean          - Remove all build artifacts"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Requirements:"
	@echo "  - TI PRU Code Generation Tools (clpru) for firmware"
	@echo "  - Go 1.23+ for utility"
	@echo "  - Device tree overlays: BB-I2S-PRU-00A0, BB-PRU-DDR-00A0"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Connect INMP441/SPH0645 microphone to P9 header"
	@echo "  2. Add jumper wire: P9_25 -> P9_28 (clock routing)"
	@echo "  3. Ensure BB-I2S-PRU-00A0 and BB-PRU-DDR-00A0 overlays are loaded"
	@echo "  4. make deploy-all              # Deploy firmware"
	@echo "  5. sudo ./audio-util test       # Verify operation"
	@echo ""
	@echo "Current settings:"
	@echo "  DEPLOY_HOST = $(DEPLOY_HOST)"
