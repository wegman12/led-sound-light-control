# I2S MEMS Microphone Integration - Refactor Plan

**Date:** 2025-12-17
**Branch:** feature/i2s-microphone
**Author:** Claude Code

---

## Executive Summary

This document outlines the plan to migrate from the current analog ADC-based audio input (Adafruit 1713 with MAX4466) to a digital I2S MEMS microphone (INMP441 or ICS-43434). This migration will significantly improve audio quality, SNR, and frequency response across the full music spectrum.

**Current Performance:**
- Overall SNR: 1.73 dB
- Mid-frequency (200-4000 Hz): 4-5 dB SNR ✅
- Bass (<200 Hz): -0.21 dB SNR ❌
- Treble (>4000 Hz): -2.17 dB SNR ❌
- Limited by analog microphone and ADC noise floor

**Target Performance:**
- Overall SNR: >10 dB (6x improvement)
- Flat frequency response: 50 Hz - 20 kHz
- Digital signal path eliminates ADC quantization noise
- Better consistency and reliability

---

## Hardware Changes

### Recommended Microphone

**Primary Choice: INMP441**
- I2S digital output (24-bit)
- SNR: 61 dB typical
- Frequency range: 50 Hz - 20 kHz (±3 dB)
- Cost: ~$5-8
- Widely available, good documentation

**Alternative: ICS-43434**
- Similar specs to INMP441
- SNR: 65 dB
- May have better bass response

### Wiring Diagram

```
INMP441 Module          BeagleBone Black
--------------          ----------------
VDD        →            P9_3 (3.3V)
GND        →            P9_1 (DGND)
SD (Data)  →            P9_28 (McASP0_AXR2 / I2S Data)
WS (L/R)   →            P9_29 (McASP0_FSX / I2S Frame Sync)
SCK (Clock)→            P9_31 (McASP0_ACLKX / I2S Bit Clock)
L/R        →            GND (for left channel) or VDD (for right)
```

**Pin Configuration:**
- **P9_28:** McASP0_AXR2 - I2S serial data input
- **P9_29:** McASP0_FSX - Word select (frame sync)
- **P9_31:** McASP0_ACLKX - Bit clock

### Hardware Removal

**Components to remove:**
- Adafruit 1713 microphone module
- Wiring to P9_39 (AIN0) and P9_40 (AIN1)
- Any associated pull-up/down resistors

**Components to keep:**
- BeagleBone Black and power supply
- LED control wiring (unchanged)
- Remote control PRU (PRU0) - no changes needed

---

## Software Architecture

### Current Architecture

```
┌─────────────────┐
│ Analog Mic      │
│ (Adafruit 1713) │
└────────┬────────┘
         │ Analog 0-1.8V
         ↓
┌─────────────────┐
│ BBB ADC (12-bit)│
│ AIN1 (P9_40)    │
└────────┬────────┘
         │ 40 kHz sampling
         ↓
┌─────────────────┐
│ PRU1            │
│ - ADC read      │
│ - FFT (256pt)   │
│ - Band extract  │
│ - Smoothing     │
└────────┬────────┘
         │ Band values
         ↓
┌─────────────────┐
│ Go API          │
│ - Read results  │
│ - LED control   │
└─────────────────┘
```

### New Architecture (Option A: ALSA + PRU)

```
┌─────────────────┐
│ I2S MEMS Mic    │
│ (INMP441)       │
└────────┬────────┘
         │ I2S Digital (24-bit)
         ↓
┌─────────────────┐
│ McASP Peripheral│
│ (Hardware I2S)  │
└────────┬────────┘
         │ DMA to kernel buffer
         ↓
┌─────────────────┐
│ ALSA Driver     │
│ /dev/snd/pcmC0D0c
└────────┬────────┘
         │ 48 kHz, 16-bit samples
         ↓
┌─────────────────┐
│ Go Audio Thread │
│ - Read ALSA     │
│ - Downsample?   │
│ - Write to PRU  │
└────────┬────────┘
         │ Shared memory
         ↓
┌─────────────────┐
│ PRU1            │
│ - Read buffer   │
│ - FFT (256pt)   │
│ - Band extract  │
│ - Smoothing     │
└────────┬────────┘
         │ Band values
         ↓
┌─────────────────┐
│ Go API          │
│ - Read results  │
│ - LED control   │
└─────────────────┘
```

