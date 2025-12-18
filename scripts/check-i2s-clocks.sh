#!/bin/bash
# Check I2S clock status after loading overlay

BBB_HOST="bbb2.wegman"

echo "======================================"
echo "Checking I2S Clock Configuration"
echo "======================================"
echo

echo "[1] Checking if overlay loaded..."
ssh "$BBB_HOST" "cat /proc/device-tree/chosen/overlays/name 2>/dev/null | tr '\0' '\n' | grep -i i2s" || echo "Overlay info not available in /proc/device-tree"

echo
echo "[2] Checking dmesg for McASP initialization..."
ssh "$BBB_HOST" "dmesg | grep -i 'mcasp\|simple-card' | tail -20"

echo
echo "[3] Checking for audio devices..."
ssh "$BBB_HOST" "arecord -l"

echo
echo "[4] Checking McASP clock registers..."
ssh "$BBB_HOST" "sudo cat /sys/kernel/debug/davinci_mcasp/48038000.mcasp/regs 2>/dev/null | grep -E 'ACLKXCTL|AHCLKXCTL|PDIR|GBLCTL'" || echo "Debug registers not available (module not loaded?)"

echo
echo "[5] Attempting to start audio stream (will show register state)..."
ssh "$BBB_HOST" "timeout 2 arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 /dev/null 2>&1 || true"

echo
echo "[6] Checking register state after attempted recording..."
ssh "$BBB_HOST" "sudo cat /sys/kernel/debug/davinci_mcasp/48038000.mcasp/regs 2>/dev/null | grep -E 'ACLKXCTL|AHCLKXCTL|PDIR|GBLCTL'" || echo "Debug registers not available"

echo
echo "======================================"
echo "Analysis:"
echo "======================================"
echo "Look for the following in the output above:"
echo "  1. McASP probe messages showing successful initialization"
echo "  2. Audio device 'I2S-Microphone' listed in arecord -l"
echo "  3. ACLKXCTL register should show bit 5 (ACLKXE) set"
echo "  4. PDIR should show ACLKX and FSX configured as outputs"
echo "  5. Check if clock dividers were calculated correctly"
echo
echo "Current test: 24 MHz system clock (mcasp0_fck default)"
echo "Expected: May not generate clocks correctly due to divider limitations"
echo
echo "If clocks still don't appear:"
echo "  - This confirms the driver limitation"
echo "  - Next step: Modify driver to add clk_set_rate() support"
echo "  - Alternative: Use external 24.576 MHz oscillator"
echo
