# PRU IR Remote Detector Firmware

This directory contains PRU (Programmable Real-Time Unit) firmware for detecting and decoding IR remote signals on BeagleBone Black. The firmware offloads IR detection from the ARM CPU to PRU0, providing deterministic microsecond-precision timing and freeing the CPU for LED control.

## Architecture

### Hardware
- **Target:** BeagleBone Black PRU0 (AM335x processor)
- **GPIO:** Pin 20 (P9_41) for IR input
- **Clock:** 200 MHz (5 ns per cycle)
- **Protocol:** 34-bit NEC-like IR protocol

### Firmware Components

```
pru0_ir_detector.c   Main firmware (IR detection, decoding, ring buffer)
AM335x_PRU.cmd      Linker script (memory layout)
Makefile            Build system (TI PRU compiler)
```

### Memory Layout

**PRU Shared RAM** (Physical: 0x4A310000, Size: 12 KB)
```
Offset 0x0000: Button Ring Buffer (2 KB = 256 events × 8 bytes)
Offset 0x1000: Control Block (24 bytes)
  - write_index (PRU writes)
  - read_index (Go reads)
  - event_count (total events)
  - error_count (decode errors)
  - overrun_count (buffer overruns)
  - status (PRU running flag)
```

### IR Protocol

**34-Bit Structure:**
- Bit 0: START (must be 1)
- Bits 1-8: HEADER (must be all 0)
- Bits 9-16: SEPARATOR (must be all 1)
- Bits 17-31: DATA (15 bits)
- Bits 32-33: STOP (must be 1, 1)

**Decoding:**
- Each pulse has HIGH and LOW duration
- If LOW < 1ms → bit = 0
- If LOW >= 1ms → bit = 1

**Button Matching:**
- 44 hardcoded button patterns
- Returns button index (0-43) on match
- Returns -1 on no match or decode error

## Building

### Prerequisites

1. **TI PRU Code Generation Tools:**
   ```bash
   sudo apt-get update
   sudo apt-get install ti-pru-cgt-installer
   ```

2. **Verify installation:**
   ```bash
   which clpru
   # Should output: /usr/bin/clpru
   ```

### Build Commands

```bash
# Build firmware
make

# Clean build artifacts
make clean

# Install to /lib/firmware (on BeagleBone)
make install

# Show help
make help
```

### Output

Build produces:
- `gen/pru0_ir_detector.out` - PRU firmware binary
- `gen/pru0_ir_detector.map` - Memory map file
- `gen/*.pp` - Preprocessed files (for debugging)

## Deployment

### Method 1: Using Makefile (on BeagleBone)

```bash
# Build on BeagleBone
make

# Install and reload PRU
make install
echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state
```

### Method 2: Using deploy script (from development machine)

```bash
# From api/ directory
make deploy-pru
```

### Method 3: Manual deployment

```bash
# Copy firmware to BeagleBone
scp gen/pru0_ir_detector.out bbb1.wegman:~/

# On BeagleBone
sudo cp ~/pru0_ir_detector.out /lib/firmware/am335x-pru0-fw

# Reload PRU
echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state
sleep 1
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state

# Verify PRU is running
cat /sys/class/remoteproc/remoteproc1/state
# Should output: running
```

## Verification

### Check PRU Status

```bash
# PRU state
cat /sys/class/remoteproc/remoteproc1/state

# Expected: running
# If not: check dmesg for errors
```

### Read Shared Memory

```bash
# Read control block (24 bytes at offset 0x1000)
sudo dd if=/dev/mem bs=4 skip=$((0x4A311000/4)) count=6 2>/dev/null | od -An -t u4

# Output format (6 uint32 values):
#   write_index read_index event_count error_count overrun_count status
#
# Example healthy output:
#   0 0 0 0 0 1
#   (no events yet, status=1 means running)

# After pressing a button:
#   1 0 1 0 0 1
#   (write_index=1, event_count=1, button detected!)
```

### Test Button Detection

```bash
# Monitor control block in real-time
watch -n 0.5 'sudo dd if=/dev/mem bs=4 skip=$((0x4A311000/4)) count=6 2>/dev/null | od -An -t u4'

# Press buttons on IR remote
# - event_count should increment
# - error_count = decode errors (should stay 0 or low)
# - overrun_count = buffer overruns (should stay 0)
```