**Key Changes:**
1. **Go goroutine** continuously reads from ALSA
2. **Go** writes samples to PRU shared memory circular buffer
3. **PRU** reads from buffer instead of ADC
4. **Sample rate:** 48 kHz (was 40 kHz)
5. **Bit depth:** 16-bit (was 12-bit)

### New Architecture (Option B: PRU Direct - Future Enhancement)

```
┌─────────────────┐
│ I2S MEMS Mic    │
│ (INMP441)       │
└────────┬────────┘
         │ I2S Digital
         ↓
┌─────────────────┐
│ McASP Registers │
│ (Memory-mapped) │
└────────┬────────┘
         │ Direct PRU access
         ↓
┌─────────────────┐
│ PRU1            │
│ - McASP control │
│ - I2S read      │
│ - FFT (256pt)   │
│ - Band extract  │
│ - Smoothing     │
└────────┬────────┘
         │ Band values
         ↓
┌─────────────────┐
│ Go API          │
│ - Read results  │
│ - LED control   │
└─────────────────┘
```

**Benefits:**
- Lowest latency
- Minimal ARM CPU usage
- Most similar to current design

**Drawbacks:**
- Complex McASP register programming
- Harder to debug
- Less flexible (sample rate changes require firmware rebuild)

**Recommendation:** Start with **Option A (ALSA)**, migrate to Option B later if needed.

---

## Implementation Plan

### Phase 1: Device Tree & ALSA Setup (Week 1)

**Goal:** Get I2S microphone recognized by Linux and accessible via ALSA.

**Tasks:**

1. **Create Device Tree Overlay** (`BB-I2S-MIC.dts`)
   ```dts
   /dts-v1/;
   /plugin/;

   / {
       compatible = "ti,beaglebone", "ti,beaglebone-black";

       part-number = "BB-I2S-MIC";
       version = "00A0";

       fragment@0 {
           target = <&am33xx_pinmux>;
           __overlay__ {
               i2s_pins: pinmux_i2s_pins {
                   pinctrl-single,pins = <
                       0x190 0x20  /* P9_31: mcasp0_aclkx, INPUT */
                       0x194 0x20  /* P9_29: mcasp0_fsx, INPUT */
                       0x19c 0x22  /* P9_28: mcasp0_axr2, INPUT */
                   >;
               };
           };
       };

       fragment@1 {
           target = <&mcasp0>;
           __overlay__ {
               pinctrl-names = "default";
               pinctrl-0 = <&i2s_pins>;
               status = "okay";

               op-mode = <0>;  /* MCASP_IIS_MODE */
               tdm-slots = <2>;
               serial-dir = <0 0 2 0>;  /* 2 = input on AXR2 */

               tx-num-evt = <1>;
               rx-num-evt = <1>;
           };
       };

       fragment@2 {
           target-path = "/";
           __overlay__ {
               sound {
                   compatible = "simple-audio-card";
                   simple-audio-card,name = "I2S-Microphone";
                   simple-audio-card,format = "i2s";
                   simple-audio-card,bitclock-master = <&dailink0_master>;
                   simple-audio-card,frame-master = <&dailink0_master>;

                   simple-audio-card,cpu {
                       sound-dai = <&mcasp0>;
                   };

                   dailink0_master: simple-audio-card,codec {
                       sound-dai = <&codec>;
                   };
               };

               codec: spdif-transmitter {
                   #sound-dai-cells = <0>;
                   compatible = "linux,spdif-dir";
               };
           };
       };
   };
   ```

2. **Compile and Install Device Tree**
   ```bash
   dtc -O dtb -o BB-I2S-MIC.dtbo -b 0 -@ BB-I2S-MIC.dts
   sudo cp BB-I2S-MIC.dtbo /lib/firmware/
   sudo sh -c "echo 'BB-I2S-MIC' > /sys/devices/platform/bone_capemgr/slots"
   ```

3. **Test ALSA Detection**
   ```bash
   arecord -l  # Should show new capture device
   arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 test.wav
   aplay test.wav  # Verify audio captured correctly
   ```

**Deliverables:**
- Device tree overlay file
- Installation script
- ALSA verification test script
- Documentation: `docs/I2S_HARDWARE_SETUP.md`

---

### Phase 2: Go ALSA Reader (Week 1-2)

**Goal:** Create Go service that reads from ALSA and makes samples available to PRU.

**Tasks:**

