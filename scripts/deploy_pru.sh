#!/bin/bash
# Quick deployment script for PRU firmware updates
# Run this on the BeagleBone Black as root after making PRU firmware changes

set -e

# Get absolute paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PRU_DIR="$PROJECT_DIR/pru"

echo "====================================="
echo "PRU Firmware Deployment"
echo "====================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root"
    echo "Please run: sudo $0"
    exit 1
fi

echo ""
echo "[1/3] Building PRU firmware..."
cd "$PRU_DIR"
make clean
make
cd "$PROJECT_DIR"

echo ""
echo "[2/3] Installing PRU firmware..."
cp "$PRU_DIR/gen/pru0_adc_sampler.out" /lib/firmware/am335x-pru0-fw

echo ""
echo "[3/3] Reloading PRU..."
# Stop PRU
echo 'stop' > /sys/class/remoteproc/remoteproc1/state
sleep 1

# Start PRU
echo 'start' > /sys/class/remoteproc/remoteproc1/state
sleep 1

# Check status
STATE=$(cat /sys/class/remoteproc/remoteproc1/state)
echo "PRU state: $STATE"

if [ "$STATE" = "running" ]; then
    echo ""
    echo "====================================="
    echo "PRU firmware deployed successfully!"
    echo "====================================="
else
    echo ""
    echo "ERROR: PRU did not start properly"
    echo "Check dmesg for errors: dmesg | tail -20"
    exit 1
fi
