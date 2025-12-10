# PRU IR Detector Firmware Makefile
# Builds firmware for BeagleBone Black PRU0

# Build directories
GEN_DIR := gen
SRC_DIR := .

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

# Phony targets
.PHONY: all clean install help

# Default target
all: $(GEN_DIR) $(TARGET)
	@echo "PRU firmware build complete: $(TARGET)"

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

# Clean build artifacts
clean:
	@echo "Cleaning PRU build artifacts..."
	@rm -rf $(GEN_DIR)
	@echo "Clean complete"

# Install firmware to BeagleBone /lib/firmware
install: $(TARGET)
	@echo "Installing PRU firmware to /lib/firmware/am335x-pru0-fw..."
	@sudo cp $(TARGET) /lib/firmware/am335x-pru0-fw
	@echo "Firmware installed. Reload PRU with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"

# Help target
help:
	@echo "PRU IR Detector Firmware Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build PRU firmware (default)"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Install firmware to /lib/firmware (requires sudo)"
	@echo "  help    - Show this help message"
	@echo ""
	@echo "Requirements:"
	@echo "  - TI PRU Code Generation Tools (clpru)"
	@echo "  - Install with: sudo apt-get install ti-pru-cgt-installer"
