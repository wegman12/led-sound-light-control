#!/bin/bash
# Setup script for BeagleBone Black PRU ADC sampling
# Run this on the BeagleBone Black as root

set -e

# Get absolute paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PRU_DIR="$PROJECT_DIR/pru"

echo "====================================="
echo "BeagleBone Black PRU Setup"
echo "====================================="
echo ""
echo "Project directory: $PROJECT_DIR"
echo "PRU directory: $PRU_DIR"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root"
    echo "Please run: sudo $0"
    exit 1
fi

echo "[1/6] Updating package list..."
apt-get update

echo ""
echo "[2/6] Installing PRU development tools..."

# Install device tree compiler (always needed)
apt-get install -y device-tree-compiler

# Try to install PRU compiler (try GCC first, then TI)
if apt-cache show gcc-pru &>/dev/null; then
    echo "Installing gcc-pru compiler..."
    apt-get install -y gcc-pru
elif apt-cache show ti-pru-cgt-installer &>/dev/null; then
    echo "Installing TI PRU compiler..."
    apt-get install -y ti-pru-cgt-installer
else
    echo "WARNING: No PRU compiler found in repositories"
    echo "You may need to install manually from:"
    echo "  https://github.com/dinuxbg/gnupru"
fi

# Try to install PRU software package (optional, may not be available)
if apt-cache show am335x-pru-package &>/dev/null; then
    echo "Installing am335x-pru-package..."
    apt-get install -y am335x-pru-package
else
    echo "Note: am335x-pru-package not available in repos"
    echo "This is OK - PRU support may already be in kernel"
fi

# Check if PRU software support is already present
if [ -d /usr/lib/ti/pru-software-support-package ]; then
    echo "✓ PRU software support package found"
elif [ -d /opt/source/pru-software-support-package ]; then
    echo "✓ PRU software support package found in /opt"
else
    echo "Note: PRU software support package not found"
    echo "Firmware will still compile with standard includes"
fi

echo ""
echo "[3/6] Installing device tree overlay..."
if [ -f "$PRU_DIR/PRU-ADC-00A0.dts" ]; then
    cp "$PRU_DIR/PRU-ADC-00A0.dts" /lib/firmware/
    cd /lib/firmware
    dtc -O dtb -o PRU-ADC-00A0.dtbo -b 0 -@ PRU-ADC-00A0.dts
    cd "$PROJECT_DIR"
    echo "Device tree overlay installed at /lib/firmware/PRU-ADC-00A0.dtbo"
else
    echo "WARNING: PRU-ADC-00A0.dts not found at $PRU_DIR, skipping device tree overlay"
fi

echo ""
echo "[4/6] Configuring boot environment..."
UENV_FILE="/boot/uEnv.txt"
if [ -f "$UENV_FILE" ]; then
    # Check if PRU remoteproc is already enabled
    if ! grep -q "uboot_overlay_pru=" "$UENV_FILE"; then
        echo "" >> "$UENV_FILE"
        echo "# Enable PRU remoteproc" >> "$UENV_FILE"
        echo "uboot_overlay_pru=/lib/firmware/AM335X-PRU-RPROC-4-19-TI-00A0.dtbo" >> "$UENV_FILE"
        echo "PRU remoteproc enabled in $UENV_FILE"
    else
        echo "PRU remoteproc already configured"
    fi

    # Add device tree overlay
    if ! grep -q "PRU-ADC-00A0" "$UENV_FILE"; then
        echo "dtb_overlay=/lib/firmware/PRU-ADC-00A0.dtbo" >> "$UENV_FILE"
        echo "Device tree overlay configured"
    fi
else
    echo "WARNING: $UENV_FILE not found"
fi

echo ""
echo "[5/6] Building PRU firmware..."
if [ -d "$PRU_DIR" ]; then
    cd "$PRU_DIR"
    make clean
    make
    echo "PRU firmware built successfully"
    cd "$PROJECT_DIR"
else
    echo "ERROR: PRU directory not found at $PRU_DIR"
    exit 1
fi

echo ""
echo "[6/6] Installing PRU firmware..."
if [ -f "$PRU_DIR/gen/pru0_adc_sampler.out" ]; then
    cp "$PRU_DIR/gen/pru0_adc_sampler.out" /lib/firmware/am335x-pru0-fw
    echo "PRU firmware installed at /lib/firmware/am335x-pru0-fw"
else
    echo "ERROR: PRU firmware not found at $PRU_DIR/gen/pru0_adc_sampler.out"
    exit 1
fi

echo ""
echo "====================================="
echo "Setup Complete!"
echo "====================================="
echo ""
echo "Next steps:"
echo "1. Reboot the BeagleBone Black: sudo reboot"
echo "2. After reboot, verify PRU is running:"
echo "   cat /sys/class/remoteproc/remoteproc1/state"
echo "   (should show 'running')"
echo "3. Build and run the Go application as root:"
echo "   cd .. && make build"
echo "   sudo ./bin/led-sound-light-control"
echo ""
