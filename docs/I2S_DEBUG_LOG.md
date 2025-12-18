# I2S INMP441 Debugging Log

**Date Started**: 2025-12-17
**Device**: BeagleBone Black + INMP441 I2S MEMS Microphone
**Issue**: arecord fails with I/O error, no audio data captured

---

## Hardware Verification

| Check | Status | Result | Notes |
|-------|--------|--------|-------|
| Wiring physical inspection | ✅ PASS | 100% correct | Photo verified VDD→P9_3, GND→P9_1, BCLK→P9_31, DOUT→P9_30, LRCL→P9_29, SEL→GND |
| Power supply (3.3V) | ✅ PASS | 3.3V measured | User confirmed with multimeter |
| Pin configuration | ✅ PASS | All pins in Mode 0 | PIN101: 0x00, PIN102: 0x20, PIN103: 0x00 (McASP mode) |

---

## Device Tree Configuration

| Iteration | Change | Result | Notes |
|-----------|--------|--------|-------|
| 1 | Initial config with TDM slots | ❌ FAIL | I/O error |
| 2 | Fixed clock master (CPU not codec) | ❌ FAIL | Same error |
| 3 | Fixed pin modes (clocks OUTPUT, data INPUT) | ❌ FAIL | Same error |
| 4 | Set tx-num-evt = 0 (capture only) | ❌ FAIL | Same error |
| 5 | Fixed P9_31 offset (0x190 → 0x19C) | ✅ IMPROVED | Pin now in McASP mode |
| 6 | Removed system-clock-frequency | ❌ FAIL | Still I/O error |
| 7 | Set tx-num-evt = 32 (enable TX) | ❌ FAIL | Same error, but clock enables |

**Current DTS**: `BB-I2S-MIC-00A0.dts` with pins properly configured, tx-num-evt=32

---

## Software Stack Verification

| Component | Status | Evidence |
|-----------|--------|----------|
| Device tree loads | ✅ PASS | No compile errors, overlay loads at boot |
| ALSA detects card | ✅ PASS | `arecord -l` shows card 0: I2S-Microphone |
| Driver binds | ✅ PASS | asoc-simple-card driver attached to "sound" device |
| DMA channels assigned | ✅ PASS | dma0chan8 (TX), dma0chan9 (RX) linked to mcasp |
| Device opens | ✅ PASS | arecord can open hw:0,0 |
| hw_params succeeds | ✅ PASS | ALSA verbose shows successful configuration |
| Clock enables | ✅ PASS | mcasp0_fck enable_count: 0→1 during capture |

---

## Kernel Function Tracing Results

**Method**: ftrace with `davinci_mcasp*` filter

**Functions Called Successfully**:
- ✅ `davinci_mcasp_runtime_resume` - Power management
- ✅ `davinci_mcasp_startup` - Device initialization
- ✅ `davinci_mcasp_hw_rule_*` - Parameter negotiation
- ✅ `davinci_mcasp_hw_params` - Hardware configuration
- ✅ `davinci_mcasp_set_dai_fmt` - I2S format setup
- ✅ `davinci_mcasp_trigger` - START command sent

**Functions NEVER Called**:
- ❌ `davinci_mcasp_rx_irq_handler` - RX interrupt handler
- ❌ `davinci_mcasp_tx_irq_handler` - TX interrupt handler
- ❌ `davinci_mcasp_common_irq_handler` - Common interrupt handler

**Conclusion**: Driver configures hardware successfully but interrupts never fire.

---

## PRU Signal Monitor Results

**Method**: Custom PRU0 firmware monitoring P9_29, P9_30, P9_31

**Build Issues Resolved**:
1. ❌ Missing program headers → ✅ Fixed by two-step compile+link with LFLAGS
2. ❌ Wrong linker script (PRU1) → ✅ Fixed by using PRU0 script

**Monitoring Results**:
- WS (P9_29) transitions: **0**
- SCK (P9_31) transitions: **0**
- SD (P9_30) transitions: **0**

**Conclusion**: McASP is NOT generating I2S clocks on output pins.

**Note**: Can no longer use PRU monitor due to pin conflict with McASP after fixing pin mux.

---

## Clock System Analysis

| Check | Status | Result |
|-------|--------|--------|
| McASP0 clock source | ✅ VERIFIED | 24 MHz from clk_24mhz |
| Clock prepared | ✅ YES | prepare_count = 1 |
| Clock enabled (idle) | ✅ NO | enable_count = 0 (normal when idle) |
| Clock enabled (during arecord) | ✅ YES | enable_count = 1 |

---

## ALSA Proc Status (During Capture Attempt)

```
state: RUNNING
hw_ptr: 0        ← Never advances
appl_ptr: 0      ← No data transferred
avail: 0         ← No samples available
```

**Conclusion**: Device state is RUNNING but frozen - no data flow.

---

## Kernel Messages

**Boot Time**:
```
[   68.520717] davinci-mcasp 48038000.mcasp: IRQ common not found
```

**During Capture**: No additional error messages.

**Interpretation**: "IRQ common not found" warning suggests interrupt routing issue.

