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

# Write to pinmux register using Python
python3 - <<PYTHON_SCRIPT
import struct
import os

PINMUX_BASE = $PINMUX_BASE
PIN_OFFSET = $PIN_OFFSET
PIN_CONFIG = $PIN_CONFIG
PAGE_SIZE = 4096

# Open /dev/mem
with open('/dev/mem', 'r+b', buffering=0) as mem:
    # Calculate page-aligned address
    page_addr = (PINMUX_BASE + PIN_OFFSET) & ~(PAGE_SIZE - 1)
    page_offset = (PINMUX_BASE + PIN_OFFSET) - page_addr

    # Map the page
    import mmap
    mm = mmap.mmap(mem.fileno(), PAGE_SIZE,
                   mmap.MAP_SHARED,
                   mmap.PROT_READ | mmap.PROT_WRITE,
                   offset=page_addr)

    # Write the configuration value
    mm.seek(page_offset)
    mm.write(struct.pack('<I', PIN_CONFIG))

    # Read back to verify
    mm.seek(page_offset)
    value = struct.unpack('<I', mm.read(4))[0]

    mm.close()

    print(f"Pin configured: wrote 0x{PIN_CONFIG:02x}, read back 0x{value:02x}")
PYTHON_SCRIPT

echo "Pin configured via Python"

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
