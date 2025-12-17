#!/bin/bash
# Test BeagleBone Black ADC Health
# This script checks if the ADC inputs are functioning correctly

echo "BeagleBone Black ADC Health Check"
echo "=================================="
echo ""

# Enable ADC
echo "Enabling ADC..."
if [ ! -d "/sys/bus/iio/devices/iio:device0" ]; then
    echo "ERROR: ADC not found. Is it enabled in device tree?"
    exit 1
fi

echo "ADC found at: /sys/bus/iio/devices/iio:device0"
echo ""

# Test with nothing connected (should read near 0)
echo "Test 1: Reading AIN0 (P9_39) with nothing connected..."
echo "Expected: Close to 0 (0-100 typical for floating input)"
AIN0=$(cat /sys/bus/iio/devices/iio:device0/in_voltage0_raw)
echo "AIN0: $AIN0 / 4095 ($(echo "scale=2; $AIN0 * 1.8 / 4095" | bc)V)"
echo ""

echo "Test 2: Reading AIN1 (P9_40) - YOUR MICROPHONE INPUT"
echo "Expected: Should vary with sound, stay under 4095"
echo "If constantly at 4095, the input voltage is >1.8V (BAD)"
for i in {1..5}; do
    AIN1=$(cat /sys/bus/iio/devices/iio:device0/in_voltage1_raw)
    VOLTAGE=$(echo "scale=3; $AIN1 * 1.8 / 4095" | bc)
    echo "  Sample $i: $AIN1 / 4095 ($VOLTAGE V)"
    sleep 0.5
done
echo ""

# Check if clipping
if [ "$AIN1" -eq 4095 ]; then
    echo "⚠️  WARNING: AIN1 is at maximum (4095)"
    echo "    This means input voltage is >1.8V"
    echo "    DISCONNECT MICROPHONE IMMEDIATELY to prevent ADC damage!"
    echo ""
fi

# Test other ADC channels
echo "Test 3: Other ADC channels (should be near 0 if unused)"
for i in 2 3 4 5 6; do
    VAL=$(cat /sys/bus/iio/devices/iio:device0/in_voltage${i}_raw)
    echo "AIN${i}: $VAL / 4095"
done
echo ""

echo "=================================="
echo "Interpretation:"
echo "  0-100:   Normal (floating or ground)"
echo "  100-3900: Valid signal range"
echo "  3900-4095: Clipping (input too high)"
echo "  4095:     Constant clipping (DAMAGE RISK)"
echo "=================================="
