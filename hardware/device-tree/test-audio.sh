#!/bin/bash
# Test I2S microphone audio capture
# Run on BeagleBone Black

echo "I2S Microphone Audio Test"
echo "========================="
echo ""

# Check if ALSA device exists
echo "1. Checking for ALSA capture devices..."
DEVICE_COUNT=$(arecord -l 2>/dev/null | grep -c "card")

if [ "$DEVICE_COUNT" -eq 0 ]; then
    echo "✗ No capture devices found"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check device tree overlay is loaded:"
    echo "     cat /sys/devices/platform/bone_capemgr/slots | grep I2S"
    echo "  2. Check dmesg for errors:"
    echo "     dmesg | grep -i 'mcasp\|audio\|i2s'"
    echo "  3. Verify wiring connections"
    exit 1
fi

echo "✓ Found $DEVICE_COUNT capture device(s)"
arecord -l
echo ""

# Determine device
DEVICE="hw:0,0"  # Default to first card, first device
echo "Using device: $DEVICE"
echo ""

# Record 5 seconds of audio
echo "2. Recording 5 seconds of audio..."
echo "   Sample rate: 48000 Hz"
echo "   Format: 16-bit signed little-endian"
echo "   Channels: 1 (mono)"
echo ""
echo "   ** Make some noise near the microphone! **"
echo ""

OUTPUT_FILE="/tmp/i2s_test_$(date +%Y%m%d_%H%M%S).wav"

if arecord -D "$DEVICE" -f S16_LE -r 48000 -c 1 -d 5 "$OUTPUT_FILE" 2>&1; then
    echo "✓ Recording complete: $OUTPUT_FILE"
else
    echo "✗ Recording failed"
    echo ""
    echo "Check dmesg for errors:"
    dmesg | tail -20
    exit 1
fi

# Check file size
FILE_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
EXPECTED_SIZE=$((48000 * 2 * 5 + 44))  # 48kHz * 2 bytes * 5 sec + WAV header

echo ""
echo "3. Validating recording..."
echo "   File size: $FILE_SIZE bytes"
echo "   Expected: ~$EXPECTED_SIZE bytes"

if [ "$FILE_SIZE" -lt 1000 ]; then
    echo "✗ File too small - recording may have failed"
    exit 1
fi

echo "✓ File size looks good"
echo ""

# Analyze audio data
echo "4. Analyzing audio content..."
if command -v sox &> /dev/null; then
    echo ""
    sox "$OUTPUT_FILE" -n stat 2>&1 | grep -E "RMS|Maximum|Minimum"
    echo ""
else
    echo "   (Install 'sox' for detailed audio analysis)"
fi

# Check for silence (all zeros)
if command -v hexdump &> /dev/null; then
    # Check first 1000 bytes after WAV header for non-zero data
    NON_ZERO=$(hexdump -v -e '1/1 "%02x\n"' "$OUTPUT_FILE" | tail -n +45 | head -n 1000 | grep -v "00" | wc -l)

    if [ "$NON_ZERO" -gt 100 ]; then
        echo "✓ Audio data contains signal (not all zeros)"
    else
        echo "⚠  Audio data appears to be mostly silence"
        echo "   This could indicate:"
        echo "   - No audio input"
        echo "   - Microphone not powered"
        echo "   - Wrong pin configuration"
    fi
fi

echo ""
echo "5. Playback test"
echo "   To hear the recording, run:"
echo "     aplay $OUTPUT_FILE"
echo ""
echo "   Or copy to local machine:"
echo "     scp $(hostname):$OUTPUT_FILE ."
echo ""

# Summary
echo "========================="
echo "Test Summary"
echo "========================="
echo "Recording: $OUTPUT_FILE"
echo "Size: $FILE_SIZE bytes"
echo ""
echo "Next steps:"
echo "  1. Listen to the recording to verify audio quality"
echo "  2. Check signal strength and noise level"
echo "  3. If quality is poor, check wiring and try adjusting microphone position"
echo "  4. Proceed to Phase 2: Go ALSA integration"
