# PRU1 Audio Sampling - Development Session Log

## Session 1: Initial Setup and Basic Testing (Stages 0-3)

**Date**: 2025-12-13
**Status**: ⚠️ Partial completion - firmware hangs at 22 seconds

### ✅ Completed Work

#### Stage 0: Project Planning ✓
- **Created PROJECT_PLAN.md**: Comprehensive 11-stage implementation plan
  - Memory layout design (PRU1 local DRAM at 0x00002000 + shared memory at 0x00012000)
  - Real-time sampling strategy with double buffering
  - FFT implementation approach (fixed-point Cooley-Tukey)
  - 40 kHz sampling target with <10ms FFT processing
  - 4 configurable frequency bins: Bass (0-150Hz), Mid-Low (150-1000Hz), Mid-High (1000-2000Hz), Treble (2000+Hz)
  - Includes BeagleBone kernel version info (5.10.168-ti-r83, remoteproc driver)

#### Stage 1: PRU1 Basic Infrastructure ✓
- **pru1_audio.c**: Simple test firmware
  - Toggles status bit in shared memory every second
  - Counter increments each iteration
  - Uses shared memory offset 0x2000 (8KB, no conflict with PRU0 at 0x0000-0x1000)
  - Status indicator: 0x41554431 ("AUD1" in ASCII)

- **am335x_pru.cmd**: PRU1-specific linker script
  - Configured for PRU1 local DRAM (PRU_DMEM_1_0 at 0x00002000)
  - Proper memory layout for PRU1

- **resource_table_empty.h**: RemoteProc resource table (required by kernel)

- **Makefile**: Complete build and deployment system
  - Firmware compilation with TI clpru compiler
  - Utility build for local (amd64) and BeagleBone (ARMv7)
  - `deploy-all` target for complete deployment
  - Builds firmware on BeagleBone (has clpru installed)

#### Stage 2: Utility Application ✓
- **Cobra CLI application** in `pru/audio/scripts/`
  - **main.go**: Entry point
  - **cmd/root.go**: Root command definition
  - **cmd/test.go**: Test command implementation
  - **go.mod + go.sum**: Go module with Cobra v1.8.1 dependency

- **test command features**:
  - Memory-maps PRU shared memory via /dev/mem
  - Reads audio control block at offset 0x2000
  - Monitors PRU1 status, counter, and toggle bit
  - Real-time display with 1-second updates
  - Clear error messages for common issues

#### Stage 3: Deployment ⚠️ Partial
- ✅ Successfully deployed PRU1 firmware to BeagleBone
- ✅ PRU1 loads and starts (verified: remoteproc2 state = "running")
- ✅ Utility application deployed and functional
- ✅ Shared memory communication working (status reads as "1DUA" = "AUD1" reversed due to endianness)
- ❌ **ISSUE**: Firmware hangs at 22 seconds - see below

#### Git Commits (Traceability)
```
b536180 - Add PRU1 audio sampling project plan
7242bcb - Add PRU1 basic infrastructure and test firmware
f4038dd - Add audio utility CLI with test command
3246849 - Add PRU1 audio system README
801d33f - Fix Makefile clean target and add go.sum
```

### ⚠️ BLOCKING ISSUE: PRU1 Firmware Hangs at 22 Seconds

#### Problem Description
The firmware counter increments properly from 0-22, then stops. Toggle bit also stops updating. The PRU remains in "running" state (not crashed), indicating it's stuck in an infinite loop.

#### Timeline
- **00:00-21s**: Counter increments 0→21, toggle bit flips every second ✓
- **22s**: Counter reaches 22, toggle bit makes final flip
- **23s+**: Counter stuck at 22, toggle bit stuck at 0, no further updates

#### Significance of 22 Seconds
- PRU cycle counter is 32-bit at 200 MHz
- Overflow occurs at: 2^32 / 200,000,000 = 21.47 seconds
- This strongly suggests cycle counter overflow is involved

#### What We Tried (All Failed)

1. **Chunked delays** ❌
   - Broke `__delay_cycles(200000000)` into 10× 100ms chunks
   - Broke into 1000× 1ms chunks (200,000 cycles each)
   - Result: Still hangs at 22 seconds

2. **Counter reset (PRU0 pattern)** ❌
   - Added `reset_counter()` at beginning of loop
   - Moved `reset_counter()` to after delay
   - Removed counter reset entirely
   - Result: No change

3. **PRU0 cooldown pattern** ❌
   - Used 5× `__delay_cycles(40000000)` (5× 200ms, same as PRU0)
   - PRU0 uses this successfully for indefinite operation
   - Result: Still hangs at 22 seconds

4. **Volatile counter loop** ❌
   - Replaced `__delay_cycles()` with volatile counter loop
   - `for (volatile uint32_t i = 0; i < 50000000; i++)`
   - Result: Still hangs at 22 seconds

#### Key Observations

1. **Consistent failure point**: Always exactly 22 seconds, never varies
2. **PRU doesn't crash**: `remoteproc2/state` remains "running"
3. **Shared memory writes work initially**: Counter 0-22 and toggle updates prove writes function
4. **PRU0 works fine**: Similar code patterns in PRU0 remote detector work indefinitely
5. **Compiler warnings**: `>> WARNING: object file specified, but linking not enabled` during compilation