1. **Create ALSA Capture Package** (`api/audio/alsa_capture.go`)

   **Dependencies:**
   ```bash
   # Install ALSA development libraries
   sudo apt-get install libasound2-dev

   # Add to go.mod
   require github.com/yobert/alsa v0.0.0-20200618200352-d079056f5370
   ```

   **Interface:**
   ```go
   package audio

   type ALSACapture struct {
       device      *alsa.Device
       sampleRate  int
       bufferSize  int
       callback    func([]int16)
   }

   func NewALSACapture(deviceName string, sampleRate int, callback func([]int16)) (*ALSACapture, error)
   func (a *ALSACapture) Start() error
   func (a *ALSACapture) Stop() error
   func (a *ALSACapture) Close() error
   ```

2. **Create PRU Sample Writer** (`api/audio/pru_sample_writer.go`)

   **Circular buffer protocol:**
   ```go
   // Shared memory layout (same 12KB space)
   // Offset 0x0000: Control block (256 bytes)
   // Offset 0x0100: Sample buffer (6016 samples)

   type AudioControlBlock struct {
       WriteIndex    uint32  // Updated by Go
       ReadIndex     uint32  // Updated by PRU
       SampleRate    uint32  // 48000
       Overruns      uint32  // Count of buffer overruns
       Status        uint32  // Status flags
   }

   func (w *PRUSampleWriter) WriteSamples(samples []int16) error {
       // Write to circular buffer
       // Update write index atomically
       // Check for overruns
   }
   ```

3. **Integrate into Service** (`api/server/audio_service.go`)

   ```go
   type AudioService struct {
       alsa      *audio.ALSACapture
       pruWriter *audio.PRUSampleWriter
   }

   func (s *AudioService) Start() error {
       // Start ALSA capture
       // Route samples to PRU
       // Monitor for errors/overruns
   }
   ```

**Deliverables:**
- `api/audio/alsa_capture.go`
- `api/audio/pru_sample_writer.go`
- `api/server/audio_service.go`
- Unit tests
- Integration test: ALSA → PRU → verify samples

---

### Phase 3: PRU Firmware Updates (Week 2)

**Goal:** Update PRU1 firmware to read samples from shared memory instead of ADC.

**Tasks:**

1. **Create New Firmware** (`pru/audio/pru1_audio_i2s.c`)

   **Key changes:**
   ```c
   // Remove ADC initialization
   // Remove IEP timer (Go provides timing)

   // New: Shared memory buffer read
   #define SAMPLE_RATE_HZ      48000  // Updated from 40000
   #define AUDIO_BUFFER_SIZE   6016   // Same as raw capture

   struct audio_control_block {
       volatile uint32_t write_index;  // Updated by Go
       volatile uint32_t read_index;   // Updated by PRU
       volatile uint32_t sample_rate;
       volatile uint32_t overruns;
       volatile uint32_t status;
   };

   void main(void) {
       struct audio_control_block *ctrl = ...;
       int16_t *sample_buffer = ...;

       while (1) {
           // Check if new samples available
           if (ctrl->write_index != ctrl->read_index) {
               // Read samples from circular buffer
               // Accumulate samples for FFT window (512 samples @ 48kHz = 10.67ms)
               // When window full, perform FFT
               // Extract bands and smooth
               // Update read_index
           }

           // Small delay to avoid busy loop
           __delay_cycles(1000);
       }
   }
   ```

2. **Update FFT Frequency Bins**

   ```c
   // Old: 40000 Hz / 256 bins = 156.25 Hz per bin
   // New: 48000 Hz / 256 bins = 187.5 Hz per bin

   // Update band boundaries:
   // Bass: 0-200 Hz = bins 0-1 (was 0-1)
   // MidLow: 200-1500 Hz = bins 1-8 (was 1-9)
   // MidHigh: 1500-4000 Hz = bins 8-21 (was 9-25)
   // Treble: 4000-20000 Hz = bins 21-106 (was 25-128)
   ```

3. **Test with Synthetic Data**
   - Create Go test that writes known waveforms to buffer
   - Verify PRU FFT output matches expected frequencies
   - Validate band extraction and smoothing

**Deliverables:**
- `pru/audio/pru1_audio_i2s.c`
- Updated Makefile with `i2s-deploy-all` target
- Unit tests (synthetic waveform validation)
- Performance measurements (latency, CPU usage)

---

### Phase 4: Integration & Testing (Week 3)

**Goal:** End-to-end integration with real I2S microphone.

**Tasks:**

