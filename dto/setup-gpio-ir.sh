#!/bin/bash
# GPIO IR Receiver Pin Configuration Script
# Configures P9.41 (GPIO0_20) with pull-up resistor for IR receiver

set -e

# Pin configuration
# P9_41: offset 0x1B4, GPIO0_20
# Mode 7 (GPIO) | Input | Pull-up enabled
# Value: 0x37 (0b00110111)
#   Bit 5: Pull-up/down select (1 = pull-up)
#   Bit 4: Pull-up/down enable (1 = pull-up/down enabled)
#   Bit 3: Input enable (1 = input)
#   Bits 0-2: Mode (7 = GPIO)

PINMUX_BASE=0x44E10000
PIN_OFFSET=0x1B4
PIN_CONFIG=0x37

echo "Configuring GPIO0_20 (P9.41) for IR receiver..."

# Write to pinmux register using devmem2
if command -v devmem2 &> /dev/null; then
    devmem2 $((PINMUX_BASE + PIN_OFFSET)) w 0x$PIN_CONFIG
    echo "Pin configured via devmem2"
elif [ -f /dev/mem ]; then
    # Fallback: use busybox devmem if available
    busybox devmem $((PINMUX_BASE + PIN_OFFSET)) 32 0x$PIN_CONFIG 2>/dev/null || {
        echo "ERROR: Neither devmem2 nor busybox devmem available"
        exit 1
    }
    echo "Pin configured via busybox devmem"
else
    echo "ERROR: No method available to configure pinmux"
    exit 1
fi

# Export GPIO if not already exported
if [ ! -d /sys/class/gpio/gpio20 ]; then
    echo 20 > /sys/class/gpio/export
    echo "GPIO20 exported"
fi

# Set as input
echo in > /sys/class/gpio/gpio20/direction
echo "GPIO20 configured as input"

# Read current value to verify
VALUE=$(cat /sys/class/gpio/gpio20/value)
echo "Current GPIO20 value: $VALUE (should be 1 with pull-up)"

echo "GPIO IR receiver configuration complete"
