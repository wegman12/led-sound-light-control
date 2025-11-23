#!/bin/bash
# Check PRU status and display debugging information
# Can be run as a regular user

echo "====================================="
echo "PRU Status Check"
echo "====================================="
echo ""

echo "[1] PRU Remoteproc Status:"
if [ -d /sys/class/remoteproc/remoteproc1 ]; then
    STATE=$(cat /sys/class/remoteproc/remoteproc1/state)
    echo "  State: $STATE"
    if [ "$STATE" = "running" ]; then
        echo "  ✓ PRU is running"
    else
        echo "  ✗ PRU is not running"
        echo "  To start: echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"
    fi
else
    echo "  ✗ PRU remoteproc not found"
    echo "  Check if PRU remoteproc is enabled in /boot/uEnv.txt"
fi

echo ""
echo "[2] PRU Firmware:"
if [ -f /lib/firmware/am335x-pru0-fw ]; then
    echo "  ✓ PRU firmware exists at /lib/firmware/am335x-pru0-fw"
    ls -lh /lib/firmware/am335x-pru0-fw
else
    echo "  ✗ PRU firmware not found"
    echo "  Run: cd pru && make && sudo make install"
fi

echo ""
echo "[3] Device Tree Overlay:"
if [ -f /lib/firmware/PRU-ADC-00A0.dtbo ]; then
    echo "  ✓ Device tree overlay exists"
else
    echo "  ✗ Device tree overlay not found"
fi

echo ""
echo "[4] PRU Shared Memory (requires root):"
if [ "$EUID" -eq 0 ]; then
    echo "  Reading first 64 bytes of PRU shared memory..."
    dd if=/dev/mem bs=1 skip=$((0x4A310000)) count=64 2>/dev/null | hexdump -C
else
    echo "  Skipped (run with sudo to view memory)"
fi

echo ""
echo "[5] Recent Kernel Messages:"
dmesg | grep -i pru | tail -10

echo ""
echo "[6] PRU Control Block (if running as root):"
if [ "$EUID" -eq 0 ]; then
    # Read control block at offset 0x2000
    echo "  Write Index | Read Index | Sample Count | Overrun Count"
    dd if=/dev/mem bs=4 skip=$((0x4A310000/4 + 0x2000/4)) count=4 2>/dev/null | od -An -t u4
else
    echo "  Skipped (run with sudo to view control block)"
fi

echo ""
echo "====================================="
echo "Status Check Complete"
echo "====================================="
