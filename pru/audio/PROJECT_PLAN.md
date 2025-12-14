# PRU1 Audio Sampling and FFT Analysis Project Plan

## Project Overview

Migrate audio sampling and FFT analysis from Go API to PRU1 for high-precision 40 kHz sampling and real-time frequency analysis. This will enable tight-loop LED color modulation based on music frequency profiles.

## System Information

### BeagleBone Black Environment
- **Kernel Version**: 5.10.168-ti-r83 #1bookworm SMP PREEMPT
- **Architecture**: armv7l
- **Distribution**: Debian Bookworm
- **PRU Subsystem**: AM335x PRU-ICSS (remoteproc driver)

## System Architecture

### Hardware Configuration
- **PRU**: PRU1 (PRU0 is used for IR remote detection)
- **ADC Channel**: AIN1 (fixed by hardware cape)
- **Sampling Rate**: 40 kHz (25 μs period) - CRITICAL requirement
- **Sample Window**: 1024 samples (default, configurable: 512-2048)
  - 1024 samples @ 40 kHz = 25.6 ms latency
  - Frequency resolution: 39.06 Hz per bin
  - Good balance between LED responsiveness and bass detection accuracy

### Memory Layout

#### PRU1 Local DRAM (8KB at 0x00002000)
Used for internal/working memory:
- **Sample Buffer 0**: Offset 0x0000, 2KB (1024 samples × 2 bytes max)
- **Sample Buffer 1**: Offset 0x0800, 2KB (1024 samples × 2 bytes max)
- **FFT Working Memory**: Offset 0x1000, 4KB
  - Complex FFT results: 1024 samples × 4 bytes (real+imag as int16)
  - Magnitude calculations
  - Temporary computation space

#### Shared Memory (at 0x00010000)
Used for API-accessible results:
- **PRU0 Usage** (existing IR detector):
  - Offset 0x0000: Ring buffer (2KB)
  - Offset 0x1000: Control block (28 bytes)

- **PRU1 Usage** (audio sampling - NEW):
  - **Offset 0x2000**: Audio Control Block (64 bytes)
    - Enable/disable flag
    - Sample size configuration
    - Frequency bin boundaries (4 × uint16)
    - Status flags
    - Statistics counters
  - **Offset 0x2100**: Sound Profile Ring Buffer (configurable, ~1KB)
    - Ring buffer of sound profiles
    - Each profile: 16 bytes (4 × float32 for bins + timestamp)
    - Buffer size: 64 profiles = 1KB

### Sound Profile Output

#### Frequency Bins (Configurable)
Default configuration:
- **Bass**: 0-150 Hz (bins 0-3 @ 39 Hz resolution)
- **Mid-Low**: 150-1000 Hz (bins 4-25)
- **Mid-High**: 1000-2000 Hz (bins 26-51)
- **Treble**: 2000-20000 Hz (bins 52-511, Nyquist @ 20 kHz)

All bin boundaries stored in shared memory control block for runtime tuning.

#### Output Structure
```c
struct sound_profile {
    uint32_t bass;        // Magnitude sum for bass frequencies
    uint32_t mid_low;     // Magnitude sum for mid-low frequencies
    uint32_t mid_high;    // Magnitude sum for mid-high frequencies
    uint32_t treble;      // Magnitude sum for treble frequencies
    uint32_t timestamp;   // PRU cycle counter timestamp
    uint32_t reserved[3]; // Future expansion (total 32 bytes)
};
```

### Real-Time Sampling Strategy

To guarantee 40 kHz sampling without disruption from FFT processing:

1. **Double Buffering**: Two sample buffers in PRU1 DRAM
   - Buffer A: Collecting samples
   - Buffer B: Being processed (FFT)
   - Swap when Buffer A is full

2. **Timer-Based Sampling**: Use PRU IEP (Industrial Ethernet Peripheral) timer
   - Configure IEP for 25 μs interrupts (40 kHz)
   - ISR reads ADC via PRU direct access
   - Minimal ISR overhead (<1 μs)

3. **Background FFT Processing**:
   - FFT runs on completed buffer while new buffer fills
   - If FFT takes >25.6 ms, skip that buffer (log warning)
   - FFT should take ~5-10 ms with optimized fixed-point implementation

### FFT Implementation

**Algorithm**: Radix-2 Cooley-Tukey FFT (decimation in time)
- Optimized for power-of-2 sizes (512, 1024, 2048)
- Fixed-point arithmetic (Q15 format: 16-bit signed, 15 fractional bits)
- In-place computation to minimize memory
- Bit-reversal reordering
- Pre-computed twiddle factors in flash

**Performance Target**: <10 ms for 1024-point FFT
- PRU runs at 200 MHz
- ~400k cycles available per FFT
- Simple operations: ~400 cycles per butterfly

## Implementation Stages