#### Current Firmware State
```c
static void delay_one_second(void) {
    /* 40M cycles = 200ms @ 200MHz (same as PRU0's COOLDOWN_AFTER_SUCCESS) */
    /* Do this 5 times for 1 second total */
    __delay_cycles(40000000);
    __delay_cycles(40000000);
    __delay_cycles(40000000);
    __delay_cycles(40000000);
    __delay_cycles(40000000);
}

void main(void) {
    // ... initialization ...

    while (1) {
        ctrl->counter++;
        ctrl->toggle_bit = (ctrl->toggle_bit == 0) ? 1 : 0;
        delay_one_second();
        reset_counter();  // Reset after delay
    }
}
```

#### Theories Not Yet Tested

1. **Compiler-generated code issue**
   - `__delay_cycles()` may compile to code that uses cycle counter incorrectly
   - Need to examine generated assembly to see actual implementation
   - May need to use IEP timer instead of cycle counter for delays

2. **PRU0 vs PRU1 difference**
   - Despite being "identical", there may be subtle hardware differences
   - PRU1 cycle counter may behave differently than PRU0's
   - May need PRU1-specific workarounds

3. **Optimization issue**
   - Compiler with `-Ooff` (optimization off) may still optimize something incorrectly
   - Shared memory writes may be getting optimized out after certain point
   - Try different optimization flags

4. **Linker script issue**
   - PRU1 linker script may have subtle issues
   - Memory sections may not be configured correctly
   - Compare more carefully with PRU0's working linker script

5. **Shared memory contention**
   - Both PRU0 and PRU1 accessing shared memory simultaneously
   - May need memory barriers or synchronization
   - Though unlikely given clean initial operation

#### Files Modified (Uncommitted)
```
pru/audio/pru1_audio.c  - Multiple delay implementations tried
```

### 🔄 Next Session Action Items

#### Immediate Actions (Resume Point)
1. **Examine generated assembly**
   ```bash
   ssh bbb1.wegman "cd ~/led-sound-light-control/pru-audio/ && cat gen/pru1_audio.asm | grep -A 30 'delay_one_second'"
   ```
   - See what `__delay_cycles()` actually compiles to
   - Look for cycle counter usage patterns

2. **Try IEP timer instead of cycle counter**
   - IEP (Industrial Ethernet Peripheral) has dedicated timer
   - May be more reliable than cycle counter for long delays
   - Reference: pru_iep.h in PRU software support package

3. **Simplest possible test**
   - Remove ALL delays, just increment counter rapidly
   - See if counter wraps past 22 or if 22 has other significance
   - Proves/disproves delay-specific issue

4. **Compare with PRU0 assembly**
   - Build PRU0 remote code
   - Compare generated assembly for delay functions
   - Look for PRU0-specific patterns we're missing

#### Alternative Approaches
- **Option A**: Accept rapid updates (no 1-second delay) for test firmware
  - Fast counter increment proves everything works
  - Real audio sampling won't need long delays anyway
  - Move forward to Stage 4 (ADC configuration)

- **Option B**: Deep dive on assembly/compiler
  - Generate assembly with `-as` flag
  - Hand-optimize delay function if needed
  - May take significant time

- **Option C**: Use simpler test pattern
  - Have PRU write incrementing values without delays
  - Go utility sleeps between reads (delay is in Go, not PRU)
  - Proves shared memory communication works

### 📝 Current System State

#### On BeagleBone (bbb1.wegman)
- **PRU1 Firmware**: `/lib/firmware/am335x-pru1-fw` (hangs at 22s)
- **Source Files**: `~/led-sound-light-control/pru-audio/`
- **Utility**: `~/led-sound-light-control/pru-audio/audio-util`
- **PRU1 State**: running (remoteproc2)
- **PRU0 State**: running (remoteproc1, IR detector functional)

#### On Development Machine
- **Source**: `/home/kevin/repositories/personal/led-sound-light-control/pru/audio/`
- **Branch**: project-setup
- **Uncommitted changes**: pru1_audio.c (various delay attempts)

#### Test Commands
```bash
# Check PRU1 status
ssh bbb1.wegman "cat /sys/class/remoteproc/remoteproc2/state"

# Run test (will hang at 22s)
ssh bbb1.wegman "cd ~/led-sound-light-control/pru-audio/ && sudo ./audio-util test"

# Reload PRU1 firmware
ssh bbb1.wegman "echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc2/state && \
                 sleep 1 && \
                 echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state"

# Check kernel logs
ssh bbb1.wegman "dmesg | tail -20"
```

### 📚 References

- **PROJECT_PLAN.md**: Full implementation plan (11 stages)
- **README.md**: Quick start guide and troubleshooting
- **pru/remote/pru0_ir_detector.c**: Working PRU0 reference implementation
- **TI PRU-ICSS Reference**: spruh73 (AM335x TRM)

### 💡 Recommendations for Next Session

1. **Start with simplest test** (Option A above) - don't delay, just prove continuous operation
2. **If that works**, move forward to Stage 4 (ADC configuration)
3. **If simplest test also hangs at 22s**, then deep dive on assembly/hardware issue
4. **Keep PRU0 remote detector running** during tests to prove PRU subsystem is healthy

The 22-second barrier is frustrating but not blocking - the real audio sampling application won't need 1-second delays, it needs microsecond-precision timing for 40 kHz sampling. We can revisit delay implementation when we add IEP timer support for real-time sampling.

---

**Session End**: Stages 0-2 complete ✓, Stage 3 partially complete ⚠️
**Blocker**: Firmware hang at 22 seconds
**Next**: Debug delay issue or move forward with no-delay test
