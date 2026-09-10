# I2S MEMS Microphone Audio Implementation Plan

## Overview

This document outlines the plan to transition PRU1 audio sampling from the analog ADC (AIN1 at 40 kHz) to an I2S MEMS microphone (INMP441/SPH0645 at 48 kHz) with direct PRU control of McASP0.

## Goals

1. Replace ADC sampling with I2S/McASP sampling on PRU1
2. Maintain FFT processing on PRU for CPU offloading
3. Support 24-bit audio resolution (stored in 32-bit containers)
4. Achieve 48 kHz sample rate with 1024-sample buffers
5. Keep output format compatible with existing API (frequency bands)

## Current Architecture

```
ADC (AIN1) → PRU1 IEP Timer (40 kHz) → Sample Buffer → FFT → Frequency Bands
```

**Current Memory Layout (Shared Memory 12KB):**
```
0x00010000-0x00010FFF: PRU0 IR ring buffer (4KB reserved, 2KB used)
0x00011000-0x0001101F: PRU0 IR control block (32 bytes)
0x00012000-0x00012057: PRU1 audio control block (88 bytes)
```

**Current PRU1 DRAM Layout (8KB):**
```
0x0000-0x07FF: Sample Buffer A (2KB = 1024 × 16-bit)
0x0800-0x0FFF: Sample Buffer B (2KB = 1024 × 16-bit)
0x1000-0x1FFF: FFT working buffer (4KB = 1024 × complex Q15)
```

## Target Architecture

```
I2S Mic → McASP0 → PRU1 (48 kHz) → Sample Buffer → FFT → Frequency Bands
```

**New Shared Memory Layout (12KB):**
```
0x0000-0x07FF: PRU0 IR ring buffer (2KB)
0x0800-0x08FF: PRU0 IR control block (256B reserved)
0x0900-0x09FF: PRU1 audio control block (256B reserved)
0x0A00-0x19FF: PRU1 Sample Buffer A (4KB = 1024 × 32-bit)
0x1A00-0x29FF: PRU1 Sample Buffer B (4KB = 1024 × 32-bit)
0x2A00-0x2FFF: Reserved (1.5KB)
```

**New PRU1 DRAM Layout (8KB):**
```
0x0000-0x0FFF: FFT working buffer (4KB = 1024 × complex Q15)
0x1000-0x1FFF: Preprocessing scratch buffer (4KB)
```

## Hardware Configuration

### I2S MEMS Microphone Pins
- P9_25: CLKOUT2 (24.576 MHz clock output from onboard oscillator)
- P9_28: McASP0_AHCLKR (24.576 MHz clock input - jumper from P9_25)
- P9_29: McASP0_FSX (I2S Frame Sync / Word Select output)
- P9_30: McASP0_AXR0 (I2S Data input from mic)
- P9_31: McASP0_ACLKX (I2S Bit Clock output - 3.072 MHz)

### Clock Configuration
- Master clock: 24.576 MHz (onboard oscillator via CLKOUT2)
- Bit clock: 3.072 MHz (24.576 MHz / 8)
- Sample rate: 48 kHz (3.072 MHz / 64 bits per frame)
- Frame: 64 bits (2 × 32-bit slots for stereo, we use left channel only)

### McASP0 Register Base
- Configuration registers: 0x48038000
- Data/FIFO registers: 0x46000000

## Implementation Phases

### Phase 1: Shared Memory Reorganization

**Files to modify:**

1. Create shared header:
   - `pru/pru_shared_memory.h` - Central memory layout definitions

2. PRU0 IR Remote updates:
   - `pru/remote/pru0_ir_detector.c` - Update CONTROL_BLOCK offset (0x1000 → 0x0800)
   - `pru/remote/test_r31.c` - Update CONTROL_BLOCK offset

3. PRU1 Audio updates:
   - `pru/audio/pru1_audio.c` - Update AUDIO_CONTROL_BLOCK offset (0x2000 → 0x0900)
   - `pru/audio/pru1_audio.c` - Add shared memory buffer offsets

4. Go API updates:
   - `api/remote/pru_detector.go` - Update controlBlockOffset (0x1000 → 0x0800)
   - `api/audio/pru_sampler.go` - Update audioControlBlockOffset (0x2000 → 0x0900)
   - `api/audio/pru_raw_sampler.go` - Update rawControlBlockOffset
   - `pru/audio/scripts/cmd/pru_memory.go` - Update audioControlBlockOffset
   - `api/cmd/pru-debug.go` - Update controlBlockOffset

5. Script updates:
   - `scripts/debug_pru.sh` - Update memory read address

6. Documentation updates:
   - `pru/remote/README.md`
   - `pru/audio/README.md`
   - `pru/audio/PROJECT_PLAN.md`

### Phase 2: Device Tree for PRU-Controlled McASP

Create new device tree overlay that:
- Configures McASP0 pins for I2S
- Enables CLKOUT2 (24.576 MHz)
- Does NOT create ALSA audio card (PRU controls McASP directly)

**Files to create:**
- `hardware/device-tree/BB-I2S-PRU-00A0.dts` - PRU-controlled I2S overlay

**Key differences from existing BB-I2S-MIC-00A0.dts:**
- Remove `simple-audio-card` fragment
- Remove `dmic_codec` fragment
- Keep pin mux configuration
- Keep McASP clock configuration

### Phase 3: McASP Register Definitions

Create McASP register header for PRU access.

**Files to create:**
- `pru/audio/am335x_mcasp.h` - McASP register definitions