1. **Hardware Validation**
   - Wire INMP441 to BeagleBone
   - Verify I2S signals with oscilloscope/logic analyzer (optional)
   - Test ALSA capture quality

2. **Software Integration**
   - Deploy device tree, Go service, PRU firmware
   - Verify sample flow: Mic → ALSA → Go → PRU → FFT → Bands
   - Monitor for buffer overruns or timing issues

3. **Performance Testing**
   - Measure end-to-end latency
   - CPU usage profiling (Go + PRU)
   - Memory usage analysis
   - Stress test: loud music, silence, white noise

4. **Audio Quality Analysis**
   - Capture new baseline and music samples
   - Run optimize_audio_params.py
   - Compare SNR with old ADC system
   - **Target: >10 dB SNR overall, good performance 50Hz-20kHz**

5. **LED Response Testing**
   - Play music with known frequency content
   - Verify LED bands respond correctly
   - Check responsiveness (no lag/stuttering)
   - Visual quality assessment

**Deliverables:**
- Integration test suite
- Performance benchmarks
- Audio quality comparison report
- LED visualization validation
- User acceptance testing

---

### Phase 5: Documentation & Deployment (Week 3-4)

**Goal:** Production-ready system with complete documentation.

**Tasks:**

1. **Documentation**
   - Hardware setup guide with photos/diagrams
   - Software installation instructions
   - Configuration options
   - Troubleshooting guide
   - Migration guide from ADC system

2. **Deployment Automation**
   - Ansible/shell scripts for full deployment
   - Systemd service configuration
   - Auto-start on boot
   - Health monitoring

3. **Rollback Plan**
   - Document how to revert to ADC system
   - Keep old firmware available
   - Branch management strategy

4. **Code Review & Cleanup**
   - Remove dead code
   - Add comments and documentation
   - Code quality checks
   - Security review

**Deliverables:**
- `docs/I2S_HARDWARE_SETUP.md`
- `docs/I2S_SOFTWARE_SETUP.md`
- `docs/MIGRATION_GUIDE.md`
- `docs/TROUBLESHOOTING.md`
- Deployment scripts
- Updated README.md

---

## Testing Strategy

### Unit Tests

**Go:**
- ALSA capture mock tests
- PRU sample writer buffer logic
- Circular buffer wrap-around edge cases
- Overrun detection and recovery

**PRU:**
- FFT correctness (synthetic waveforms)
- Band extraction accuracy
- Smoothing algorithm
- Buffer read logic

### Integration Tests

1. **ALSA → Go:** Verify samples read correctly from ALSA device
2. **Go → PRU:** Verify samples written to shared memory correctly
3. **PRU → Go:** Verify band results read correctly
4. **End-to-end:** Known audio input → expected LED output

### Performance Tests

- **Latency:** Measure delay from microphone to LED response
  - Target: <50ms total latency
- **CPU Usage:** Monitor ARM and PRU load
  - Target: ARM <10%, PRU <80%
- **Reliability:** 24-hour stress test with continuous music
  - Target: Zero crashes, <1 overrun per hour

### Audio Quality Tests

- **SNR Measurement:** Compare baseline vs music samples
  - Target: >10 dB overall
- **Frequency Response:** Sweep 50 Hz - 20 kHz, measure band response
  - Target: All bands responsive and proportional
- **Dynamic Range:** Test quiet and loud audio
  - Target: LEDs respond to full dynamic range

---

## Risk Assessment

### High Risk Items

1. **Device Tree Complexity**
   - Risk: McASP configuration incorrect, audio not captured
   - Mitigation: Start with known-working examples, test incrementally
   - Contingency: Use USB audio adapter as temporary solution

2. **Real-time Performance**
   - Risk: Go ALSA reader can't keep up with 48 kHz, buffer overruns
   - Mitigation: Use buffered I/O, dedicated goroutine, high priority
   - Contingency: Reduce sample rate to 32 kHz or 24 kHz

3. **PRU Memory Constraints**
   - Risk: Not enough shared memory for sample buffer + FFT
   - Mitigation: Profile memory usage, optimize buffer sizes
   - Contingency: Use smaller FFT (128-point) or smaller sample buffer

### Medium Risk Items

4. **Sample Rate Mismatch**
   - Risk: 48 kHz breaks PRU FFT timing assumptions
   - Mitigation: Update all constants, test thoroughly
   - Contingency: Downsample to 40 kHz in Go before writing to PRU

