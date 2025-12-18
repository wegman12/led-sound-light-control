# I2S Microphone Debugging Findings

## Summary

After extensive kernel-level debugging, we discovered that **writing to the PDIR (pin direction) register clears the ACLKXCTL and related clock control registers** on the TI AM335x McASP hardware. This appears to be undocumented hardware behavior that prevents proper clock generation in master mode.

## Current Status

- ✅ Device tree loads successfully
- ✅ ALSA detects audio card
- ✅ Pin muxing is correct (P9_29, P9_30, P9_31 in McASP mode)
- ✅ DMA transfers data (hw_ptr advances)
- ✅ Driver executes master mode configuration code path
- ❌ McASP hardware does not generate I2S clocks
- ❌ All captured audio data is zeros (silence)

## Root Cause

### Hardware Behavior Discovered

When the driver writes to the PDIR register to configure pins as outputs during `mcasp_start_rx()`:

1. ACLKXCTL register value **before** PDIR write: `0x001800a0` (clock enabled, divider configured)
2. Driver writes to PDIR: `0xbc000000` (pins 26,27,28,29,31 as outputs)
3. ACLKXCTL register value **after** PDIR write: `0x000000a3` (upper bits cleared!)

The cleared bits in ACLKXCTL include:
- Bits 16-20: Clock divider configuration
- These bits are set earlier by `__davinci_mcasp_set_clkdiv()`

### Register Read Results (During Active Capture)

Reading McASP registers at 0x48038000 via `/dev/mem` during capture:
```
PDIR     = 0x00000000 (expect 0xbc000000)
ACLKXCTL = 0x00000000 (expect bit 5 set)
TXFMCTL  = 0x00000000 (expect bit 0 set)
GBLCTL   = 0x0000131F (state machines running)
```

All critical registers read as zero except GBLCTL, indicating the hardware is active but not properly configured.

## Debug Module Testing

Built a custom kernel module from BeagleBoard kernel sources with extensive debug logging:

```c
// In mcasp_set_clk_pdir(), we tried to restore clock config after PDIR write:
mcasp_set_bits(mcasp, DAVINCI_MCASP_ACLKXCTL_REG, ACLKXE);
mcasp_set_bits(mcasp, DAVINCI_MCASP_ACLKRCTL_REG, ACLKRE);
mcasp_set_bits(mcasp, DAVINCI_MCASP_TXFMCTL_REG, AFSXE);
mcasp_set_bits(mcasp, DAVINCI_MCASP_RXFMCTL_REG, AFSRE);
```

**Result:** Bits are set temporarily but get cleared again before the function returns, suggesting continuous hardware-level clearing or a required initialization sequence we're missing.

## Device Tree Configuration Tested

### Initial Configuration
```dts
serial-dir = < 2 1 0 0... >;  // AXR0=RX, AXR1=TX
tx-num-evt = <32>;
rx-num-evt = <32>;
```

### Simplified Configuration (Current)
```dts
serial-dir = < 2 0 0 0... >;  // AXR0=RX only
rx-num-evt = <32>;              // No TX serializer
```

Both configurations show identical symptoms.

## Code Paths Verified

1. **set_dai_fmt()**: Correctly identifies CBS_CFS (CPU as clock master) mode
2. **mcasp_start_rx()**: Executes synchronous mode branch, calls mcasp_set_clk_pdir()
3. **mcasp_set_clk_pdir()**: Sets PDIR register, attempts to restore clock bits
4. **GBLCTL operations**: State machines are released properly (RXHCLKRST, RXCLKRST, etc.)

## Attempted Solutions

### 1. Clock Property Configuration
- Added `clocks = <&mcasp0_fck>` to CPU node
- Added `system-clock-direction-out` flag
- **Result:** No change in behavior

### 2. Codec Change
- Changed from `linux,spdif-dir` to `dmic-codec`
- **Result:** No change in behavior

### 3. Kernel Module Modifications
- Save and restore clock control registers around PDIR writes
- Re-enable clock bits after PDIR write
- Move clock enable to end of mcasp_start_rx() after all state machines released
- **Result:** Registers don't retain values even when explicitly set

### 4. Device Tree Simplification
- Removed TX serializer (AXR1) for RX-only operation
- Removed tx-num-evt parameter
- **Result:** No change in behavior

## Hardware vs Software Issue

Evidence this may NOT be purely a software issue:

1. **Other users would have found this**: If the vanilla McASP driver had this bug for I2S master mode, it would have been discovered and fixed long ago.

2. **Working configurations exist**: BeagleBone Black HDMI audio works in master mode, suggesting the hardware CAN work.

3. **Register writes don't stick**: Even direct writes to ACLKXCTL from kernel context don't retain values, suggesting a hardware constraint or missing prerequisite.

4. **Power/Clock domain issue**: Registers may require specific power domain or clock states that our configuration doesn't enable.

## Key Differences from Working HDMI Audio

The BeagleBone Black HDMI audio configuration (which works) has:

```dts
clk_mcasp0_fixed: clk_mcasp0_fixed {
    compatible = "fixed-clock";
    clock-frequency = <24576000>;
};

clk_mcasp0: clk_mcasp0 {
    compatible = "gpio-gate-clock";
    clocks = <&clk_mcasp0_fixed>;
    enable-gpios = <&gpio1 27 0>;  // External oscillator enable
};
```

This uses an **external 24.576 MHz audio oscillator** on the HDMI cape, enabled via GPIO. Our configuration relies on internal clock generation, which may have different requirements.

## Next Steps / Recommendations

### Option 1: External Audio Oscillator (Recommended)
Add an external audio clock oscillator to the hardware:
- 24.576 MHz crystal oscillator
- Connect to McASP0 AHCLKX input
- Configure as clock source in device tree
- This matches proven working configurations

### Option 2: Find Required Device Tree Properties
Search TI documentation/forums for:
- Required power domain properties
- Clock tree configuration for internal clock generation
- AHCLK (auxiliary high-speed clock) configuration
- Any AM335x-specific errata related to McASP master mode

### Option 3: Alternative I2S Interface
Consider using:
- PRU (Programmable Real-Time Unit) to bit-bang I2S
- External I2S codec chip with built-in PLL
- USB audio interface

### Option 4: Contact TI Support
This appears to be undocumented hardware behavior. TI technical support may have insights into:
- Required initialization sequence for McASP master mode without external clock
- Known errata or limitations
- Reference configurations for internal clock generation

## Files Modified

- `hardware/device-tree/BB-I2S-MIC-00A0.dts` - Simplified device tree overlay
- Built custom kernel module: `~/mcasp-debug-build/snd-soc-davinci-mcasp.ko` (on BeagleBone)
- Debug patches applied to: `~/repositories/external/linux_kernel/beagleboard/sound/soc/ti/davinci-mcasp.c`

## Relevant Hardware

- BeagleBone Black Rev C
- INMP441 I2S MEMS Microphone
- Kernel: 5.10.168-ti-r83
- Pins: P9_29 (FSX), P9_30 (AXR0), P9_31 (ACLKX)

## Conclusion

The McASP hardware on AM335x appears to require specific conditions (likely an external clock source or specific power/clock domain configuration) for the clock control registers to function properly in master mode. The device tree configuration alone is insufficient - either additional hardware (external oscillator) or undocumented device tree properties are required.