---

## Unchecked Items (From Research)

Based on interrupt handler debugging research, we have NOT checked:

### 1. IRQ Line Registration
- [ ] Verify interrupt line mapping in device tree
- [ ] Check `/proc/interrupts` for McASP IRQ registration
- [ ] Confirm IRQ number assignment

### 2. McASP Internal Interrupt Enable Bits
- [ ] Read RXFTIE (RX FIFO threshold interrupt enable)
- [ ] Read TXFTIE (TX FIFO threshold interrupt enable)
- [ ] Check other interrupt enable bits in McASP control registers

### 3. Interrupt Status Flags
- [ ] Read McASP status register (SR) during capture
- [ ] Check if RXIF, TXIF flags are being set
- [ ] Determine if hardware events occur but interrupts don't propagate

### 4. Device Tree Interrupt Configuration
- [ ] Verify interrupt properties in mcasp0 node
- [ ] Check interrupt-parent mapping
- [ ] Confirm interrupt numbers match hardware

### 5. Clock Generator Registers
- [ ] Read ACLKXCTL (TX clock control)
- [ ] Read AFSXCTL (TX frame sync control)
- [ ] Read PFUNC (pin function register)
- [ ] Read PDIR (pin direction register)
- [ ] Read GBLCTL (global control register)

---

## Option A Results (Device Tree Master Mode Configuration)

### Interrupt Registration Check
✅ Interrupts ARE registered in `/proc/interrupts`:
```
 31:          0      INTC  80 Level     48038000.mcasp_tx
 32:          0      INTC  81 Level     48038000.mcasp_rx
```
- IRQ numbers: 31 (TX), 32 (RX)
- Counter stays at 0 → handlers never fire

### McASP Hardware Register Analysis

**Tool Created**: `mcasp_simple.c` - reads registers via /dev/mem

**Register Values During Capture Attempt**:
```
PFUNC    (0x010) = 0x00000000  ← All pins in McASP mode (correct)
PDIR     (0x014) = 0xB4000000  ← Pin directions set
ACLKXCTL (0x020) = 0x00000000  ← ❌ TX CLOCK CONTROL IS ZERO!
AHCLKXCTL(0x024) = 0x00000000  ← TX high-freq clock disabled
AFSXCTL  (0x028) = 0x00000000  ← ❌ TX FRAME SYNC CONTROL IS ZERO!
RXSTAT   (0x040) = 0x00000000  ← No RX status
GBLCTL   (0x044) = 0x00000202  ← Clocks in reset
EVTCTLR  (0x054) = 0x00000000  ← No events configured
RXINTCTL (0x058) = 0x00000000  ← ❌ RX INTERRUPTS DISABLED!
SRCTL0   (0x180) = 0x00000002  ← Serializer 0 = RX mode (correct)
```

**Critical Finding**:
- `ACLKXCTL` bit 5 (CLKXM) = 0 → Clock is EXTERNAL (should be 1 for internal generation)
- `AFSXCTL` bit 0 (FSXM) = 0 → Frame sync is EXTERNAL (should be 1 for internal generation)
- Driver is NOT configuring hardware for clock master mode!

### Device Tree Changes Attempted

| Attempt | Change | Register Result | Outcome |
|---------|--------|-----------------|---------|
| 1 | Added `ti,mcasp-auxclk-mode` property | No change | ❌ FAILED |
| 2 | Enabled TX serializer (AXR1) alongside RX (AXR0) | No change | ❌ FAILED |
| 3 | Set `serial-dir = <2 1 0 0...>` for TX+RX | ACLKXCTL still 0x00 | ❌ FAILED |

### Root Cause

**The davinci-mcasp driver does NOT write master mode clock configuration to hardware registers**, despite:
- simple-audio-card specifying `bitclock-master` and `frame-master`
- Driver's `davinci_mcasp_set_dai_fmt` function being called (verified via ftrace)
- TX and RX serializers both being configured

**Conclusion**: This is a driver bug or missing configuration that cannot be fixed via device tree alone.

---

## Next Steps

**Option A**: ❌ EXHAUSTED - Cannot fix via device tree
- Driver does not honor master mode configuration
- Hardware registers remain in slave mode

**Option B**: Proceed with direct register control (4-6 hours)
- Manually write ACLKXCTL, AFSXCTL, GBLCTL registers
- Bypass driver's broken clock configuration
- This is the only remaining path to verify if INMP441 hardware works

---

## Failed Approaches (Do Not Retry)

- ❌ Using PRU to monitor signals while McASP is configured (pin conflict)
- ❌ Codec as clock master (INMP441 requires CPU as master)
- ❌ TX disabled (may need TX for clock generation)
- ❌ Using 24.576 MHz system clock (BeagleBone only has 24 MHz)
- ❌ Device tree `ti,mcasp-auxclk-mode` property (no effect)
- ❌ Enabling TX serializer alongside RX (ACLKXCTL still 0x00)
- ❌ simple-audio-card master mode specification (driver ignores it)

---

**Last Updated**: 2025-12-17 23:00 UTC
