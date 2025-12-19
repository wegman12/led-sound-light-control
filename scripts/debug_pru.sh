#!/bin/bash
#
# PRU IR Detector Debug Script
# Checks PRU status and shared memory to diagnose issues
#

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "======================================"
echo "PRU IR Detector Debug Tool"
echo "======================================"
echo ""

# 1. Check PRU status
echo -e "${BLUE}1. PRU Status:${NC}"
PRU_STATE=$(cat /sys/class/remoteproc/remoteproc1/state 2>/dev/null)
if [ "$PRU_STATE" = "running" ]; then
    echo -e "   ${GREEN}✓ PRU is running${NC}"
else
    echo -e "   ${RED}✗ PRU is NOT running (state: $PRU_STATE)${NC}"
    echo "   Run: echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state"
    exit 1
fi
echo ""

# 2. Check firmware
echo -e "${BLUE}2. Firmware:${NC}"
if [ -f "/lib/firmware/am335x-pru0-fw" ]; then
    FIRMWARE_SIZE=$(stat -c%s "/lib/firmware/am335x-pru0-fw")
    echo -e "   ${GREEN}✓ Firmware exists${NC} (size: $FIRMWARE_SIZE bytes)"
else
    echo -e "   ${RED}✗ Firmware missing${NC}"
    exit 1
fi
echo ""

# 3. Read PRU shared memory control block
# Control block is at offset 0x0800 from base 0x4A310000 = 0x4A310800
echo -e "${BLUE}3. PRU Control Block (0x4A310800):${NC}"
echo "   Reading 6 uint32 values..."
echo "   Format: write_idx read_idx event_cnt error_cnt overrun_cnt status"
echo ""

# Read control block (base 0x4A310000 + offset 0x800 = 0x4A310800)
CONTROL_BLOCK=$(sudo dd if=/dev/mem bs=4 skip=$((0x4A310800/4)) count=6 2>/dev/null | od -An -t u4)

if [ -z "$CONTROL_BLOCK" ]; then
    echo -e "   ${RED}✗ Failed to read shared memory${NC}"
    echo "   Are you running with sudo?"
    exit 1
fi

# Parse values
read -r WRITE_IDX READ_IDX EVENT_CNT ERROR_CNT OVERRUN_CNT STATUS <<< "$CONTROL_BLOCK"

echo "   write_index:   $WRITE_IDX"
echo "   read_index:    $READ_IDX"
echo "   event_count:   $EVENT_CNT"
echo "   error_count:   $ERROR_CNT"
echo "   overrun_count: $OVERRUN_CNT"
echo "   status:        $STATUS"
echo ""

# 4. Analyze control block
echo -e "${BLUE}4. Analysis:${NC}"

# Check status
if [ "$STATUS" -eq 1 ]; then
    echo -e "   ${GREEN}✓ PRU firmware initialized${NC} (status=1)"
else
    echo -e "   ${RED}✗ PRU firmware NOT initialized${NC} (status=$STATUS)"
    echo "   PRU may have crashed or not started properly"
    exit 1
fi

# Check for events
if [ "$EVENT_CNT" -gt 0 ]; then
    echo -e "   ${GREEN}✓ PRU has detected $EVENT_CNT button presses${NC}"

    # Check if Go app is consuming
    BUFFERED=$((WRITE_IDX - READ_IDX))
    if [ "$BUFFERED" -lt 0 ]; then
        BUFFERED=$((256 + BUFFERED))
    fi

    if [ "$BUFFERED" -gt 0 ]; then
        echo -e "   ${YELLOW}⚠ $BUFFERED events in buffer (not consumed by Go app)${NC}"
    else
        echo -e "   ${GREEN}✓ Go app is consuming events${NC}"
    fi
else
    echo -e "   ${YELLOW}⚠ No button presses detected yet${NC}"
    echo "   - Press a button on your IR remote"
    echo "   - If event_count doesn't increase, check IR sensor wiring"
fi

# Check for errors
if [ "$ERROR_CNT" -gt 0 ]; then
    echo -e "   ${RED}✗ $ERROR_CNT decode errors detected${NC}"
    echo "   - IR signal may be noisy or wrong protocol"
    echo "   - Check IR sensor wiring and power"
fi

# Check for overruns
if [ "$OVERRUN_CNT" -gt 0 ]; then
    echo -e "   ${RED}✗ $OVERRUN_CNT buffer overruns${NC}"
    echo "   - Go app not polling fast enough"
fi

echo ""

# 5. Monitor in real-time
echo -e "${BLUE}5. Real-time Monitoring:${NC}"
echo "   Press CTRL+C to stop"
echo ""
echo "   Timestamp       | write | read | events | errors | overruns | buffered"
echo "   ----------------+-------+------+--------+--------+----------+---------"

LAST_EVENT_CNT=$EVENT_CNT

while true; do
    sleep 0.5

    CONTROL_BLOCK=$(sudo dd if=/dev/mem bs=4 skip=$((0x4A310800/4)) count=6 2>/dev/null | od -An -t u4)
    read -r WRITE_IDX READ_IDX EVENT_CNT ERROR_CNT OVERRUN_CNT STATUS <<< "$CONTROL_BLOCK"

    BUFFERED=$((WRITE_IDX - READ_IDX))
    if [ "$BUFFERED" -lt 0 ]; then
        BUFFERED=$((256 + BUFFERED))
    fi

    TIMESTAMP=$(date '+%H:%M:%S.%N' | cut -c1-12)

    # Highlight if event count changed
    if [ "$EVENT_CNT" -gt "$LAST_EVENT_CNT" ]; then
        echo -e "   $TIMESTAMP | ${GREEN}$WRITE_IDX${NC}     | $READ_IDX    | ${GREEN}$EVENT_CNT${NC}      | $ERROR_CNT      | $OVERRUN_CNT         | $BUFFERED"
    else
        echo "   $TIMESTAMP | $WRITE_IDX     | $READ_IDX    | $EVENT_CNT      | $ERROR_CNT      | $OVERRUN_CNT         | $BUFFERED"
    fi

    LAST_EVENT_CNT=$EVENT_CNT
done
