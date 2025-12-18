#!/bin/bash
# Load modified McASP driver and test

set -e

BBB_HOST="bbb2.wegman"
BUILD_DIR="~/mcasp-clkrate-build"

echo "======================================"
echo "Loading Modified McASP Driver"
echo "======================================"
echo

echo "[1/4] Removing old McASP module..."
ssh "$BBB_HOST" "sudo rmmod snd_soc_davinci_mcasp 2>/dev/null || echo 'Module not loaded'"

echo "[2/4] Loading new module with clk_set_rate support..."
ssh "$BBB_HOST" "cd $BUILD_DIR && sudo insmod snd-soc-davinci-mcasp.ko"

echo "[3/4] Checking module loaded..."
ssh "$BBB_HOST" "lsmod | grep davinci_mcasp"

echo "[4/4] Checking dmesg for initialization..."
echo "--------------------------------------"
ssh "$BBB_HOST" "dmesg | tail -30 | grep -i 'mcasp\|clock'"
echo "--------------------------------------"
echo

echo "======================================"
echo "Driver loaded successfully!"
echo "======================================"
echo
echo "Look for messages like:"
echo "  'Set system clock to X Hz (target Y Hz)'"
echo "  'Functional clock not available' (if using fixed clock)"
echo
echo "Next: Test audio recording"
echo "  ssh $BBB_HOST"
echo "  arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 3 test.wav"
echo "  hexdump -C test.wav | head -20"
echo
