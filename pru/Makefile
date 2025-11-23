# PRU Firmware Makefile
# Builds PRU firmware for BeagleBone Black

# Directories - define early so they can be used in compiler flags
GEN_DIR := gen

# Detect which compiler is available
ifneq ($(wildcard /usr/bin/clpru),)
    # TI Compiler
    CC := clpru
    CFLAGS := -v3 -O2 --display_error_number --endian=little --hardware_mac=on \
              --obj_directory=$(GEN_DIR) --pp_directory=$(GEN_DIR) -ppd -ppa
    LFLAGS := -i/usr/share/ti/cgt-pru/lib -i/usr/share/ti/cgt-pru/include \
              --reread_libs --warn_sections --stack_size=0x100 --heap_size=0x100 \
              -m $(GEN_DIR)/pru0_adc_sampler.map
    LDFILE := AM335x_PRU.cmd
    COMPILER := TI
else ifneq ($(wildcard /usr/bin/pru-gcc),)
    # GCC PRU Compiler
    CC := pru-gcc
    CFLAGS := -O2 -Wall -mmcu=am335x.pru0
    LFLAGS := -mmcu=am335x.pru0
    COMPILER := GCC
else
    $(error No PRU compiler found. Install ti-pru-cgt-installer or gcc-pru)
endif

# Find PRU software support package
PRU_SW := $(shell \
	if [ -d /usr/lib/ti/pru-software-support-package ]; then \
		echo /usr/lib/ti/pru-software-support-package; \
	elif [ -d /opt/source/pru-software-support-package ]; then \
		echo /opt/source/pru-software-support-package; \
	elif [ -d /usr/share/ti/pru-software-support-package ]; then \
		echo /usr/share/ti/pru-software-support-package; \
	else \
		echo /usr/lib/ti/pru-software-support-package; \
	fi)

# Include paths (only add if directories exist)
INCLUDE := $(shell \
	result=""; \
	[ -d /usr/share/ti/cgt-pru/include ] && result="$$result --include_path=/usr/share/ti/cgt-pru/include"; \
	[ -d $(PRU_SW)/include ] && result="$$result --include_path=$(PRU_SW)/include"; \
	[ -d $(PRU_SW)/include/am335x ] && result="$$result --include_path=$(PRU_SW)/include/am335x"; \
	echo $$result)

# Source files
SOURCE := pru0_adc_sampler.c
TARGET := $(GEN_DIR)/pru0_adc_sampler.out

.PHONY: all clean install

all: $(GEN_DIR) $(TARGET)
	@echo "Build complete using $(COMPILER) compiler"
	@echo "Output: $(TARGET)"

$(GEN_DIR):
	mkdir -p $(GEN_DIR)

ifeq ($(COMPILER),TI)
$(TARGET): $(SOURCE)
	@echo "Compiling with TI compiler..."
	$(CC) $(CFLAGS) $(INCLUDE) $(SOURCE) -z $(LDFILE) $(LFLAGS) -o $(TARGET)
else
$(TARGET): $(SOURCE)
	@echo "Compiling with GCC PRU..."
	$(CC) $(CFLAGS) -o $(TARGET) $(SOURCE)
endif

clean:
	rm -rf $(GEN_DIR)

install: $(TARGET)
	@echo "Installing PRU firmware..."
	sudo cp $(TARGET) /lib/firmware/am335x-pru0-fw
	@echo "Firmware installed. Reload PRU with:"
	@echo "  echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state"
	@echo "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"

help:
	@echo "PRU Firmware Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build PRU firmware (default)"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Copy firmware to /lib/firmware and show reload commands"
	@echo "  help    - Show this help"
	@echo ""
	@echo "Detected compiler: $(COMPILER)"