5. **Integration Complexity**
   - Risk: Many moving parts, hard to debug when issues occur
   - Mitigation: Test each component individually before integration
   - Contingency: Keep ADC firmware available for comparison

### Low Risk Items

6. **Hardware Availability**
   - Risk: INMP441 out of stock or takes time to arrive
   - Mitigation: Order from multiple suppliers
   - Contingency: Use alternative (ICS-43434, SPH0645)

7. **Audio Quality Regression**
   - Risk: I2S system performs worse than expected
   - Mitigation: Extensive testing before full migration
   - Contingency: Revert to ADC system, investigate issues

---

## Migration Strategy

### Parallel Operation Period

**Week 1-2:** Run both systems side-by-side
- Keep ADC system operational
- Add I2S system as alternative audio source
- Compare results in real-time
- Use feature flag to switch between systems

**Configuration:**
```yaml
# /etc/led-sound-light-control/config.yaml
audio:
  source: "i2s"  # or "adc"
  i2s:
    device: "hw:0,0"
    sample_rate: 48000
  adc:
    channel: 1  # AIN1
    sample_rate: 40000
```

### Cutover Plan

**Week 3:** Switch to I2S as primary
- Monitor for issues
- Keep ADC as backup
- Document any problems

**Week 4:** Deprecate ADC
- Remove ADC hardware
- Archive ADC firmware
- Update documentation

---

## Success Criteria

### Must Have (P0)

- [ ] I2S microphone recognized by ALSA
- [ ] 48 kHz audio captured without dropouts
- [ ] PRU receives samples via shared memory
- [ ] FFT and band extraction working correctly
- [ ] LEDs respond to music in real-time
- [ ] Overall SNR >5 dB (3x improvement over ADC)
- [ ] System stable for >24 hours continuous operation

### Should Have (P1)

- [ ] Overall SNR >10 dB (6x improvement)
- [ ] All frequency bands (50Hz-20kHz) responsive
- [ ] End-to-end latency <50ms
- [ ] ARM CPU usage <10%
- [ ] Complete documentation
- [ ] Automated deployment scripts

### Nice to Have (P2)

- [ ] Multiple microphone support
- [ ] Dynamic sample rate adjustment
- [ ] Web UI for audio monitoring
- [ ] Spectrum analyzer visualization
- [ ] Remote debugging tools

---

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 1. Device Tree & ALSA | 3-5 days | I2S mic working in Linux |
| 2. Go ALSA Reader | 3-4 days | ALSA → PRU sample flow |
| 3. PRU Firmware | 2-3 days | PRU reads from Go buffer |
| 4. Integration & Testing | 4-5 days | End-to-end validation |
| 5. Documentation | 2-3 days | Production ready |
| **Total** | **2-3 weeks** | Full I2S system |

---

## Resources

### Hardware

- **INMP441 Module:** ~$5-8 (Amazon, Adafruit, SparkFun)
- **Jumper wires:** 5x female-to-female
- **Optional:** Logic analyzer for I2S debugging

### Software

- **Go Libraries:**
  - `github.com/yobert/alsa` - ALSA bindings
  - Or `github.com/gordonklaus/portaudio` - Cross-platform audio
- **Linux Packages:**
  - `libasound2-dev` - ALSA development
  - `alsa-utils` - Testing tools

### Documentation

- BeagleBone Black System Reference Manual (McASP section)
- AM335x Technical Reference Manual (McASP peripheral)
- INMP441 Datasheet
- Linux ALSA Project documentation
- Device Tree documentation for BeagleBone

### Example Code

- BeagleBone audio cape device trees (reference examples)
- PRU shared memory examples from TI
- ALSA capture examples in Go

---

## Next Steps

1. **Order Hardware:** Purchase INMP441 module (or ICS-43434 alternative)
2. **Phase 1 Start:** Begin device tree development and testing
3. **Parallel Work:** Start Go ALSA reader development in parallel
4. **Checkpoint:** After Phase 1, validate I2S microphone working in Linux
5. **Continue:** Proceed through phases 2-5 as outlined

---

## Questions & Decisions Needed

- [ ] Confirm INMP441 as microphone choice (vs ICS-43434)
- [ ] Approve ALSA approach (vs PRU direct I2S)
- [ ] Confirm 48 kHz sample rate acceptable (vs 40 kHz)
- [ ] Review device tree pin assignments
- [ ] Decide on migration timeline (aggressive vs conservative)

---

**Status:** Ready to begin Phase 1 upon hardware arrival

**Last Updated:** 2025-12-17
