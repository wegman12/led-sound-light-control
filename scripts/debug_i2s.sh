#!/bin/bash
#
# PRU I2S Audio Debug Script
# Checks PRU1, McASP, and I2S audio configuration
#

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo "======================================"
echo "PRU I2S Audio Debug Tool"
echo "======================================"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script requires root privileges${NC}"
    echo "Run: sudo $0"
    exit 1
fi

# 1. Check PRU1 status (remoteproc2)
echo -e "${BLUE}1. PRU1 Status:${NC}"
PRU1_STATE=$(cat /sys/class/remoteproc/remoteproc2/state 2>/dev/null)
if [ "$PRU1_STATE" = "running" ]; then
    echo -e "   ${GREEN}✓ PRU1 is running${NC}"
else
    echo -e "   ${RED}✗ PRU1 is NOT running (state: $PRU1_STATE)${NC}"
    echo "   Run: echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"
fi
echo ""

# 2. Check firmware
echo -e "${BLUE}2. PRU1 Firmware:${NC}"
if [ -f "/lib/firmware/am335x-pru1-fw" ]; then
    FIRMWARE_SIZE=$(stat -c%s "/lib/firmware/am335x-pru1-fw")
    echo -e "   ${GREEN}✓ Firmware exists${NC} (size: $FIRMWARE_SIZE bytes)"
else
    echo -e "   ${RED}✗ Firmware missing${NC}"
    echo "   Deploy with: cd pru/audio && make i2s-deploy-all"
fi
echo ""

# 3. Check device tree overlay
echo -e "${BLUE}3. Device Tree Overlay:${NC}"
if [ -d "/proc/device-tree/chosen/overlays" ]; then
    if ls /proc/device-tree/chosen/overlays/ 2>/dev/null | grep -q "BB-I2S-PRU"; then
        echo -e "   ${GREEN}✓ BB-I2S-PRU overlay loaded${NC}"
    else
        echo -e "   ${YELLOW}⚠ BB-I2S-PRU overlay not found${NC}"
        echo "   Available overlays:"
        ls /proc/device-tree/chosen/overlays/ 2>/dev/null | head -5 | sed 's/^/      /'
        echo "   Load with: sudo config-pin overlay BB-I2S-PRU-00A0"
    fi
else
    echo -e "   ${YELLOW}⚠ Cannot check overlays (directory not found)${NC}"
fi
echo ""

# 4. Check pin configuration
echo -e "${BLUE}4. Pin Configuration:${NC}"
echo "   Checking I2S pins on P9 header..."

check_pin() {
    PIN=$1
    EXPECTED=$2
    DESC=$3

    if command -v config-pin &> /dev/null; then
        MODE=$(config-pin -q $PIN 2>/dev/null | grep -o "Mode:.*" || echo "Mode: unknown")
        echo "   $PIN: $DESC"
        echo "      Current: $MODE"
        echo "      Expected: $EXPECTED"
    else
        echo "   $PIN: $DESC (config-pin not available)"
    fi
}

echo ""
echo "   P9_25: CLKOUT2 (24.576 MHz clock output) -> jumper to P9_28"
echo "   P9_28: McASP0_AHCLKR (24.576 MHz clock input from P9_25)"
echo "   P9_29: McASP0_FSX (48 kHz frame sync output)"
echo "   P9_30: McASP0_AXR0 (I2S data input from mic)"
echo "   P9_31: McASP0_ACLKX (3.072 MHz bit clock output)"
echo ""
echo -e "   ${CYAN}IMPORTANT: Ensure jumper wire connects P9_25 to P9_28${NC}"
echo ""

# 5. Read PRU1 audio control block
echo -e "${BLUE}5. PRU1 Audio Control Block (0x4A310900):${NC}"

# Control block is at offset 0x0900 from base 0x4A310000
CONTROL_ADDR=$((0x4A310900))

