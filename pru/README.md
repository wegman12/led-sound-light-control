# PRU High-Speed ADC Sampling for BeagleBone Black

This directory contains PRU (Programmable Real-time Unit) firmware for high-speed ADC sampling at 48 kHz from analog input AIN1 (P9_40).

## Overview

The PRU provides deterministic real-time sampling that's much faster and more reliable than the standard Linux IIO interface. This implementation achieves 48,000 samples per second with microsecond-level timing precision.

## Architecture

```
┌─────────────────────┐
│   PRU0 Firmware     │  ← Samples ADC at 48 kHz
│  (pru0_adc_sampler) │  ← Writes to shared memory
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  PRU Shared Memory  │  ← 4096-sample ring buffer
│    (0x4A310000)     │  ← Control block with indices
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Go Application    │  ← Reads via memory mapping
│   (pru_reader.go)   │  ← Processes samples
└─────────────────────┘
```

## Files

- **pru0_adc_sampler.c**: PRU firmware source code
- **AM335x_PRU.cmd**: Linker command file for TI compiler
- **Makefile**: Build system supporting both TI and GCC compilers
- **PRU-ADC-00A0.dts**: Device tree overlay for hardware configuration
- **PRU_SETUP.md**: Detailed setup instructions

## Quick Start

### On the BeagleBone Black:

1. **Initial Setup** (one-time):
   ```bash
   cd scripts
   sudo ./setup_beaglebone.sh
   sudo reboot
   ```

2. **Verify PRU is Running**:
   ```bash
   ./scripts/check_pru_status.sh
   ```

3. **Test PRU Sampling**:
   ```bash
   make build
   sudo ./bin/led-sound-light-control pru-test
   ```

## Hardware Connection

Connect your audio or analog signal to:
- **Pin P9_40** (AIN1) - Signal input
- **Pin P9_34** (AGND) - Analog ground

**IMPORTANT**: The BeagleBone Black ADC accepts 0-1.8V only. Do not exceed this voltage!

## Memory Layout

### Ring Buffer (Offset 0x0000)
- Size: 8192 bytes (4096 × 16-bit samples)
- Stores raw ADC values (0-4095 for 12-bit ADC)
- Circular buffer with write/read indices

### Control Block (Offset 0x2000)
```c
struct {
    uint32_t write_index;   // Current PRU write position
    uint32_t read_index;    // Current Go read position
    uint32_t sample_count;  // Total samples written
    uint32_t overrun_count; // Buffer overflows (reading too slow)
}
```

## Modifying Sample Rate

To change the sampling rate, edit `pru0_adc_sampler.c`:

```c
// For 96 kHz:
#define CYCLES_PER_SAMPLE (200000000 / 96000)

// For 20 kHz:
#define CYCLES_PER_SAMPLE (200000000 / 20000)
```

Then rebuild and deploy:
```bash
cd pru
make
sudo ../scripts/deploy_pru.sh
```

## Troubleshooting

### "Failed to open /dev/mem"
- Must run as root: `sudo ./bin/led-sound-light-control`

### "PRU state: offline"
- Check device tree overlay is loaded
- Verify `/boot/uEnv.txt` has PRU enabled
- Run: `sudo ./scripts/check_pru_status.sh`

### Buffer Overruns
If you see overrun warnings:
1. Reduce `sleepTime` in `reader.go`
2. Increase buffer size in PRU firmware
3. Lower the sampling rate
4. Optimize Go processing pipeline

### No ADC Readings (all zeros)
- Verify signal is connected to P9_40
- Check voltage is between 0-1.8V
- Ensure ground is connected (P9_34)
- Use multimeter to verify input voltage

## Performance Notes

- **PRU Clock**: 200 MHz (5ns per cycle)
- **ADC Conversion Time**: ~1-2 microseconds
- **Maximum Practical Rate**: ~100 kHz (limited by ADC hardware)
- **Recommended Rates**: 20-48 kHz for audio applications

## Development

### Rebuild PRU Firmware
```bash
cd pru
make clean
make
sudo make install
```

### Reload PRU Without Reboot
```bash
sudo ../scripts/deploy_pru.sh
```

### View PRU Memory in Real-Time
```bash
sudo watch -n 0.5 'dd if=/dev/mem bs=4 skip=$((0x4A310000/4 + 0x2000/4)) count=4 2>/dev/null | od -An -t u4'
```

## References

- [PRU Assembly Instruction Guide](http://processors.wiki.ti.com/index.php/PRU_Assembly_Instructions)
- [BeagleBone PRU Guide](https://beagleboard.org/pru)
- [AM335x TRM - PRU-ICSS](https://www.ti.com/lit/ug/spruh73q/spruh73q.pdf)