### Stage 0: Project Planning ✓
- **Deliverable**: This document (PROJECT_PLAN.md)
- **Purpose**: Reference for session recovery

### Stage 1: Basic PRU1 Infrastructure
**Files to create**:
- `pru/audio/pru1_audio.c` - Simple test firmware
- `pru/audio/Makefile` - Build system
- `pru/audio/resource_table_empty.h` - RemoteProc resource table
- `pru/audio/am335x_pru.cmd` - Linker script for PRU1

**Test**: Toggle a bit in shared memory every second
**Validation**: Go utility reads and prints bit state

**Git commit**: "Add PRU1 basic infrastructure and test firmware"

### Stage 2: Utility Application
**Files to create**:
- `pru/audio/scripts/main.go` - Cobra CLI entry point
- `pru/audio/scripts/cmd/root.go` - Root command
- `pru/audio/scripts/cmd/test.go` - Test command (read bit toggle)
- `pru/audio/scripts/go.mod` - Go module definition

**Commands**:
- `audio-util test` - Monitor PRU1 status bit toggle

**Git commit**: "Add audio utility CLI with test command"

### Stage 3: Deployment and Validation
**Actions**:
- Deploy PRU1 firmware to BeagleBone
- Deploy utility application
- Run test command
- Verify PRU1 operation

**Git commit**: "Verify PRU1 basic operation on BeagleBone"

### Stage 4: ADC Configuration
**Files to modify**:
- `dto/uEnv.txt` - Add ADC overlay and PRU1 configuration

**Configuration needed**:
```bash
# Enable ADC
cape_enable=bone_capemgr.enable_partno=BB-ADC

# Enable PRU (kernel 5.10 uses remoteproc)
uboot_overlay_pru=/lib/firmware/AM335X-PRU-RPROC-4-19-TI-00A0.dtbo
```

**Note**: Kernel 5.10.168-ti-r83 uses remoteproc driver, not UIO driver
- PRU control via `/sys/class/remoteproc/remoteproc1/` and `/remoteproc2/`
- Firmware loaded via `/lib/firmware/`

**Potential custom overlay** (if needed):
- Configure AIN1 for continuous conversion
- Set ADC clock for adequate sampling
- Grant PRU access to ADC registers

**Git commit**: "Configure ADC and PRU1 in device tree"

### Stage 5: ADC Overlay Deployment
**Actions**:
- Copy uEnv.txt to BeagleBone
- Reboot if needed
- Verify ADC accessible at `/sys/bus/iio/devices/iio:device0`
- Test ADC reading from command line
- Verify PRU remoteproc devices are available

**Git commit**: "Deploy and verify ADC configuration"

### Stage 6: Simple Audio Detection
**Modifications**:
- Update `pru1_audio.c`:
  - Read from ADC at 10 Hz (slow test rate)
  - Write raw samples to shared memory buffer
  - Simple polling loop (no timer yet)

**Utility command**:
- `audio-util sample` - Read and display raw ADC samples

**Validation**:
- Connect audio source to AIN1
- Run sample command
- Verify samples change with audio

**Git commit**: "Implement basic ADC sampling and readback"

### Stage 7: Audio Detection Deployment
**Actions**:
- Deploy updated PRU1 firmware
- Deploy utility with sample command
- Test with various audio sources
- Troubleshoot any ADC access issues

**Git commit**: "Verify ADC sampling on BeagleBone"

### Stage 8: Full FFT Implementation
**Major implementation** - Split into sub-tasks:

#### 8a. Timer-Based Sampling
- Configure IEP timer for 40 kHz interrupts
- Implement ISR for ADC reads
- Double-buffer management
- Buffer full flag for FFT processing

#### 8b. FFT Implementation
- Fixed-point Cooley-Tukey FFT
- Twiddle factor tables (sine/cosine)
- Bit-reversal permutation
- In-place butterfly operations

#### 8c. Magnitude Calculation and Binning
- Calculate magnitude from complex results
- Accumulate into 4 configurable bins
- Write results to shared memory ring buffer

#### 8d. Control Block Integration
- Read configuration from shared memory
- Support enable/disable from Go API
- Configurable bin boundaries
- Statistics tracking

**Key Requirements**:
- Sampling must never be disrupted by FFT
- If FFT overruns, skip that buffer and log warning
- Test with various buffer sizes (512, 1024, 2048)
- Verify 40 kHz sampling rate with oscilloscope or logic analyzer

**Git commits** (multiple):
- "Implement IEP timer-based 40 kHz sampling"
- "Add fixed-point FFT implementation"
- "Implement frequency binning and magnitude calculation"
- "Add control block and configuration support"

### Stage 9: Sound Profile Utility Command
**Additions**:
- `pru/audio/scripts/cmd/stream.go` - Stream sound profiles

**Command**:
- `audio-util stream` - Read and display live sound profile data
  - Show bass, mid-low, mid-high, treble levels
  - Update rate statistics
  - Visual bar graph (optional)