# Read status code (first uint32)
STATUS=$(dd if=/dev/mem bs=4 skip=$((CONTROL_ADDR/4)) count=1 2>/dev/null | od -An -t x4 | tr -d ' ')
STATUS_DEC=$(printf "%d" 0x$STATUS 2>/dev/null || echo "0")

# Decode status
echo -n "   Status: 0x$STATUS "
case "$STATUS" in
    "49325331")
        echo -e "${GREEN}(I2S1 - I2S firmware running, 48 kHz)${NC}"
        FIRMWARE_MODE="I2S"
        ;;
    "41554431")
        echo -e "${GREEN}(AUD1 - ADC firmware running, 40 kHz)${NC}"
        FIRMWARE_MODE="ADC"
        ;;
    "4d435350")
        echo -e "${YELLOW}(MCSP - McASP initialization)${NC}"
        FIRMWARE_MODE="I2S"
        ;;
    "53414d50")
        echo -e "${GREEN}(SAMP - Sampling active)${NC}"
        FIRMWARE_MODE="I2S"
        ;;
    "46465450")
        echo -e "${CYAN}(FFTP - FFT processing)${NC}"
        FIRMWARE_MODE="I2S"
        ;;
    "45525221")
        echo -e "${RED}(ERR! - Error occurred)${NC}"
        FIRMWARE_MODE="ERROR"
        ;;
    "00000000")
        echo -e "${RED}(Not running or not initialized)${NC}"
        FIRMWARE_MODE="NONE"
        ;;
    *)
        echo -e "${YELLOW}(Unknown: $STATUS)${NC}"
        FIRMWARE_MODE="UNKNOWN"
        ;;
esac
echo ""

# 6. Read more control block fields
echo -e "${BLUE}6. Audio Statistics:${NC}"

# Read 20 uint32 values from control block
CTRL_DATA=$(dd if=/dev/mem bs=4 skip=$((CONTROL_ADDR/4)) count=20 2>/dev/null | od -An -t u4)
read -r -a VALUES <<< "$CTRL_DATA"

if [ "${#VALUES[@]}" -ge 15 ]; then
    echo "   FFT Enable:       ${VALUES[0]}"
    echo "   Bass Max Hz:      ${VALUES[1]}"
    echo "   MidLow Max Hz:    ${VALUES[2]}"
    echo "   MidHigh Max Hz:   ${VALUES[3]}"
    echo "   Smoothing Alpha:  ${VALUES[4]}/1000"
    echo "   Status:           ${VALUES[5]} (0x$(printf '%08x' ${VALUES[5]}))"
    echo "   Total Samples:    ${VALUES[6]}"
    echo "   Buffer Count:     ${VALUES[7]}"
    echo "   Current Buffer:   ${VALUES[8]}"
    echo "   Samples in Buf:   ${VALUES[9]}"
    echo "   McASP Errors:     ${VALUES[10]}"
    echo "   Missed Samples:   ${VALUES[11]}"
    echo "   Last Sample:      ${VALUES[12]}"
    echo "   Min Sample:       ${VALUES[13]}"
    echo "   Max Sample:       ${VALUES[14]}"

    if [ "${#VALUES[@]}" -ge 20 ]; then
        echo "   FFT Count:        ${VALUES[15]}"
        echo "   FFT Time (cycles):${VALUES[16]}"
        echo "   FFT Skipped:      ${VALUES[17]}"

        # Calculate approximate FFT rate
        if [ "${VALUES[15]}" -gt 0 ] && [ "${VALUES[7]}" -gt 0 ]; then
            FFT_PER_BUFFER=$((VALUES[15] * 100 / VALUES[7]))
            echo "   FFT/Buffer:       $((FFT_PER_BUFFER / 100)).$((FFT_PER_BUFFER % 100))"
        fi
    fi
else
    echo -e "   ${RED}Failed to read control block${NC}"
fi
echo ""

# 7. McASP Register Check (if accessible)
echo -e "${BLUE}7. McASP0 Registers (0x48038000):${NC}"
MCASP_BASE=$((0x48038000))

