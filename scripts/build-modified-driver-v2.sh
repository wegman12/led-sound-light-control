#!/bin/bash
# Build modified davinci-mcasp driver - Simplified version

set -e

BBB_HOST="bbb2.wegman"
KERNEL_SOURCE="/home/kevin/repositories/external/linux_kernel/beagleboard"
BUILD_DIR="~/mcasp-clkrate-build"

echo "======================================"
echo "Building Modified McASP Driver v2"
echo "with Dynamic Clock Rate Support"
echo "======================================"
echo

# Apply modifications locally
echo "[1/6] Applying modifications to driver locally..."
python3 scripts/apply-driver-modifications.py \
    "$KERNEL_SOURCE/sound/soc/ti/davinci-mcasp.c" \
    /tmp/davinci-mcasp-modified.c

echo "[2/6] Copying modified source and headers to BeagleBone..."
scp /tmp/davinci-mcasp-modified.c "$BBB_HOST:$BUILD_DIR/davinci-mcasp.c"
# Copy all local header files needed
for header in edma-pcm.h sdma-pcm.h udma-pcm.h davinci-mcasp.h; do
    scp "$KERNEL_SOURCE/sound/soc/ti/$header" "$BBB_HOST:$BUILD_DIR/" 2>/dev/null || echo "Warning: $header not found"
done

# Create Makefile
echo "[3/6] Creating Makefile on BeagleBone..."
ssh "$BBB_HOST" "cat > $BUILD_DIR/Makefile << 'EOF'
# Makefile for davinci-mcasp module with clk_set_rate support
obj-m += snd-soc-davinci-mcasp.o
snd-soc-davinci-mcasp-objs := davinci-mcasp.o

KDIR := /lib/modules/\$(shell uname -r)/build

all:
	make -C \$(KDIR) M=\$(PWD) modules

clean:
	make -C \$(KDIR) M=\$(PWD) clean

install:
	sudo insmod snd-soc-davinci-mcasp.ko

remove:
	sudo rmmod snd-soc-davinci-mcasp 2>/dev/null || true

EOF
"

# Build the module
echo "[4/6] Building kernel module on BeagleBone..."
ssh "$BBB_HOST" "cd $BUILD_DIR && make"

# Check build result
echo "[5/6] Checking build output..."
ssh "$BBB_HOST" "cd $BUILD_DIR && ls -lh *.ko"

# Count changes
echo "[6/6] Summary of changes..."
ssh "$BBB_HOST" "cd $BUILD_DIR && wc -l davinci-mcasp.c"

echo
echo "======================================"
echo "Build Complete!"
echo "======================================"
echo "Module location: $BBB_HOST:$BUILD_DIR/snd-soc-davinci-mcasp.ko"
echo
echo "Next: Load and test the module"
echo "  ./scripts/load-modified-driver.sh"
echo
