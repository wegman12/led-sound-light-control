# PRU1 Audio Sampling System

High-precision audio sampling and FFT analysis on BeagleBone Black PRU1.

## Quick Start

### Build Locally
```bash
# Build PRU1 firmware (requires clpru on BeagleBone)
make

# Build utility for local testing
make build-util

# Build utility for BeagleBone
make build-util-bbb
```

### Deploy to BeagleBone
```bash
# Deploy everything (firmware + utility) - RECOMMENDED
make deploy-all

# This will:
# 1. Copy source files to BeagleBone
# 2. Build firmware on BeagleBone (has clpru installed)
# 3. Install firmware to /lib/firmware/am335x-pru1-fw
# 4. Load PRU1 with new firmware
# 5. Deploy utility application
```

### Test on BeagleBone
```bash
ssh bbb1.wegman
cd ~/led-sound-light-control/pru-audio/
sudo ./audio-util test
```

Expected output:
```
PRU1 Status: 0x41554431 (AUD1)

Monitoring PRU1 toggle bit (Ctrl+C to exit)...

[12:34:56] Counter:     1 | Toggle: 0
[12:34:57] Counter:     2 | Toggle: 1 ✓ TOGGLED (+1)
[12:34:58] Counter:     3 | Toggle: 0 ✓ TOGGLED (+1)
```

## Project Structure

```
pru/audio/
├── PROJECT_PLAN.md          # Comprehensive implementation plan
├── README.md                # This file
├── Makefile                 # Build and deployment system
├── pru1_audio.c             # PRU1 firmware source
├── resource_table_empty.h   # RemoteProc resource table
├── am335x_pru.cmd           # PRU1 linker script
├── gen/                     # Build output (created by make)
│   └── pru1_audio.out       # Compiled firmware
└── scripts/                 # Go utility application
    ├── main.go
    ├── go.mod
    └── cmd/
        ├── root.go
        └── test.go          # Test command
```

## Memory Layout

### PRU1 Local DRAM (8KB at 0x00002000)
- **Sample Buffer 0**: 0x0000-0x07FF (2KB)
- **Sample Buffer 1**: 0x0800-0x0FFF (2KB)
- **FFT Working Memory**: 0x1000-0x1FFF (4KB)

### Shared Memory (12KB at 0x00010000)
- **PRU0 (IR Detector)**: 0x0000-0x1FFF (8KB)
  - Ring buffer: 0x0000-0x07FF
  - Control block: 0x1000-0x101B
- **PRU1 (Audio Sampling)**: 0x2000-0x3FFF (8KB)
  - Audio control block: 0x2000-0x203F (64 bytes)
  - Sound profile ring buffer: 0x2100-0x24FF (~1KB)

## Current Status

### ✅ Completed (Stages 0-2)
- [x] Project plan written
- [x] PRU1 basic infrastructure created
- [x] Simple test firmware (toggle bit every second)
- [x] Cobra CLI utility with test command
- [x] Build and deployment system
- [x] Git commits with traceability

### 🔄 Next Steps (Stage 3)
- [ ] Deploy to BeagleBone
- [ ] Verify PRU1 operation with test command
- [ ] Validate shared memory access
- [ ] Confirm no conflicts with PRU0

### 📋 Remaining Stages
- Stage 4: Configure ADC in device tree
- Stage 5: Deploy ADC configuration
- Stage 6: Implement simple audio detection (10 Hz)
- Stage 7: Test audio detection
- Stage 8: Full 40 kHz sampling + FFT implementation
- Stage 9: Add sound profile streaming command
- Stage 10: Test complete system
- Stage 11: Integrate with Go API

## Development Workflow

### Making Changes
1. Edit source files (pru1_audio.c, utility commands, etc.)
2. Commit changes with descriptive messages
3. Deploy with `make deploy-all`
4. Test on BeagleBone
5. Iterate

### Adding New Utility Commands
1. Create new file in `scripts/cmd/` (e.g., `sample.go`)
2. Define cobra command and add to root command
3. Implement functionality
4. Build and test locally if possible
5. Deploy with `make deploy-util`

### Troubleshooting

#### PRU1 won't start
```bash
# Check PRU1 state
cat /sys/class/remoteproc/remoteproc2/state

# Check firmware exists
ls -la /lib/firmware/am335x-pru1-fw

# Check kernel logs
dmesg | tail -20
```

#### Utility can't access shared memory
```bash
# Must run as root
sudo ./audio-util test

# Check /dev/mem permissions
ls -la /dev/mem
```

#### Build failures
```bash
# Firmware build requires TI PRU compiler
which clpru

# If missing, install on BeagleBone:
sudo apt-get update
sudo apt-get install ti-pru-cgt-installer

# Utility build requires Go 1.23+
go version
```

## Hardware Configuration

- **PRU**: PRU1 (remoteproc2)
- **ADC Channel**: AIN1 (hardware cape requirement)
- **Target Sample Rate**: 40 kHz
- **PRU Clock**: 200 MHz

## References

- [PROJECT_PLAN.md](PROJECT_PLAN.md) - Detailed implementation plan
- [BeagleBone PRU Guide](https://beagleboard.org/pru)
- [TI PRU-ICSS Reference](https://www.ti.com/lit/ug/spruh73q/spruh73q.pdf)

## Support

If the session is interrupted, refer to PROJECT_PLAN.md and resume from the appropriate stage based on git commit history.

Current stage tracked in: `pru/audio/PROJECT_PLAN.md` and git commits
