# I2S Microphone Solutions Guide

## Problem Summary

The AM335x McASP hardware **cannot generate standard audio clock rates (44.1kHz, 48kHz) using only internal clocks**. The internal AUXCLK is derived from a 24 MHz oscillator, which doesn't divide evenly into standard audio rates. When clock dividers are released from reset, the hardware clears configuration bits 19-20 in the AHCLKXCTL and ACLKXCTL registers, likely indicating the clock source is invalid.

## Solution 1: External Audio Oscillator (RECOMMENDED) ⭐

### Why This Solution

- **Standard approach**: Used in all production BeagleBone audio capes (HDMI audio, audio capes, etc.)
- **Hardware-proven**: Leverages McASP's DMA and low-latency capabilities
- **TI-recommended**: [TI documentation](https://e2e.ti.com/support/processors-group/processors/f/processors-forum/607500/linux-processor-sdk-am335x-mcasp-master-mode-configuration) explicitly states external clock is required
- **Low complexity**: Simple hardware addition, minimal software changes

### Hardware Requirements

**External Crystal Oscillator**:
- **Frequency**: 24.576 MHz (preferred) or 12.288 MHz
- **Type**: Active CMOS oscillator (not passive crystal)
- **Voltage**: 3.3V compatible
- **Package**: 4-pin (typical)

**Recommended Parts** (available from Mouser, DigiKey, LCSC):
| Part Number | Frequency | Voltage | Package | Cost |
|-------------|-----------|---------|---------|------|
| Abracon ASFLMB-24.576MHZ-LC-T | 24.576 MHz | 3.3V | 4-pin SMD | ~$1.50 |
| CTS CB3LV-3C-24M5760 | 24.576 MHz | 3.3V | 4-pin SMD | ~$1.50 |
| ECS-2520MV-246-BN | 24.576 MHz | 3.3V | 4-pin SMD | ~$0.50 |

**Why 24.576 MHz?**
```
24.576 MHz ÷ 512 = 48.000 kHz (exact)
24.576 MHz ÷ 256 = 96.000 kHz (exact)
24.576 MHz ÷ 128 = 192.000 kHz (exact)

Also works for 44.1 kHz family:
22.5792 MHz ÷ 512 = 44.100 kHz (exact)
```

### Hardware Connection Diagram

```
┌──────────────────────┐
│  24.576 MHz          │
│  Oscillator          │
│  ┌────────────┐      │
│  │ Pin 1: EN  │──────┼──> 3.3V (P9_03) - Always enabled
│  │ Pin 2: GND │──────┼──> GND  (P9_01)
│  │ Pin 3: OUT │──────┼──> P9_12 (AHCLKX) - Clock to McASP
│  │ Pin 4: VCC │──────┼──> 3.3V (P9_03)
│  └────────────┘      │
└──────────────────────┘
         │
         │ Clock signal
         v
┌────────────────────────────────────────────┐
│ BeagleBone Black - McASP0                  │
│                                            │
│  P9_12 (AHCLKX IN)  ← 24.576 MHz          │
│  P9_29 (FSX OUT)    → INMP441 WS          │
│  P9_30 (AXR0 IN)    ← INMP441 SD          │
│  P9_31 (ACLKX OUT)  → INMP441 SCK         │
│                                            │
│  McASP divides 24.576 MHz by 8 = 3.072 MHz│
│  3.072 MHz ÷ 64 = 48 kHz sample rate      │
└────────────────────────────────────────────┘
```

### Breadboard Assembly

**Step 1**: Connect oscillator power
```
Oscillator VCC (Pin 4) → BeagleBone P9_03 (3.3V)
Oscillator GND (Pin 2) → BeagleBone P9_01 (GND)
```

**Step 2**: Enable oscillator (always-on configuration)
```
Oscillator EN (Pin 1) → Connect to VCC (Pin 4)
```

**Step 3**: Connect clock output
```
Oscillator OUT (Pin 3) → BeagleBone P9_12
                         + Optional: 10kΩ pull-down resistor to GND
```

**Step 4**: Connect INMP441 microphone
```
INMP441 VCC → BeagleBone 3.3V (P9_03)
INMP441 GND → BeagleBone GND (P9_01)
INMP441 WS  → BeagleBone P9_29
INMP441 SD  → BeagleBone P9_30
INMP441 SCK → BeagleBone P9_31
INMP441 L/R → GND (left channel) or 3.3V (right channel)
```

### Software Configuration

The device tree overlay has been updated in `hardware/device-tree/BB-I2S-MIC-00A0.dts` to support the external clock.

**Key Changes**:
1. Added `clk_mcasp0_fixed` node defining the 24.576 MHz clock
2. Configured P9_12 as AHCLKX input
3. Changed clock reference from `mcasp0_fck` to `clk_mcasp0_fixed`

**Installation**:
```bash
# On BeagleBone
cd /path/to/hardware/device-tree
dtc -@ -I dts -O dtb -o BB-I2S-MIC-00A0.dtbo BB-I2S-MIC-00A0.dts
sudo cp BB-I2S-MIC-00A0.dtbo /lib/firmware/
sudo reboot

# After reboot, check audio card
arecord -l
```

**Testing**:
```bash
# Record 5 seconds of audio
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 test.wav

# Play it back (if you have speakers)
aplay test.wav

# Check recording level
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 -vv test.wav
```

### Expected Results

With the external clock:
- ✅ McASP registers should retain their values after clock reset
- ✅ AHCLKXCTL and ACLKXCTL bits 19-20 should remain set
- ✅ I2S clocks (BCLK, WS) should appear on oscilloscope
- ✅ Audio data should be non-zero (mic picks up room noise)

### Cost Analysis

**Total Hardware Cost**: $1-5 depending on oscillator choice

**Parts Needed**:
- 24.576 MHz oscillator: $0.50 - $2.00
- Breadboard (if not owned): $2-3
- Jumper wires (if not owned): $1-2

**Development Time**: 1-2 hours (hardware assembly + testing)

---

## Solution 2: PRU-Based I2S Bit-Banging (ADVANCED)

### Overview

The PRU (Programmable Real-Time Unit) can directly control GPIO pins to generate and read I2S signals with precise timing, bypassing the McASP hardware entirely.

### What is the PRU?

The AM335x contains two **PRU cores** (Programmable Real-Time Units):
- 32-bit RISC processors running at 200 MHz
- Independent from main ARM CPU (no Linux interference)
- Deterministic execution (no interrupts, caches, or OS)
- Direct access to all GPIO pins
- 5 ns cycle time (200 MHz = single cycle GPIO toggle)
- 12 KB instruction RAM, 8 KB data RAM per PRU
- Shared memory for communication with Linux

**PRU Block Diagram**:
```
┌─────────────────────────────────────────────┐
│ AM335x SoC                                  │
│                                             │
│  ┌──────────────┐      ┌──────────────┐    │
│  │ ARM Cortex-A8│      │ PRU-ICSS     │    │
│  │ (Linux)      │◄────►│ ┌─────────┐  │    │
│  │ 1 GHz        │ Shared│ │ PRU0    │  │    │
│  └──────────────┘ Memory│ │ 200 MHz │◄─┼───┼──> GPIO pins
│                         │ └─────────┘  │    │   (direct access)
│                         │ ┌─────────┐  │    │
│                         │ │ PRU1    │  │    │
│                         │ │ 200 MHz │◄─┼───┼──> GPIO pins
│                         │ └─────────┘  │    │
│                         └──────────────┘    │
└─────────────────────────────────────────────┘
```

### How PRU Generates I2S

**I2S Timing Requirements (48 kHz, 32-bit stereo)**:
- Bit clock (BCLK): 3.072 MHz (48000 × 2 channels × 32 bits)
- Frame sync (WS): 48 kHz (toggles every 32 bits)
- PRU cycles available: 200 MHz ÷ 3.072 MHz = **65 cycles per bit**

**PRU I2S Receive Pseudo-Code**:
```c
// PRU firmware (assembly or C)
#define BCLK_PIN  (1 << 31)  // P9_31
#define WS_PIN    (1 << 29)  // P9_29
#define SD_PIN    (1 << 30)  // P9_30

uint32_t sample_buffer[1024];  // Shared memory with Linux
uint32_t buffer_index = 0;

while (1) {
    uint32_t left_sample = 0;
    uint32_t right_sample = 0;

    // Wait for WS to go low (left channel)
    while (read_gpio() & WS_PIN);

    // Read 32 bits of left channel
    for (int i = 0; i < 32; i++) {
        set_gpio(BCLK_PIN);              // BCLK high
        __delay_cycles(32);              // Wait half bit period
        left_sample = (left_sample << 1) | ((read_gpio() & SD_PIN) ? 1 : 0);
        clear_gpio(BCLK_PIN);            // BCLK low
        __delay_cycles(32);              // Wait half bit period
    }

    // Wait for WS to go high (right channel)
    while (!(read_gpio() & WS_PIN));

    // Read 32 bits of right channel
    for (int i = 0; i < 32; i++) {
        set_gpio(BCLK_PIN);
        __delay_cycles(32);
        right_sample = (right_sample << 1) | ((read_gpio() & SD_PIN) ? 1 : 0);
        clear_gpio(BCLK_PIN);
        __delay_cycles(32);
    }

    // Store samples to shared buffer
    sample_buffer[buffer_index++] = left_sample;
    if (buffer_index >= 1024) {
        // Signal Linux to read buffer
        send_interrupt_to_arm();
        buffer_index = 0;
    }
}
```

### Existing PRU Audio Projects

**1. [Bela Platform](https://github.com/BelaPlatform/Bela)**
- **Most mature PRU audio project**
- Professional ultra-low-latency audio environment
- Uses PRU for buffer management, still uses McASP for I2S
- Achieves <1ms audio latency
- **Lesson**: Even Bela uses McASP hardware, PRU just optimizes buffer handling

**2. [pru-mic (PDM Microphones)](https://github.com/mkj/pru-mic)**
- Bit-bangs PDM (1-bit) microphone input via PRU
- PRU generates bit clock and reads 8 PDM streams
- Runs at 2.4 MHz bit clock
- **Demonstrates**: PRU can handle high-speed digital audio

**3. [beaglemic (Microphone Array)](https://github.com/dinuxbg/beaglemic)**
- PDM microphone array for PocketBeagle
- 4-8 microphone channels
- PRU handles all timing and data capture
- **Demonstrates**: Multi-channel capability

**4. [PRU-Audio-Processing (CIC Filter)](https://github.com/Scrashdown/PRU-Audio-Processing)**
- 1-bit audio input with CIC filtering on PRU
- Shows PRU can do real-time DSP
- **Demonstrates**: PRU processing power

### Architecture for Your Use Case

**System Design**:
```
┌─────────────────────────────────────────────────────┐
│ User Application (Python, C, etc.)                  │
│  - Read audio samples via /dev/mem or character dev │
└────────────────┬────────────────────────────────────┘
                 │ ioctl/mmap
┌────────────────▼────────────────────────────────────┐
│ Linux Kernel Module (Optional)                      │
│  - Character device /dev/pru_i2s                    │
│  - Maps PRU shared memory to userspace              │
│  - Handles interrupts from PRU                      │
└────────────────┬────────────────────────────────────┘
                 │ RemoteProc
┌────────────────▼────────────────────────────────────┐
│ PRU Firmware (pru_i2s.elf)                          │
│  - Bit-bang I2S on GPIO pins                        │
│  - Generate BCLK and WS                             │
│  - Sample SD pin on BCLK edges                      │
│  - Write samples to shared memory                   │
│  - Send interrupt when buffer full                  │
└────────────────┬────────────────────────────────────┘
                 │ Direct GPIO
┌────────────────▼────────────────────────────────────┐
│ INMP441 I2S Microphone                              │
│  WS  ← PRU GPIO                                     │
│  SCK ← PRU GPIO                                     │
│  SD  → PRU GPIO                                     │
└─────────────────────────────────────────────────────┘
```

### Implementation Steps

**1. PRU Development Environment Setup**
```bash
# Install PRU compiler and tools
sudo apt-get install ti-pru-cgt-installer
sudo apt-get install am335x-pru-package

# Clone PRU software support package
git clone https://github.com/beagleboard/pru-software-support-package.git
cd pru-software-support-package
```

**2. Write PRU Firmware** (pru_i2s.c)
```c
#include <stdint.h>
#include <pru_cfg.h>
#include <pru_intc.h>

// GPIO registers (simplified)
#define GPIO1_BASE 0x4804C000
#define GPIO_SETDATAOUT  0x194
#define GPIO_CLEARDATAOUT 0x190
#define GPIO_DATAIN 0x138

// Pin definitions (adjust for your setup)
#define BCLK_PIN  (1 << 31)  // Example: GPIO1_31
#define WS_PIN    (1 << 29)  // Example: GPIO1_29
#define SD_PIN    (1 << 30)  // Example: GPIO1_30

// Shared memory for samples
#define SAMPLE_BUFFER_SIZE 1024
volatile uint32_t samples[SAMPLE_BUFFER_SIZE] __attribute__((section(".shared_mem")));
volatile uint32_t buffer_ready = 0;

void main(void) {
    uint32_t buffer_index = 0;

    // Configure pins (pseudo-code - actual PRU GPIO setup is more complex)
    // ...

    while (1) {
        // I2S bit-banging loop
        // (Implementation details omitted for brevity)
        // See full example in PRU examples

        // When buffer full, signal Linux
        if (buffer_index >= SAMPLE_BUFFER_SIZE) {
            buffer_ready = 1;
            __R31 = 35;  // Send interrupt to ARM
            buffer_index = 0;
            while (buffer_ready);  // Wait for Linux to acknowledge
        }
    }
}
```

**3. Compile PRU Firmware**
```bash
clpru -v3 -O2 --hardware_mac=on pru_i2s.c -o pru_i2s.out
```

**4. Load PRU Firmware from Linux**
```bash
# Copy firmware to lib/firmware
sudo cp pru_i2s.out /lib/firmware/am335x-pru0-fw

# Load PRU via RemoteProc
echo 'am335x-pru0-fw' | sudo tee /sys/class/remoteproc/remoteproc1/firmware
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state
```

**5. Read Samples from Linux**
```python
import mmap
import struct

# Map PRU shared memory
PRU_SHARED_MEM = 0x4A310000  # PRU shared RAM address
SAMPLE_BUFFER_SIZE = 1024

with open('/dev/mem', 'r+b') as f:
    mem = mmap.mmap(f.fileno(), SAMPLE_BUFFER_SIZE * 4,
                    offset=PRU_SHARED_MEM)

    while True:
        # Check if buffer ready
        buffer_ready = struct.unpack('I', mem[0:4])[0]
        if buffer_ready:
            # Read samples
            samples = struct.unpack(f'{SAMPLE_BUFFER_SIZE}I',
                                   mem[4:SAMPLE_BUFFER_SIZE*4+4])

            # Process samples...
            process_audio(samples)

            # Clear ready flag
            mem[0:4] = struct.pack('I', 0)
```

### Advantages of PRU Approach

✅ **No external hardware** - software-only solution
✅ **Flexible** - can generate any clock frequency (within limits)
✅ **Multi-channel** - PRU can handle multiple I2S streams
✅ **Learning opportunity** - deep understanding of I2S and embedded systems
✅ **Custom protocols** - can implement non-standard formats

### Disadvantages of PRU Approach

❌ **High complexity** - requires PRU assembly/C, RemoteProc, kernel drivers
❌ **Development time** - 2-4 weeks for experienced developer, longer for beginners
❌ **CPU overhead** - Linux must continuously poll PRU shared memory (no DMA)
❌ **Limited sample rates** - 200 MHz PRU limits maximum frequency
❌ **Jitter** - slight timing variations compared to hardware I2S
❌ **No ALSA integration** - would need custom kernel driver for ALSA support
❌ **Debugging difficulty** - PRU debugging tools are limited
❌ **Pin limitations** - PRU can only access certain GPIO pins

### Realistic Assessment

**For your single INMP441 microphone at 48 kHz**:

**Feasibility**: ✅ Technically possible
- 48 kHz × 2 channels × 32 bits = 3.072 MHz
- PRU: 200 MHz ÷ 3.072 MHz = 65 cycles per bit
- **Verdict**: Enough cycles, but tight timing

**Effort vs Benefit**:
- External oscillator: **$1-5, 1-2 hours** ⭐
- PRU implementation: **$0, 2-4 weeks, steep learning curve**

**Recommendation**: Only pursue PRU approach if:
1. You want to learn PRU programming
2. You need multiple I2S inputs (>2 microphones)
3. You need custom audio protocols
4. Cost is absolutely critical ($1-5 is too much)

### Learning Resources for PRU

If you decide to pursue the PRU route:

- [TI PRU Training](https://training.ti.com/ti-pru-training-introduction)
- [BeagleBone PRU Coding in C](https://catch22eu.github.io/website/beaglebone/beaglebone-pru-c/)
- [PRU Software Support Package](https://github.com/beagleboard/pru-software-support-package)
- [Bela Source Code](https://github.com/BelaPlatform/Bela) (reference implementation)

---

## Solution Comparison

| Aspect | External Oscillator | PRU Bit-Bang |
|--------|-------------------|--------------|
| **Cost** | $1-5 | $0 |
| **Development Time** | 1-2 hours | 2-4 weeks |
| **Difficulty** | Beginner | Advanced |
| **Reliability** | Excellent | Good |
| **Latency** | <1ms (DMA) | 1-5ms (polling) |
| **CPU Usage** | Minimal | Moderate |
| **Sample Rate** | Up to 192 kHz | Up to ~100 kHz |
| **Multi-channel** | 2 channels (McASP0) | 4+ channels possible |
| **ALSA Support** | Built-in | Custom driver needed |
| **Maintenance** | None | Ongoing |

## Recommendation

**For your project**: Use the **external oscillator solution** ⭐

**Reasons**:
1. **Proven**: Same approach as commercial BeagleBone audio capes
2. **Simple**: Hardware is straightforward, software already written
3. **Fast**: 1-2 hours vs 2-4 weeks
4. **Reliable**: Hardware I2S with DMA is rock-solid
5. **Standard**: Uses ALSA, works with all Linux audio tools

**When to use PRU**:
- Learning PRU programming
- Need >2 microphones
- Need custom audio protocol (not standard I2S)
- Building commercial product with extreme cost constraints

---

## Next Steps

### For External Oscillator Approach

1. **Order oscillator** (~1 week shipping)
   - Search DigiKey/Mouser/LCSC for "24.576 MHz oscillator 3.3V"
   - Choose 4-pin CMOS type
   - Order 2-3 (in case of damage)

2. **Test oscillator output** (when arrived)
   ```bash
   # Check oscillator is working with logic analyzer or oscilloscope
   # Should see 24.576 MHz square wave on output pin
   ```

3. **Connect hardware** (15 minutes)
   - Follow breadboard diagram above
   - Double-check VCC/GND connections

4. **Update device tree** (already done)
   ```bash
   # Compile and install (on BeagleBone)
   cd hardware/device-tree
   dtc -@ -I dts -O dtb -o BB-I2S-MIC-00A0.dtbo BB-I2S-MIC-00A0.dts
   sudo cp BB-I2S-MIC-00A0.dtbo /lib/firmware/
   sudo reboot
   ```

5. **Test and verify**
   ```bash
   # Check card appeared
   arecord -l

   # Record test
   arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 test.wav

   # Verify audio data (should see waveform, not all zeros)
   sox test.wav -n stat
   ```

6. **Debug if needed**
   - Check oscillator with oscilloscope/logic analyzer
   - Verify voltage is present on P9_12
   - Check kernel messages: `dmesg | grep mcasp`

### For PRU Approach

1. **Study PRU basics** (1-2 weeks)
   - Complete TI PRU training modules
   - Run simple PRU examples (LED blink, etc.)

2. **Set up PRU development** (2-3 days)
   - Install PRU compiler
   - Build and load example PRU firmware

3. **Implement I2S bit-bang** (1-2 weeks)
   - Write PRU firmware for I2S timing
   - Test with oscilloscope

4. **Create Linux interface** (3-5 days)
   - Kernel module or userspace /dev/mem access
   - Buffer management

5. **Integrate and test** (3-5 days)
   - End-to-end testing
   - Performance optimization

**Estimated total**: 3-6 weeks depending on experience

---

## Conclusion

The **external oscillator solution is strongly recommended** for your project. It's the standard approach used in all production audio systems, takes minimal time to implement, and provides excellent audio quality with low latency.

The PRU approach is educational and technically interesting, but represents significant development effort for marginal benefit in your single-microphone use case.

**Device tree is ready** - you just need the $1.50 oscillator chip!
