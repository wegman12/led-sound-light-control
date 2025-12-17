# I2S Microphone Troubleshooting Guide

## Current Status

### ✅ Working
- Device tree overlay compiles and deploys successfully
- ALSA detects I2S-Microphone (card 0, device 0)
- `arecord -l` shows the capture device
- Device can be opened by arecord
- HDMI audio disabled to free McASP0
- Pin conflict resolved (I2S has priority over PRU cape on P9_30)

### ❌ Not Working
- **I/O Error on Read**: `arecord: pcm_read:2221: read error: Input/output error`
- No audio data captured (only WAV header written, 44 bytes)
- Device tree warning: "IRQ common not found" (may be benign)

---

## Hardware Verification Steps

Since the software configuration appears correct but no data is being received, **hardware verification is the next critical step**.

### 1. Verify Power Supply

**Check INMP441 Power:**
```bash
# On BeagleBone, measure voltage at P9_3 (should be 3.3V)
# With multimeter: Red probe to P9_3, Black probe to P9_1 (GND)
# Expected: ~3.3V DC
```

**Verify INMP441 Module:**
- Some INMP441 modules have an LED that lights when powered
- Check for any signs of physical damage
- Verify VDD pin is making good contact with P9_3

### 2. Verify Clock Generation (Requires Oscilloscope or Logic Analyzer)

**Expected Signals:**

| Pin | Signal | Expected Frequency | Expected Voltage |
|-----|--------|-------------------|------------------|
| P9_31 (SCK) | Bit Clock | 3.072 MHz | 0-3.3V square wave |
| P9_29 (WS) | Word Select | 48 kHz | 0-3.3V square wave |
| P9_30 (SD) | Serial Data | Varies with audio | 0-3.3V |

**Test Procedure:**
```bash
# 1. SSH to BeagleBone
ssh bbb2.wegman

# 2. Start a recording (will error, but should activate clocks)
arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 /tmp/test.wav &
RECORD_PID=$!

# 3. While recording is active, measure signals with scope
#    - P9_31: Should see 3.072 MHz clock
#    - P9_29: Should see 48 kHz clock
#    - P9_30: Should see data transitions (if mic is working)

# 4. Stop recording
kill $RECORD_PID
```

**Interpretation:**
- ✅ **Clocks present**: Hardware config is correct, proceed to Step 3
- ❌ **No clocks**: Pin configuration issue, see "Software Debug" section
- ⚠️ **Wrong frequency**: System clock config issue, adjust DTS

### 3. Test INMP441 Module with Known-Good Setup

**Option A: Test on Different Platform (e.g., Raspberry Pi)**
- Wire INMP441 to Raspberry Pi using same connections
- Use existing RPi I2S drivers to verify microphone works
- If mic works on RPi, issue is BeagleBone-specific

**Option B: Test Different INMP441 Module**
- If you have a spare INMP441, swap it
- Defective modules are not uncommon

### 4. Verify Wiring Continuity

Use multimeter in continuity mode:

| INMP441 Pin | BBB Pin | Expected Continuity |
|-------------|---------|---------------------|
| VDD | P9_3 (3.3V) | Beep/Short |
| GND | P9_1 (GND) | Beep/Short |
| SD | P9_30 | Beep/Short |
| WS | P9_29 | Beep/Short |
| SCK | P9_31 | Beep/Short |
| L/R | P9_1 (GND) | Beep/Short |

**Common Issues:**
- Loose jumper wires
- Bent pins on BeagleBone header
- INMP441 module not fully seated if using socket
- Cold solder joints on INMP441 breakout board

---

## Software Debug (If Hardware is Verified Good)

### Option 1: Try Different Pin Configuration

The INMP441 requires specific I2S timing. Try P9_28 (AXR2) instead of P9_30 (AXR0):

```bash
# Edit device tree overlay
cd /home/kevin/repositories/personal/led-sound-light-control/hardware/device-tree
nano BB-I2S-MIC-00A0.dts

# Change line 31 from:
#   0x198 0x20  /* P9_30: mcasp0_axr0 */
# To:
#   0x19c 0x22  /* P9_28: mcasp0_axr2 (Mode 2, INPUT) */

# Change line 59 from:
#   serial-dir = < 2 0 0 0 ... >;
# To:
#   serial-dir = < 0 0 2 0 ... >;  /* AXR2 as receive */

# Rebuild and deploy
make deploy-full DEPLOY_HOST=bbb2.wegman
ssh bbb2.wegman 'sudo reboot'

# IMPORTANT: Also move physical wire from P9_30 to P9_28
```

