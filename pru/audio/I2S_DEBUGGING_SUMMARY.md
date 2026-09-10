# PRU I2S Audio Capture - Debugging Summary

## Overview

This document summarizes the debugging process to get PRU1 I2S audio capture working with an INMP441 MEMS microphone on BeagleBone Black.

## Final Working Configuration

### Hardware
- INMP441 I2S MEMS Microphone connected to P9 header
- Jumper wire: P9_25 -> P9_28 (routes 24.576 MHz clock to McASP - optional with internal clock mode)

### Pin Assignments
| Pin   | Signal          | Function                    |
|-------|-----------------|----------------------------|
| P9_25 | CLKOUT2         | 24.576 MHz clock output    |
| P9_28 | McASP0_AHCLKR   | High-freq clock input      |
| P9_29 | McASP0_FSX      | Frame sync (LRCLK) output  |
| P9_30 | McASP0_AXR0     | Serial data input from mic |
| P9_31 | McASP0_ACLKX    | Bit clock (BCLK) output    |

### Key Configuration Values
- GBLCTL = 0x131F (RX clocks, serializer, state machine, frame sync enabled)
- PDIR = 0xB4000000 (AFSR, ACLKR, AFSX, ACLKX as outputs)
- ACLKRCTL = 0xA7 (internal clock, div by 8, inverted polarity)
- AHCLKRCTL = 0x8000 (internal high-freq clock mode)
- RFMT = 0x180F0 (32-bit slots, 1-bit delay, LSB first)
- AFSRCTL = 0x113 (2 TDM slots, word-wide frame sync, internal, inverted)

## Key Discoveries

### 1. AFIFO Register Location (Critical Fix)
The AFIFO registers are at **CFG_BASE + 0x1000** (0x48039xxx), NOT DATA_BASE + 0x1000 (0x46001xxx).

- RFIFOCTL at 0x48039008 - FIFO control (enable, thresholds)
- RFIFOSTS at 0x4803900C - FIFO status (fill level)

This was the critical bug - the PRU was writing to the wrong address.

### 2. Data Path
The kernel driver uses DMA with AFIFO, reading from the data port at 0x46000000.
Polling RBUF registers (0x48038280) does NOT work - data only flows through AFIFO to data port.

Working data path:
1. Enable AFIFO via RFIFOCTL (bit 16 = RENA)
2. Poll RFIFOSTS for FIFO level > 0
3. Read samples from data port at 0x46000000

### 3. Pin Mux with Disabled Driver
When the McASP kernel driver is disabled (status = "disabled"), the standard pinctrl approach doesn't apply pin mux. Solution: use `bone-pinmux-helper` compatible device under the ocp node:

```dts
fragment@1 {
    target = <&ocp>;
    __overlay__ {
        i2s_pru_pinmux_helper {
            compatible = "bone-pinmux-helper";
            pinctrl-names = "default";
            pinctrl-0 = <&i2s_pru_pins>;
            status = "okay";
        };
    };
};
```

### 4. AHCLK Re-enable Bug
Hardware clears AHCLKXE/AHCLKRE bits when releasing clock dividers from reset. Must re-enable after GBLCTL clock reset sequence. This fix came from analyzing the kernel driver patch.

### 5. Internal Clock Mode
Using internal clock mode (HCLKRM=1 in AHCLKRCTL) works with the onboard 24 MHz mcasp0_fck. This gives approximately:
- 24 MHz / 8 = 3 MHz bit clock
- 3 MHz / 64 = 46.875 kHz sample rate

## Current Status

**Working:**
- Audio samples flowing at ~36 kHz
- FFT processing running at ~35 Hz
- Real frequency band data (Bass, MidLow, MidHigh, Treble)
- Device tree overlay applying pin mux correctly

**Remaining Issues:**
- Sample rate lower than expected (36 kHz vs 47 kHz)
- ROVRN (overrun) and RCKFAIL errors in RSTAT (may be transient)

## Files Modified

1. `pru/audio/am335x_mcasp.h` - Added AFIFO register definitions and corrected documentation
2. `pru/audio/pru1_audio_i2s.c` - Fixed FIFO enable and data reading
3. `hardware/device-tree/BB-I2S-PRU-00A0.dts` - Use bone-pinmux-helper
4. `hardware/device-tree/uEnv.txt` - Updated overlay configuration
5. `pru/audio/scripts/cmd/test.go` - Updated status code handling

## Testing Commands

```bash
# Deploy firmware
cd pru/audio && make i2s-deploy-all DEPLOY_HOST=bbb1.wegman

# Test on BeagleBone
ssh bbb1.wegman
cd ~/led-sound-light-control/pru-audio/
sudo ./audio-util test
```

## Next Steps

1. Investigate sample rate discrepancy (36 kHz vs expected ~47 kHz)
2. Consider if FIFO threshold tuning helps (currently RNUMEVT=1)
3. Clear ROVRN/RCKFAIL errors and monitor if they recur
4. Test with external 24.576 MHz clock (via P9_25 -> P9_28 jumper) for exact 48 kHz