**Key registers needed:**
- GBLCTL - Global control
- RGBLCTL - Receive global control
- RMASK - Receive format mask
- RFMT - Receive format
- AFSRCTL - Receive frame sync control
- ACLKRCTL - Receive bit clock control
- AHCLKRCTL - Receive high-frequency clock control
- RSRCTL0 - Receive serializer 0 control
- RSTAT - Receive status
- RBUF0 - Receive buffer 0

### Phase 4: PRU1 Firmware Update

Modify PRU1 audio firmware for I2S input.

**Changes to `pru/audio/pru1_audio.c`:**

1. Replace `init_adc()` with `init_mcasp_i2s()`:
   - Configure McASP0 for I2S receive mode
   - Set up clock dividers for 48 kHz
   - Configure serializer 0 for receive
   - Enable receive section

2. Replace ADC read with McASP read:
   - Poll RSTAT for RDATA bit (receive data ready)
   - Read from RBUF0 register
   - Extract 24-bit sample from 32-bit frame

3. Update sample format handling:
   - Current: 12-bit unsigned (0-4095)
   - New: 24-bit signed (stored in 32-bit)
   - Adjust DC offset removal for signed data

4. Update timing:
   - Current: IEP timer at 40 kHz triggers ADC
   - New: McASP provides 48 kHz clock, PRU syncs to RDATA

5. Update buffer locations:
   - Move sample buffers from PRU DRAM to shared memory

### Phase 5: FFT Parameter Updates

Update FFT for 48 kHz sample rate.

**Changes:**
- Frequency resolution: 48000 / 1024 = 46.875 Hz/bin (was 39.0625 Hz/bin)
- Nyquist frequency: 24 kHz (was 20 kHz)
- Bin calculations: `bin = (freq_hz * 1024) / 48000`

**Files to modify:**
- `pru/audio/pru1_audio.c` - Update SAMPLE_RATE_HZ constant
- `pru/audio/fft.h` - No changes needed (FFT size unchanged)

### Phase 6: Debugging Utilities

Create utilities to verify McASP operation.

**Files to create:**
- `pru/audio/scripts/cmd/mcasp_status.go` - Read McASP register state
- Extend `pru/audio/scripts/cmd/sample.go` - Add I2S-specific diagnostics

**Debug approach:**
1. Verify device tree loaded (check pin mux)
2. Verify 24.576 MHz clock on P9_25
3. Verify McASP register configuration
4. Monitor RSTAT for data flow
5. Capture raw samples before FFT
6. Compare frequency response to known signals

### Phase 7: API Layer Updates

Update Go API for 48 kHz and 24-bit samples.

**Files to modify:**
- `api/audio/pru_sampler.go` - Update SampleRateHz constant
- `api/audio/pru_models.go` - Add SampleBitDepth field
- Documentation updates

## Risk Mitigation

### Risk 1: McASP configuration complexity
- **Mitigation**: Start with minimal config, add features incrementally
- **Fallback**: Can revert to IIO/ALSA approach if PRU McASP fails

### Risk 2: Clock synchronization issues
- **Mitigation**: Use McASP's internal clock management
- **Debug**: Add clock error detection in control block

### Risk 3: Memory layout changes break existing functionality
- **Mitigation**: Update and test PRU0 IR remote first (simpler)
- **Fallback**: Git commits allow easy rollback

### Risk 4: 24-bit samples cause FFT overflow
- **Mitigation**: Scale samples during preprocessing
- **Note**: Current FFT uses Q15 (16-bit), so we truncate to 16-bit for FFT

## Testing Strategy

### Phase 1 Testing (Memory Reorganization)
1. Rebuild and deploy PRU0 IR firmware
2. Verify IR remote still works
3. Rebuild and deploy PRU1 audio firmware (still using ADC)
4. Verify audio sampling still works
5. Run `audio-util sample` to confirm control block access

### Phase 2-3 Testing (Device Tree + McASP Header)
1. Load new device tree overlay
2. Verify pin mux with `cat /sys/kernel/debug/pinctrl/*/pins`
3. Measure 24.576 MHz on P9_25 with oscilloscope/logic analyzer

### Phase 4-5 Testing (PRU1 Firmware)
1. Connect I2S microphone
2. Monitor RSTAT for data ready indication
3. Capture raw samples to verify data flow
4. Generate known frequency tone, verify FFT output
5. Compare to previous ADC-based results

### Phase 6-7 Testing (Utilities + API)
1. Run new McASP status utility
2. Test API endpoints with audio streaming
3. Verify LED audio modulation works

## Commit Strategy

Make incremental commits after each logical change:

1. Add implementation plan (this document)
2. Add shared memory layout header
3. Update PRU0 memory offsets + test
4. Update PRU1 memory offsets + test
5. Update Go API memory offsets + test
6. Add PRU-controlled McASP device tree overlay
7. Add McASP register header
8. Update PRU1 firmware for I2S (incremental sub-commits)
9. Update FFT parameters
10. Add debugging utilities
11. Update API layer
12. Final documentation updates

## Dependencies

- TI PRU C Compiler (clpru) - already installed
- Device tree compiler (dtc) - already available
- BeagleBone Black with 24.576 MHz oscillator - standard on BBB
- INMP441 or SPH0645 I2S MEMS microphone - user provided
- Jumper wire P9_25 → P9_28 - user provided

## References

- [Bela PRU Audio Implementation](https://github.com/BelaPlatform/Bela)
- [TI AM335x Technical Reference Manual](https://www.ti.com/lit/ug/spruh73p/spruh73p.pdf) - Chapter 22: McASP
- [TI StarterWare McASP Examples](https://github.com/embest-tech/AM335X_StarterWare_02_00_01_01)
- [BeagleBoard I2S Discussions](https://forum.beagleboard.org/t/mcasp-on-the-pru/37336)
