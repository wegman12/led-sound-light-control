#!/bin/bash
#
# PRU Firmware Loader Script
# Loads PRU0 IR detector firmware and starts PRU
#

set -e

# Configuration
FIRMWARE_PATH="/lib/firmware/am335x-pru0-fw"
REMOTEPROC_PATH="/sys/class/remoteproc/remoteproc1"
REMOTEPROC_STATE="${REMOTEPROC_PATH}/state"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if firmware file exists
if [ ! -f "$FIRMWARE_PATH" ]; then
    echo -e "${RED}Error: PRU firmware not found at $FIRMWARE_PATH${NC}"
    echo "Build and install firmware first:"
    echo "  cd pru && make && make install"
    exit 1
fi

# Check if remoteproc driver is available
if [ ! -d "$REMOTEPROC_PATH" ]; then
    echo -e "${RED}Error: PRU remoteproc not found at $REMOTEPROC_PATH${NC}"
    echo "PRU remoteproc driver may not be loaded."
    echo "Try: sudo modprobe pru_rproc"
    exit 1
fi

echo "PRU Firmware Loader"
echo "==================="
echo ""
echo "Firmware: $FIRMWARE_PATH"
echo "Remoteproc: $REMOTEPROC_PATH"
echo ""

# Stop PRU if running
echo -n "Stopping PRU... "
if echo "stop" | sudo tee "$REMOTEPROC_STATE" > /dev/null 2>&1; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${YELLOW}WARNING: Could not stop PRU (may already be stopped)${NC}"
fi

# Wait for PRU to stop
sleep 1

# Start PRU
echo -n "Starting PRU... "
if echo "start" | sudo tee "$REMOTEPROC_STATE" > /dev/null 2>&1; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo ""
    echo "Check dmesg for errors:"
    echo "  sudo dmesg | tail -20"
    exit 1
fi

# Wait for PRU to initialize
sleep 1

# Verify PRU is running
STATE=$(cat "$REMOTEPROC_STATE")
if [ "$STATE" = "running" ]; then
    echo ""
    echo -e "${GREEN}PRU firmware loaded successfully${NC}"
    echo "State: $STATE"
    echo ""
    echo "Test detection:"
    echo "  sudo dd if=/dev/mem bs=4 skip=\$((0x4A311000/4)) count=6 2>/dev/null | od -An -t u4"
    echo "  (Press IR remote button and run again - event_count should increment)"
    exit 0
else
    echo ""
    echo -e "${RED}Error: PRU failed to start${NC}"
    echo "State: $STATE"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check kernel logs: sudo dmesg | tail -20"
    echo "  2. Verify firmware: ls -l $FIRMWARE_PATH"
    echo "  3. Check remoteproc: ls -l $REMOTEPROC_PATH"
    exit 1
fi