**Git commit**: "Add sound profile streaming utility command"

### Stage 10: Full System Deployment and Testing
**Actions**:
- Deploy complete PRU1 audio firmware
- Deploy utility with stream command
- Test with various music sources:
  - Bass-heavy tracks
  - High-frequency content
  - Dynamic range testing
- Measure and verify timing:
  - Sample rate accuracy (should be 40.000 kHz ±0.1%)
  - FFT processing time
  - Profile update rate
- Tune frequency bin boundaries for best results

**Validation checklist**:
- [ ] Consistent 40 kHz sampling (no jitter)
- [ ] FFT completes within buffer fill time
- [ ] Bass responds to low frequencies
- [ ] Treble responds to high frequencies
- [ ] No buffer overruns under normal operation
- [ ] Can enable/disable from utility
- [ ] Can adjust bin boundaries at runtime

**Git commit**: "Verify full PRU1 audio analysis system"

### Stage 11: Go API Integration
**Files to create/modify**:
- `api/audio/pru_sampler.go` - PRU1 interface (similar to pru_detector.go)
- `api/audio/models.go` - Sound profile structures
- `api/audio/manager.go` - Audio manager integration

**API additions**:
- Enable/disable PRU audio sampling
- Read sound profile stream
- Configure frequency bins
- Get statistics

**Integration points**:
- LED controller subscribes to sound profiles
- Modulate brightness/color based on frequency bins
- Graceful degradation if PRU audio unavailable

**Git commits** (multiple):
- "Add PRU1 audio sampler to Go API"
- "Integrate sound profiles with LED controller"
- "Add audio-reactive LED modes"

## Memory Safety Verification

Before implementing PRU1, verify no conflicts with PRU0:
- PRU0 uses: Shared memory 0x00010000-0x00011000 (4KB)
- PRU1 uses: Shared memory 0x00012000-0x00013000 (4KB)
- No overlap with PRU0 buffers
- Each PRU has dedicated local DRAM (8KB each)

## Testing Strategy

### Unit Tests (where applicable)
- FFT correctness (test with known signals)
- Bin accumulation logic
- Ring buffer wraparound
- Control block state machine

### Integration Tests
- PRU0 and PRU1 running simultaneously
- Go API accessing both PRU shared memories
- No memory corruption between PRUs
- Both systems maintain real-time performance

### Performance Tests
- Sustained 40 kHz sampling for hours
- FFT timing under various configurations
- Memory access patterns
- Buffer overrun conditions

## Risk Mitigation

### Risk: FFT Too Slow
- **Mitigation**: Optimize with fixed-point math, measure early
- **Fallback**: Reduce sample size (512 instead of 1024)
- **Fallback 2**: Skip alternating buffers (20 kHz effective, still useful)

### Risk: ADC Access from PRU Problematic
- **Mitigation**: Research PRU ADC access early (Stage 6)
- **Fallback**: Use IIO kernel driver with PRU polling shared memory
- **Fallback 2**: Keep Go-based sampling with PRU doing only FFT

### Risk: 40 kHz Sampling Unachievable
- **Mitigation**: Test early with timer-based sampling
- **Fallback**: Lower to 20 kHz (still useful for music)
- **Verify**: Use oscilloscope to measure actual timing

### Risk: Shared Memory Conflicts with PRU0
- **Mitigation**: Careful memory layout documentation
- **Verification**: Test both PRUs simultaneously early
- **Tool**: Memory dump utility to visualize usage

## Success Criteria

1. **Functional**:
   - Consistent 40 kHz ADC sampling from PRU1
   - FFT produces accurate frequency analysis
   - Sound profiles update at ~39 Hz (1024 samples @ 40 kHz)
   - Go API can read profiles reliably
   - LED colors modulate with music in real-time

2. **Performance**:
   - Zero sampling jitter
   - FFT processing time <10 ms
   - No buffer overruns during normal operation
   - <1% CPU usage in Go API for reading profiles

3. **Maintainability**:
   - Configurable bin boundaries without recompilation
   - Enable/disable via API
   - Clear debugging utilities
   - Comprehensive commit history

## References

### PRU Documentation
- TI PRU-ICSS Reference Guide (spruh73)
- BeagleBone Black PRU Guide
- PRU Cookbook: https://beagleboard.org/pru
- RemoteProc driver documentation (kernel 5.10)

### FFT Resources
- Cooley-Tukey FFT algorithm
- Fixed-point FFT implementations
- Twiddle factor generation

### Existing Codebase
- `pru/remote/pru0_ir_detector.c` - PRU0 reference
- `api/remote/pru_detector.go` - Memory-mapped PRU interface
- `api/sound/` - Current Go-based audio processing

## Session Recovery

If session is interrupted, reference this plan and ask:
"Continue PRU1 audio implementation from Stage [X]"

Current implementation stage will be tracked in git commits and todo list.
