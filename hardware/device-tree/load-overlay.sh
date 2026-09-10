#!/bin/bash
# Load I2S microphone device tree overlay
# Run on BeagleBone Black

OVERLAY="BB-I2S-MIC"

echo "Loading I2S microphone device tree overlay..."
echo ""

# Check if overlay file exists
if [ ! -f "/lib/firmware/${OVERLAY}-00A0.dtbo" ]; then
    echo "ERROR: Overlay file not found at /lib/firmware/${OVERLAY}-00A0.dtbo"
    echo "Please run 'make install' first"
    exit 1
fi

# Load the overlay
echo "Loading overlay: $OVERLAY"
sudo sh -c "echo '$OVERLAY' > /sys/devices/platform/bone_capemgr/slots"

# Wait a moment for device to initialize
sleep 2

# Check if loaded successfully
if grep -q "$OVERLAY" /sys/devices/platform/bone_capemgr/slots; then
    echo "✓ Overlay loaded successfully"
else
    echo "✗ Failed to load overlay"
    exit 1
fi

echo ""
echo "Checking loaded overlays:"
cat /sys/devices/platform/bone_capemgr/slots

echo ""
echo "Checking for ALSA devices:"
arecord -l

echo ""
echo "If you see a capture device listed above, the I2S microphone is working!"
echo ""
echo "Test with:"
echo "  arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 -d 5 test.wav"
echo "  aplay test.wav"
