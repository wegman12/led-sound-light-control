#!/bin/bash
# Debug PRU sampling issues

echo "====================================="
echo "PRU Debug Information"
echo "====================================="
echo ""

echo "[1] PRU State:"
if [ -d /sys/class/remoteproc/remoteproc1 ]; then
    echo "  State: $(cat /sys/class/remoteproc/remoteproc1/state)"
    echo "  Name: $(cat /sys/class/remoteproc/remoteproc1/name 2>/dev/null || echo 'N/A')"
else
    echo "  ERROR: remoteproc1 not found"
    echo "  Searching for PRU remoteproc..."
    ls -la /sys/class/remoteproc/
fi

echo ""
echo "[2] PRU Firmware:"
if [ -f /lib/firmware/am335x-pru0-fw ]; then
    ls -lh /lib/firmware/am335x-pru0-fw
    echo "  MD5: $(md5sum /lib/firmware/am335x-pru0-fw | awk '{print $1}')"
else
    echo "  ERROR: Firmware not found at /lib/firmware/am335x-pru0-fw"
fi

echo ""
echo "[3] Kernel Messages (last 20 PRU-related):"
dmesg | grep -i pru | tail -20

echo ""
echo "[4] PRU Shared Memory Control Block:"
if [ "$EUID" -eq 0 ]; then
    echo "  Reading control block at 0x4A312000..."
    echo "  Format: write_index | read_index | sample_count | overrun_count"
    dd if=/dev/mem bs=4 skip=$((0x4A310000/4 + 0x2000/4)) count=4 2>/dev/null | od -An -t u4 | head -1
else
    echo "  (Run with sudo to view memory)"
fi

echo ""
echo "[5] First 64 bytes of PRU Shared Memory:"
if [ "$EUID" -eq 0 ]; then
    dd if=/dev/mem bs=1 skip=$((0x4A310000)) count=64 2>/dev/null | hexdump -C
else
    echo "  (Run with sudo to view memory)"
fi

echo ""
echo "[6] Check if PRU ICSS is enabled:"
if [ -d /sys/devices/platform/ocp/4a300000.pruss ]; then
    echo "  ✓ PRU ICSS found at 4a300000.pruss"
elif [ -d /sys/devices/platform/ocp/4a300000.pruss-soc-bus ]; then
    echo "  ✓ PRU ICSS found at 4a300000.pruss-soc-bus"
else
    echo "  Searching for PRU in device tree..."
    ls -la /sys/devices/platform/ocp/ | grep -i pru
fi

echo ""
echo "[7] ADC Status:"
if [ -d /sys/bus/iio/devices/iio:device0 ]; then
    echo "  ✓ ADC device found"
    echo "  Name: $(cat /sys/bus/iio/devices/iio:device0/name 2>/dev/null || echo 'N/A')"
else
    echo "  WARNING: ADC device not found"
fi

echo ""
echo "====================================="
echo "Debug Complete"
echo "====================================="