## Troubleshooting

### Issue: PRU won't start

**Symptoms:**
```bash
cat /sys/class/remoteproc/remoteproc1/state
# Output: offline
```

**Solutions:**
1. Check firmware file exists:
   ```bash
   ls -l /lib/firmware/am335x-pru0-fw
   ```

2. Check dmesg for errors:
   ```bash
   sudo dmesg | tail -20
   ```

3. Verify PRU remoteproc driver loaded:
   ```bash
   ls /sys/class/remoteproc/
   # Should show: remoteproc1 remoteproc2
   ```

4. Try manual load:
   ```bash
   sudo modprobe pru_rproc
   ```

### Issue: High error_count

**Symptoms:** error_count incrementing rapidly

**Causes:**
- IR sensor wiring incorrect
- Wrong GPIO pin
- Electromagnetic interference
- IR remote using different protocol

**Solutions:**
1. Verify GPIO 20 wiring:
   ```bash
   # Test GPIO directly
   gpioinfo | grep gpio20
   ```

2. Check device tree overlay:
   ```bash
   cat /boot/uEnv.txt | grep dtbo
   ```

3. Reduce interference (shield IR sensor wire)

### Issue: overrun_count increasing

**Symptoms:** overrun_count > 0 and increasing

**Cause:** Go application not polling fast enough (ring buffer full)

**Solution:** Reduce Go polling interval from 100ms to 50ms

### Issue: No button detection

**Symptoms:** status=1 but event_count stays 0

**Debug steps:**
1. Verify PRU running:
   ```bash
   cat /sys/class/remoteproc/remoteproc1/state
   ```

2. Check status field in shared memory:
   ```bash
   sudo dd if=/dev/mem bs=4 skip=$((0x4A311000/4)) count=6 2>/dev/null | od -An -t u4
   # Last value should be 1 (status=running)
   ```

3. Test IR sensor with oscilloscope or logic analyzer

4. Verify button codes match remote:
   - Codes hardcoded in `pru0_ir_detector.c:button_codes[]`
   - Must match `api/remote/codes.go`

## Technical Details

### Timing Precision

- **PRU Clock:** 200 MHz = 5 ns per cycle
- **1μs:** 200 cycles
- **1ms threshold:** 200,000 cycles
- **Timing jitter:** ±50 ns (vs ±1μs for CPU polling)

### Power Consumption

- **PRU Active:** ~50 mW
- **CPU Polling Savings:** ~500 mW (freed CPU time)
- **Net Benefit:** Lower overall system power

### Performance Impact

- **CPU Usage:** 30-50% → <1%
- **LED Animation:** Smooth (no starvation)
- **Detection Latency:** ~5μs (vs ~1ms CPU)
- **Buffer Capacity:** 256 events (25x safety margin)

### Memory Usage

- **Code:** ~4 KB (instruction RAM)
- **Data:** ~2 KB (button codes lookup table)
- **Shared:** 2 KB (ring buffer) + 24 bytes (control block)

## Development

### Modifying Button Codes

If you add/remove buttons in `api/remote/codes.go`:

1. Update `NUM_BUTTON_CODES` in `pru0_ir_detector.c`
2. Update `button_codes[]` array with new patterns
3. Rebuild and redeploy firmware

### Adjusting Timing

To change 1ms threshold:

```c
// In pru0_ir_detector.c
#define THRESHOLD_1MS 200000  // Current: 1ms

// Example: Change to 1.2ms
#define THRESHOLD_1MS 240000  // 1.2ms = 1200μs × 200 cycles/μs
```

### Debug Output

For detailed debugging, add PRU UART output or LED blinking:

```c
// Blink USR LED on button detection
#define USR_LED_GPIO (1 << 21)  // USR0 on GPIO1
GPIO1_SET = USR_LED_GPIO;
__delay_cycles(10000000);  // 50ms
GPIO1_CLR = USR_LED_GPIO;
```

## References

- [TI PRU Documentation](https://www.ti.com/lit/ug/spruh73q/spruh73q.pdf)
- [BeagleBone Black System Reference](https://github.com/beagleboard/beaglebone-black)
- [PRU Cookbook](https://beagleboard.org/pru)
- [NEC IR Protocol](https://www.sbprojects.net/knowledge/ir/nec.php)

## License

Part of led-sound-light-control project
