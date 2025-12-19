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

# Source files for audio processing firmware (ADC version)
SOURCES := pru1_audio.c fft.c
OBJECTS := $(GEN_DIR)/pru1_audio.obj $(GEN_DIR)/fft.obj

# Output firmware for audio processing (ADC)
TARGET := $(GEN_DIR)/pru1_audio.out

# Source files for I2S/McASP audio firmware
I2S_SOURCES := pru1_audio_i2s.c fft.c
I2S_OBJECTS := $(GEN_DIR)/pru1_audio_i2s.obj $(GEN_DIR)/fft.obj

# Output firmware for I2S audio processing
I2S_TARGET := $(GEN_DIR)/pru1_audio_i2s.out

# Source files for raw capture firmware (no FFT)
RAW_SOURCES := pru1_raw_capture.c
RAW_OBJECTS := $(GEN_DIR)/pru1_raw_capture.obj

# Output firmware for raw capture
RAW_TARGET := $(GEN_DIR)/pru1_raw_capture.out

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean

# Utility binary names
UTIL_BINARY := audio-util
UTIL_BINARY_BBB := audio-util-bbb

# Phony targets
.PHONY: all clean install help build-util build-util-bbb deploy-util deploy-all \
        raw raw-install raw-deploy-all i2s i2s-install i2s-deploy-all

# Default target
all: $(GEN_DIR) $(TARGET)
	@echo "PRU1 audio firmware build complete: $(TARGET)"

# Create output directory
$(GEN_DIR):
	@mkdir -p $(GEN_DIR)

# Compile C source files to object files
$(GEN_DIR)/pru1_audio.obj: pru1_audio.c resource_table_empty.h fft.h | $(GEN_DIR)
	@echo "Compiling pru1_audio.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c pru1_audio.c -o $(GEN_DIR)/pru1_audio.obj

$(GEN_DIR)/fft.obj: fft.c fft.h | $(GEN_DIR)
	@echo "Compiling fft.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c fft.c -o $(GEN_DIR)/fft.obj

# Link object files to create firmware
$(TARGET): $(OBJECTS) $(LDFILE)
	@echo "Linking PRU1 audio firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(TARGET) $(OBJECTS) \
		-m $(GEN_DIR)/pru1_audio.map $(LDFILE) --library=libc.a

# === Raw Capture Firmware Targets ===

# Build raw capture firmware
raw: $(GEN_DIR) $(RAW_TARGET)
	@echo "PRU1 raw capture firmware build complete: $(RAW_TARGET)"

# Compile raw capture source to object file
$(GEN_DIR)/pru1_raw_capture.obj: pru1_raw_capture.c resource_table_empty.h | $(GEN_DIR)
	@echo "Compiling pru1_raw_capture.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c pru1_raw_capture.c -o $(GEN_DIR)/pru1_raw_capture.obj

# Link raw capture object file to create firmware
$(RAW_TARGET): $(RAW_OBJECTS) $(LDFILE)
	@echo "Linking PRU1 raw capture firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(RAW_TARGET) $(RAW_OBJECTS) \
		-m $(GEN_DIR)/pru1_raw_capture.map $(LDFILE) --library=libc.a

# Clean build artifacts
clean:
	@echo "Cleaning PRU1 audio build artifacts..."
	@rm -rf $(GEN_DIR)
	@if [ -d $(SCRIPTS_DIR) ]; then cd $(SCRIPTS_DIR) && $(GOCLEAN) && rm -f $(UTIL_BINARY) $(UTIL_BINARY_BBB); fi
	@echo "Clean complete"

# Install firmware to BeagleBone /lib/firmware
install: $(TARGET)
	@echo "Installing PRU1 audio firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(TARGET) /lib/firmware/am335x-pru1-fw
	@echo "Firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Install raw capture firmware to BeagleBone /lib/firmware
raw-install: $(RAW_TARGET)
	@echo "Installing PRU1 raw capture firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(RAW_TARGET) /lib/firmware/am335x-pru1-fw
	@echo "Raw capture firmware installed. Reload PRU1 with:"
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

# Deploy raw capture firmware to BeagleBone (builds on target)
raw-deploy-all:
	@echo "Deploying PRU1 raw capture firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 raw capture firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make raw"
	@echo "Installing and loading raw capture firmware..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)gen/pru1_raw_capture.out /lib/firmware/am335x-pru1-fw && \
	                     echo 'Stopping PRU1 (if running)...' && \
	                     (echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null 2>&1 || true) && \
	                     sleep 1 && \
	                     echo 'Starting PRU1...' && \
	                     echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state > /dev/null && \
	                     sleep 1 && \
	                     echo 'Verifying PRU1 state...' && \
	                     cat /sys/class/remoteproc/remoteproc2/state"
	@echo ""
	@echo "==================================="
	@echo "Raw Capture Deployment complete!"
	@echo "==================================="
	@echo "PRU1 raw capture firmware: $(DEPLOY_PATH)gen/pru1_raw_capture.out -> /lib/firmware/am335x-pru1-fw"
	@echo ""
	@echo "The PRU is now running raw capture mode. Use the 'record-pru-raw' CLI command to capture samples."