# Try to read McASP revision register
MCASP_REV=$(dd if=/dev/mem bs=4 skip=$((MCASP_BASE/4)) count=1 2>/dev/null | od -An -t x4 | tr -d ' ')
if [ -n "$MCASP_REV" ] && [ "$MCASP_REV" != "00000000" ]; then
    echo "   Revision:     0x$MCASP_REV"

    # Read GBLCTL
    GBLCTL=$(dd if=/dev/mem bs=4 skip=$((($MCASP_BASE + 0x44)/4)) count=1 2>/dev/null | od -An -t x4 | tr -d ' ')
    echo "   GBLCTL:       0x$GBLCTL"

    # Read RSTAT
    RSTAT=$(dd if=/dev/mem bs=4 skip=$((($MCASP_BASE + 0x80)/4)) count=1 2>/dev/null | od -An -t x4 | tr -d ' ')
    echo "   RSTAT:        0x$RSTAT"

    # Decode RSTAT bits
    RSTAT_DEC=$(printf "%d" 0x$RSTAT 2>/dev/null || echo "0")
    if [ $((RSTAT_DEC & 0x20)) -ne 0 ]; then
        echo -e "     ${GREEN}RDATA: Data ready${NC}"
    fi
    if [ $((RSTAT_DEC & 0x01)) -ne 0 ]; then
        echo -e "     ${RED}ROVRN: Overrun error${NC}"
    fi
    if [ $((RSTAT_DEC & 0x02)) -ne 0 ]; then
        echo -e "     ${RED}RSYNCERR: Sync error${NC}"
    fi

    # Read SRCTL0
    SRCTL0=$(dd if=/dev/mem bs=4 skip=$((($MCASP_BASE + 0x180)/4)) count=1 2>/dev/null | od -An -t x4 | tr -d ' ')
    echo "   SRCTL0:       0x$SRCTL0"

    # Decode serializer mode
    SRCTL0_DEC=$(printf "%d" 0x$SRCTL0 2>/dev/null || echo "0")
    SRMOD=$((SRCTL0_DEC & 0x03))
    case $SRMOD in
        0) echo "     Serializer mode: Inactive" ;;
        1) echo "     Serializer mode: Transmit" ;;
        2) echo -e "     ${GREEN}Serializer mode: Receive${NC}" ;;
        3) echo "     Serializer mode: Reserved" ;;
    esac
else
    echo -e "   ${YELLOW}Cannot read McASP registers (may need kernel module unloaded)${NC}"
fi
echo ""

# 8. Real-time monitoring option
echo -e "${BLUE}8. Real-time Monitoring:${NC}"
echo "   To monitor in real-time, run:"
echo "   sudo ./audio-util stream"
echo ""
echo "   Or watch control block changes:"
echo "   watch -n 0.5 'dd if=/dev/mem bs=4 skip=\$((0x4A310900/4)) count=20 2>/dev/null | od -An -t u4'"
echo ""

# Summary
echo "======================================"
echo -e "${BLUE}Summary:${NC}"
if [ "$PRU1_STATE" = "running" ] && [ "$FIRMWARE_MODE" = "I2S" ]; then
    echo -e "${GREEN}PRU1 I2S audio is running${NC}"
    if [ "${VALUES[6]}" -gt 0 ] 2>/dev/null; then
        echo -e "${GREEN}Samples are being collected (${VALUES[6]} total)${NC}"
    else
        echo -e "${YELLOW}No samples yet - check microphone connection${NC}"
    fi
elif [ "$FIRMWARE_MODE" = "ADC" ]; then
    echo -e "${YELLOW}ADC firmware running - deploy I2S firmware with: make i2s-deploy-all${NC}"
else
    echo -e "${RED}PRU1 audio not operational${NC}"
    echo "Check:"
    echo "  1. PRU1 is started"
    echo "  2. I2S firmware is deployed"
    echo "  3. Device tree overlay is loaded"
    echo "  4. Jumper wire connects P9_25 to P9_28"
fi
echo "======================================"
