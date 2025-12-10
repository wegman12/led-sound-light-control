# Next Steps - P9_31 Migration

## Current Status (2025-12-10)

### What's Working
- ✅ Device tree overlay loaded successfully after reboot
- ✅ Custom-pins overlay present in `/proc/device-tree/chosen/overlays/`
- ✅ PRU firmware compiled and deployed with P9_31 direct GPIO code
- ✅ PRU running (status: 1)
- ✅ `__R31` register access confirmed working

### Current Issue - P9_31 Reading Continuous LOW

**Symptom**: PRU monitoring shows:
```
events: 0    (full __R31 register value)
errors: 0    (bit 0 specifically)
```

**Problem**: P9_31 is reading **continuous LOW (0)** but an idle IR receiver should read **HIGH (1)**.

**Why this matters**: The PRU waits for GPIO LOW to trigger packet reading. If the pin is stuck at LOW, it will immediately try to read a packet, fail, and the error_count would increment. Since error_count=0, the pin appears stuck at 0 with no transitions.

### Root Cause Investigation Needed

Three possible causes:

1. **Hardware Connection Issue**
   - IR receiver might not be physically connected to P9_31
   - Still connected to old P9_41 location?
   - Check: Use multimeter to measure voltage on P9_31 pin

2. **IR Receiver Power Issue**
   - If receiver isn't powered, output could float or pull low
   - Check: Verify 3.3V power and ground connections to receiver

3. **Pin Mapping Error**
   - P9_31 might not map to PRU0 __R31 bit 0 as expected
   - Device tree shows: `0x190 0x36  /* mcasp0_aclkx.pr1_pru0_pru_r31_0 */`
   - Comment says MODE6, but we should verify this is correct

## Next Steps

### 1. Verify Hardware Connection (IMMEDIATE)

**Check physical wiring**:
```bash
# Measure voltage on P9_31 (should be ~3.3V in idle state)
# Measure on the physical pin with multimeter
```

**Questions to answer**:
- Is IR receiver signal wire soldered to P9_31 (pin 31 on P9 header)?
- Is IR receiver powered (VCC connected to 3.3V)?
- Is IR receiver ground connected?

### 2. Test with pru-gpio-reader (if hardware is correct)

If the hardware connection is verified, deploy the raw GPIO debug tool:

```bash
cd api
make deploy-pru-utils

# On BeagleBone, run:
sudo ~/led-sound-light-control/pru-gpio-reader
# Press IR button and check if signal toggles between 0 and 1
```

### 3. Verify Pin Mapping (if signal still stuck)

Check BeagleBone Black PRU pin mapping documentation to confirm:
- Does P9_31 actually map to PRU0 __R31 bit 0 in MODE6?
- Reference: https://beagleboard.org/static/prumaps/index.html

Alternative pins to try if P9_31 is wrong:
- P9_27: MODE5 = pr1_pru0_pru_r31_5
- P9_25: MODE5 = pr1_pru0_pru_r31_7

### 4. Fallback Options

If P9_31 direct GPIO doesn't work:

**Option A**: Try P9_27 or P9_25 (different PRU input bits)
- Update device tree overlay with new offset
- Update PRU code to read different bit in `__R31`

**Option B**: Return to memory-mapped GPIO
- Revert to GPIO peripheral register access
- Fix the struct alignment issue with durations array
- Use explicit volatile reads (already fixed)

## Code Changes Made Today

### pru/pru0_ir_detector.c
- Added `__R31` debug logging at line 473-474:
  ```c
  ctrl->event_count = __R31;           // Full register value
  ctrl->error_count = (__R31 & 0x1);   // Bit 0 specifically
  ```
- This should be **removed** once hardware issue is resolved

### dto/custom-pins.dts
- Configured for P9_31: `0x190 0x36` (INPUT | PULLUP | MODE6)
- Deployed to `/boot/dtbs/5.10.168-ti-r83/overlays/custom-pins.dtbo`

## Files Reference

**PRU Firmware**: `pru/pru0_ir_detector.c`
- Line 126: `return (__R31 & 0x1);` - reads P9_31 via PRU direct GPIO
- Line 465-491: `run_ir_detection_loop()` - main detection logic
- Line 473-474: DEBUG code writing __R31 values (TO BE REMOVED)

**Device Tree Overlay**: `dto/custom-pins.dts`
- Line 19: P9_31 configuration

**Debug Utilities**:
- `api/cmd/pru-debug.go` - Monitor control block in real-time
- `api/cmd/pru-gpio-reader.go` - Capture raw GPIO samples
- `api/cmd/pru-bits-reader.go` - Read decoded packet bits

## Build & Deploy Commands

```bash
# From api/ directory:
make deploy-pru              # Build and deploy PRU firmware
make deploy-pru-utils        # Deploy debug utilities
make help                    # Show all targets

# On BeagleBone:
sudo ~/led-sound-light-control/pru-debug         # Monitor PRU status
sudo ~/led-sound-light-control/pru-gpio-reader   # Capture raw GPIO
sudo ./led-sound-light-control test-remote       # Test button detection
```

## Background Context

### Why P9_31?
- PRU can directly access certain GPIO pins via `__R31` hardware register
- Single instruction read, no memory-mapped I/O overhead
- Eliminates struct alignment issues from previous duration measurement attempts
- Much faster and more reliable than peripheral register access

### Previous State (P9_41)
- Used memory-mapped GPIO0_20 via OCP bus
- Required explicit volatile reads to avoid compiler caching
- Struct alignment mismatch prevented duration measurements
- Working for basic detection but limited by hardware access latency

### Migration Commits
- "Migrate from memory-mapped GPIO to PRU direct register access (P9_31)"
- "Update device tree overlay for P9_31 PRU input"
- "Add __R30 and __R31 register declarations for TI PRU compiler"

## Session Notes

- Overlay loaded successfully after reboot
- PRU firmware compilation successful with `__R31` access
- No errors in dmesg related to overlay
- Clean PRU state (events=0, errors=0) suggests no signal detection
- Debug code confirms `__R31` reading works, but returns 0 continuously