# === I2S/McASP Audio Firmware Targets ===

# Build I2S audio firmware
i2s: $(GEN_DIR) $(I2S_TARGET)
	@echo "PRU1 I2S audio firmware build complete: $(I2S_TARGET)"

# Compile I2S audio source to object file
$(GEN_DIR)/pru1_audio_i2s.obj: pru1_audio_i2s.c resource_table_empty.h fft.h am335x_mcasp.h pru_shared_memory.h | $(GEN_DIR)
	@echo "Compiling pru1_audio_i2s.c..."
	$(CC) $(INCLUDE) $(CFLAGS) -c pru1_audio_i2s.c -o $(GEN_DIR)/pru1_audio_i2s.obj

# Link I2S audio object files to create firmware
$(I2S_TARGET): $(I2S_OBJECTS) $(LDFILE)
	@echo "Linking PRU1 I2S audio firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(I2S_TARGET) $(I2S_OBJECTS) \
		-m $(GEN_DIR)/pru1_audio_i2s.map $(LDFILE) --library=libc.a

# Install I2S audio firmware to BeagleBone /lib/firmware
i2s-install: $(I2S_TARGET)
	@echo "Installing PRU1 I2S audio firmware to /lib/firmware/am335x-pru1-fw..."
	@sudo cp $(I2S_TARGET) /lib/firmware/am335x-pru1-fw
	@echo "I2S audio firmware installed. Reload PRU1 with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Deploy I2S audio firmware to BeagleBone (builds on target)
i2s-deploy-all:
	@echo "Deploying PRU1 I2S audio firmware to $(DEPLOY_HOST)..."
	@echo "Copying PRU source files to BeagleBone..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp -r *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@scp ../pru_shared_memory.h $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Building PRU1 I2S audio firmware on BeagleBone..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make i2s"
	@echo "Installing and loading I2S audio firmware..."
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
	@echo "I2S Audio Deployment complete!"
	@echo "===================================="
	@echo "PRU1 I2S firmware: $(DEPLOY_PATH)gen/pru1_audio_i2s.out -> /lib/firmware/am335x-pru1-fw"
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
	@echo "  all         - Build PRU1 ADC audio processing firmware (default)"
	@echo "  i2s         - Build PRU1 I2S/McASP audio firmware (48 kHz MEMS mic)"
	@echo "  raw         - Build PRU1 raw capture firmware (for parameter testing)"
	@echo "  clean       - Remove all build artifacts (firmware + utility)"
	@echo "  install     - Install ADC audio firmware to /lib/firmware (requires sudo)"
	@echo "  i2s-install - Install I2S audio firmware to /lib/firmware (requires sudo)"
	@echo "  raw-install - Install raw capture firmware to /lib/firmware (requires sudo)"
	@echo ""
	@echo "Utility Targets:"
	@echo "  build-util     - Build audio utility for local testing (amd64)"
	@echo "  build-util-bbb - Build audio utility for BeagleBone (ARM v7)"
	@echo "  deploy-util    - Deploy utility to BeagleBone"
	@echo ""
	@echo "Deployment Targets:"
	@echo "  deploy-all     - Deploy ADC audio firmware and utility to BeagleBone"
	@echo "  i2s-deploy-all - Deploy I2S audio firmware and utility to BeagleBone (recommended)"
	@echo "  raw-deploy-all - Deploy raw capture firmware to BeagleBone"
	@echo ""
	@echo "Help:"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Requirements:"
	@echo "  - TI PRU Code Generation Tools (clpru) for firmware"
	@echo "  - Go 1.23+ for utility"
	@echo "  - Install clpru with: sudo apt-get install ti-pru-cgt-installer"
	@echo ""
	@echo "I2S Audio Workflow (MEMS Microphone):"
	@echo "  1. Connect INMP441/SPH0645 microphone to P9 header"
	@echo "  2. Add jumper wire: P9_25 -> P9_28 (clock routing)"
	@echo "  3. Load device tree overlay: BB-I2S-PRU-00A0"
	@echo "  4. make i2s-deploy-all          # Deploy I2S firmware"
	@echo "  5. sudo ./audio-util test       # Verify operation"
	@echo ""
	@echo "Raw Capture Workflow:"
	@echo "  1. make raw-deploy-all          # Deploy raw capture firmware"
	@echo "  2. record-pru-raw -o samples.bin # Capture raw audio samples"
	@echo "  3. analyze-audio-params samples.bin # Find optimal parameters"
	@echo "  4. make i2s-deploy-all          # Switch back to I2S audio firmware"
