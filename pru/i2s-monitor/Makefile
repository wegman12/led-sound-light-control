# Makefile for PRU I2S Monitor
# Matches the proven two-step build process from pru/remote

# Build directories
GEN_DIR := gen
SRC_DIR := .

# TI PRU Compiler and Linker
CC := clpru
LD := clpru

# Source files
SOURCE := pru0_i2s_monitor.c
OBJECT := $(GEN_DIR)/pru0_i2s_monitor.obj
TARGET := $(GEN_DIR)/pru0_i2s_monitor.out

# Compiler flags - matching working firmware
CFLAGS := -v3 -O2 --display_error_number --endian=little --hardware_mac=on \
          --obj_directory=$(GEN_DIR) --pp_directory=$(GEN_DIR) -ppd -ppa

# Linker flags - CRITICAL for proper ELF generation
LFLAGS := --reread_libs --warn_sections --stack_size=0x100 --heap_size=0x100

# Library paths for linker
LIB_INCLUDE := -i/usr/share/ti/cgt-pru/lib -i/usr/share/ti/cgt-pru/include

# Include paths (TI PRU Software Support Package)
INCLUDE := --include_path=$(SRC_DIR) \
           --include_path=/usr/share/ti/cgt-pru/include \
           --include_path=/usr/share/ti/cgt-pru/lib \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include/am335x

# Linker command file (using cookbook version like working firmware)
LDFILE := am335x_pru.cmd

.PHONY: all clean

all: $(GEN_DIR) $(TARGET)

# Create output directory
$(GEN_DIR):
	@mkdir -p $(GEN_DIR)

# Step 1: Compile C source to object file
$(OBJECT): $(SOURCE) | $(GEN_DIR)
	@echo "Compiling PRU I2S monitor..."
	$(CC) $(INCLUDE) $(CFLAGS) -c $(SOURCE) -o $(OBJECT)

# Step 2: Link object file to create firmware (matching working approach)
$(TARGET): $(OBJECT) $(LDFILE)
	@echo "Linking PRU firmware..."
	$(LD) $(CFLAGS) -z $(LIB_INCLUDE) $(LFLAGS) -o $(TARGET) $(OBJECT) \
		-m $(GEN_DIR)/pru0_i2s_monitor.map $(LDFILE) --library=libc.a
	@echo "✓ Built $(TARGET)"

clean:
	rm -rf gen

install: $(TARGET)
	sudo cp $(TARGET) /lib/firmware/am335x-pru0-fw
	@echo "✓ Installed to /lib/firmware/am335x-pru0-fw"
	@echo "To load firmware:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"

# Remote compilation and deployment targets
DEPLOY_HOST := bbb2.wegman
DEPLOY_PATH := ~/i2s-monitor/

compile-remote:
	@echo "Deploying source files to $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)"
	@scp *.c *.h *.cmd Makefile $(DEPLOY_HOST):$(DEPLOY_PATH)
	@echo "Compiling on $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "cd $(DEPLOY_PATH) && make clean && make"
	@echo "Copying firmware back..."
	@scp $(DEPLOY_HOST):$(DEPLOY_PATH)$(TARGET) ./gen/
	@echo "✓ Remote compilation complete: gen/pru0_i2s_monitor.out"

deploy-full: compile-remote
	@echo "Installing firmware on $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "sudo cp $(DEPLOY_PATH)$(TARGET) /lib/firmware/am335x-pru0-fw"
	@echo "Stopping PRU0..."
	@ssh $(DEPLOY_HOST) "echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state" || true
	@sleep 1
	@echo "Starting PRU0..."
	@ssh $(DEPLOY_HOST) "echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"
	@sleep 1
	@echo "Checking PRU0 state..."
	@ssh $(DEPLOY_HOST) "cat /sys/class/remoteproc/remoteproc1/state"
	@echo "✓ Deployment complete"
