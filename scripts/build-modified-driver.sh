#!/bin/bash
# Build modified davinci-mcasp driver with clk_set_rate() support

set -e

BBB_HOST="bbb2.wegman"
KERNEL_DIR="~/linux_kernel/beagleboard"
BUILD_DIR="~/mcasp-clkrate-build"
PATCH_FILE="patches/0001-davinci-mcasp-add-dynamic-clock-rate-support.patch"

echo "======================================"
echo "Building Modified McASP Driver"
echo "with Dynamic Clock Rate Support"
echo "======================================"
echo

# Check if patch file exists
if [ ! -f "$PATCH_FILE" ]; then
    echo "ERROR: Patch file not found: $PATCH_FILE"
    exit 1
fi

# Copy patch to BeagleBone
echo "[1/7] Copying patch to BeagleBone..."
scp "$PATCH_FILE" "$BBB_HOST:/tmp/mcasp-clkrate.patch"

# Create build directory and copy source files
echo "[2/7] Setting up build environment on BeagleBone..."
ssh "$BBB_HOST" "mkdir -p $BUILD_DIR"

# Copy necessary files from kernel tree
echo "[3/7] Copying source files from kernel..."
ssh "$BBB_HOST" "cd $KERNEL_DIR && \\
    cp sound/soc/ti/davinci-mcasp.c $BUILD_DIR/ && \\
    cp sound/soc/ti/Makefile $BUILD_DIR/ 2>/dev/null || echo 'Makefile not needed'"

# Apply patch
echo "[4/7] Applying patch..."
ssh "$BBB_HOST" "cd $BUILD_DIR && \\
    patch -p3 < /tmp/mcasp-clkrate.patch && \\
    echo 'Patch applied successfully'"

# Create Makefile for module build
echo "[5/7] Creating module Makefile..."
ssh "$BBB_HOST" "cat > $BUILD_DIR/Makefile << 'EOF'
# Makefile for davinci-mcasp module with clk_set_rate support
obj-m += snd-soc-davinci-mcasp.o
snd-soc-davinci-mcasp-objs := davinci-mcasp.o

# Kernel build directory
KDIR := /lib/modules/\$(shell uname -r)/build

all:
	make -C \$(KDIR) M=\$(PWD) modules

clean:
	make -C \$(KDIR) M=\$(PWD) clean

install:
	sudo insmod snd-soc-davinci-mcasp.ko

remove:
	sudo rmmod snd-soc-davinci-mcasp

EOF
"

# Build the module
echo "[6/7] Building kernel module..."
ssh "$BBB_HOST" "cd $BUILD_DIR && \\
    make && \\
    ls -lh *.ko"

echo "[7/7] Build complete!"
echo
echo "======================================"
echo "Module built successfully!"
echo "======================================"
echo "Location: $BBB_HOST:$BUILD_DIR/snd-soc-davinci-mcasp.ko"
echo
echo "Next steps:"
echo "  1. Update device tree overlay for dynamic clocks"
echo "  2. Load the module: ./scripts/load-modified-driver.sh"
echo "  3. Test audio recording"
echo
