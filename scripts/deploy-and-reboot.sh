#!/bin/bash
# Deploy device tree overlay and reboot BeagleBone

set -e

BBB_HOST="bbb2.wegman"
DTS_FILE="hardware/device-tree/BB-I2S-MIC-00A0.dts"
DTS_NAME="BB-I2S-MIC-00A0"

echo "======================================"
echo "Deploying Device Tree Overlay"
echo "======================================"
echo

# Copy device tree source to BeagleBone
echo "[1/4] Copying device tree source to BeagleBone..."
scp "$DTS_FILE" "$BBB_HOST:/tmp/"

# Compile on BeagleBone
echo "[2/4] Compiling device tree overlay on BeagleBone..."
ssh "$BBB_HOST" "dtc -O dtb -o /tmp/${DTS_NAME}.dtbo -b 0 -@ /tmp/${DTS_NAME}.dts"

# Copy compiled overlay to firmware directory
echo "[3/4] Installing overlay to /lib/firmware..."
ssh "$BBB_HOST" "sudo cp /tmp/${DTS_NAME}.dtbo /lib/firmware/"

echo "[4/4] Rebooting BeagleBone to load new overlay..."
echo
echo "The BeagleBone will reboot now. Wait 30-60 seconds, then run:"
echo "  ./scripts/check-i2s-clocks.sh"
echo
read -p "Press Enter to reboot BeagleBone..."

ssh "$BBB_HOST" "sudo reboot" || true

echo "Waiting for BeagleBone to reboot..."
sleep 30

echo "Testing connection..."
for i in {1..10}; do
    if ssh -o ConnectTimeout=5 "$BBB_HOST" "uptime" 2>/dev/null; then
        echo "BeagleBone is back online!"
        break
    fi
    echo "Still waiting... ($i/10)"
    sleep 5
done

echo
echo "======================================"
echo "Next: Run ./scripts/check-i2s-clocks.sh to verify"
echo "======================================"
