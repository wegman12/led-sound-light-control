#!/bin/bash
# Script to test internal clock generation (Option 1)

set -e

BBB_HOST="bbb2.wegman"
DTS_FILE="hardware/device-tree/BB-I2S-MIC-00A0.dts"
DTS_NAME="BB-I2S-MIC-00A0"

echo "======================================"
echo "Testing Internal Clock Generation"
echo "Option 1: assigned-clocks with DPLL_PER"
echo "======================================"
echo

# Copy device tree source to BeagleBone
echo "[1/6] Copying device tree source to BeagleBone..."
scp "$DTS_FILE" "$BBB_HOST:/tmp/"

# Compile on BeagleBone
echo "[2/6] Compiling device tree overlay on BeagleBone..."
ssh "$BBB_HOST" "dtc -O dtb -o /tmp/${DTS_NAME}.dtbo -b 0 -@ /tmp/${DTS_NAME}.dts"

# Copy compiled overlay to firmware directory
echo "[3/6] Installing overlay to /lib/firmware..."
ssh "$BBB_HOST" "sudo cp /tmp/${DTS_NAME}.dtbo /lib/firmware/"

# Unload any existing overlay
echo "[4/6] Unloading any existing overlay..."
ssh "$BBB_HOST" "sudo rmdir /sys/kernel/config/device-tree/overlays/BB-I2S-MIC 2>/dev/null || true"

# Load the new overlay
echo "[5/6] Loading new overlay..."
ssh "$BBB_HOST" "sudo mkdir -p /sys/kernel/config/device-tree/overlays/BB-I2S-MIC"
ssh "$BBB_HOST" "echo '${DTS_NAME}.dtbo' | sudo tee /sys/kernel/config/device-tree/overlays/BB-I2S-MIC/path"

# Wait a moment for initialization
sleep 2

# Check dmesg for results
echo "[6/6] Checking dmesg for clock and McASP messages..."
echo "--------------------------------------"
ssh "$BBB_HOST" "dmesg | tail -50 | grep -i 'mcasp\|clock\|dpll\|assigned'" || echo "No relevant messages in last 50 lines"
echo "--------------------------------------"
echo

# Check if audio device exists
echo "Checking for audio devices..."
ssh "$BBB_HOST" "arecord -l" || echo "No recording devices found"
echo

# Check McASP registers if our debug module is loaded
echo "Checking McASP clock control registers..."
ssh "$BBB_HOST" "sudo cat /sys/kernel/debug/davinci_mcasp/48038000.mcasp/regs 2>/dev/null | grep -E 'ACLKXCTL|AHCLKXCTL|PDIR|GBLCTL'" || echo "Debug registers not available"
echo

echo "======================================"
echo "Next Steps:"
echo "======================================"
echo "1. Review dmesg output above for any errors"
echo "2. Look for 'assigned clock' messages showing 98.304 MHz was set"
echo "3. Check if BCLK/WS signals appear on oscilloscope:"
echo "   - P9_31 (BCLK): Should show ~3.072 MHz"
echo "   - P9_29 (WS): Should show 48 kHz"
echo "4. Try recording: ssh $BBB_HOST 'arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 test.wav'"
echo