**Note**: P9_28 conflicts with PRU cape, so remote control won't work simultaneously.

### Option 2: Adjust TDM Slot Width

Try specifying explicit slot width for 24-bit data:

```c
// In BB-I2S-MIC-00A0.dts, fragment@1
tdm-slots = <2>;
slot-width = <32>;  // Add this line
```

### Option 3: Try Different Sample Rate

```bash
# Test with 16 kHz (simpler clock division)
arecord -D hw:0,0 -f S16_LE -r 16000 -c 1 -d 2 /tmp/test_16k.wav

# Test with 8 kHz
arecord -D hw:0,0 -f S16_LE -r 8000 -c 1 -d 2 /tmp/test_8k.wav
```

### Option 4: Check Kernel Modules

```bash
# Verify required modules are loaded
lsmod | grep snd_soc

# Expected to see:
# - snd_soc_davinci_mcasp
# - snd_soc_simple_card
# - snd_soc_spdif_rx (or spdif_dir)
```

### Option 5: Enable Debug Logging

```bash
# Enable ALSA debug output
echo 1 | sudo tee /proc/asound/card0/pcm0c/xrun_debug

# Enable McASP driver debug
echo 'module davinci_mcasp +p' | sudo tee /sys/kernel/debug/dynamic_debug/control

# Try recording and check dmesg
arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 -d 1 /tmp/test.wav
dmesg | tail -50
```

---

## Known Issues and Limitations

### "IRQ common not found" Warning

```
[   67.937395] davinci-mcasp 48038000.mcasp: IRQ common not found
```

**Impact**: Possibly benign. The driver may be looking for an optional common IRQ that isn't configured in our overlay.

**Potential Fix**: Add interrupt configuration to device tree:

```c
fragment@1 {
    target = <&mcasp0>;
    __overlay__ {
        interrupts = <82>;
        interrupt-parent = <&intc>;
        // ... rest of config
    };
}
```

### Pin Conflict with PRU Cape

P9_30 (PIN102) is claimed by both I2S and PRU cape. Currently I2S has priority, which means:
- ✅ I2S microphone can use P9_30
- ❌ PRU remote control won't load (PIN102 conflict)

**Workaround**: If you need both I2S and PRU remote simultaneously, use P9_28 for I2S data instead.

---

## Diagnostic Commands Reference

```bash
# Check ALSA device
arecord -l

# Check card details
cat /proc/asound/cards

# Test recording with verbose output
arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 -d 2 --dump-hw-params /tmp/test.wav

# Check McASP driver status
dmesg | grep -i mcasp | grep -v mdio

# Check device tree overlay loaded
ls -la /lib/firmware/BB-I2S-MIC-00A0.dtbo
ls -la /boot/dtbs/5.10.168-ti-r83/overlays/BB-I2S-MIC-00A0.dtbo

# Check uEnv.txt configuration
cat /boot/uEnv.txt | grep -E 'overlay|audio'
```

---

## Next Steps

1. **PRIORITY**: Verify hardware with oscilloscope/logic analyzer
   - Confirm 3.3V power at INMP441 VDD
   - Confirm clocks present at P9_29 and P9_31 during recording
   - Check for data transitions at P9_30

2. **If clocks are NOT present**: Pin configuration issue (software)
   - Double-check pinmux offsets in device tree
   - Verify Cape Universal isn't interfering

3. **If clocks ARE present but no data**: Microphone issue (hardware)
   - Test INMP441 on different platform
   - Try different INMP441 module
   - Check for proper L/R pin connection (must be GND)

4. **If everything looks correct**: Advanced debugging
   - Capture I2S signals with logic analyzer
   - Compare timing to INMP441 datasheet requirements
   - Check if McASP is configured for correct I2S variant (Philips vs DSP mode)

---

## Reference Documents

- **INMP441 Datasheet**: https://invensense.tdk.com/products/digital/inmp441/
- **AM335x TRM Chapter 22 (McASP)**: https://www.ti.com/lit/ug/spruh73q/spruh73q.pdf
- **BeagleBone Black SRM**: https://github.com/beagleboard/beaglebone-black/wiki/System-Reference-Manual
- **Linux Simple Audio Card**: https://www.kernel.org/doc/Documentation/devicetree/bindings/sound/simple-card.txt

---

**Created**: 2025-12-17
**Project**: led-sound-light-control
**Branch**: feature/i2s-microphone
