# LED Sound Light Control

Control LED lights based on real-time audio frequency analysis using BeagleBone Black's PRU for high-speed ADC sampling.

## Features

- **48 kHz ADC Sampling**: Uses PRU (Programmable Real-time Unit) for deterministic real-time sampling
- **Real-time FFT Analysis**: Frequency domain analysis of audio signals
- **High Performance**: PRU-based sampling eliminates Linux kernel scheduling jitter
- **Low Latency**: Direct memory-mapped communication between PRU and Go application

## Hardware Requirements

- BeagleBone Black
- Audio/analog signal source
- Connection to P9_40 (AIN1) and P9_34 (AGND)
- **Input voltage range: 0-1.8V** (BeagleBone ADC limit)

## Quick Start

### 1. Initial Setup on BeagleBone Black

```bash
cd scripts
sudo ./setup_beaglebone.sh
sudo reboot
```

### 2. Verify PRU Status

```bash
./scripts/check_pru_status.sh
```

### 3. Build and Test

```bash
# Build the application
make build

# Test PRU sampling (run as root)
sudo ./bin/led-sound-light-control pru-test

# Run the full application
sudo ./bin/led-sound-light-control
```

## Project Structure

```
.
├── pru/                    # PRU firmware for ADC sampling
│   ├── pru0_adc_sampler.c  # PRU C firmware (48 kHz sampling)
│   ├── Makefile            # PRU build system
│   ├── PRU-ADC-00A0.dts    # Device tree overlay
│   └── README.md           # Detailed PRU documentation
├── sound/                  # Audio processing package
│   ├── pru_reader.go       # Memory-mapped PRU interface
│   ├── reader.go           # Sample reader with PRU support
│   ├── processor.go        # FFT frequency analysis
│   └── manager.go          # Audio processing manager
├── scripts/                # Deployment and setup scripts
│   ├── setup_beaglebone.sh # One-time BeagleBone setup
│   ├── deploy_pru.sh       # Quick PRU firmware deployment
│   └── check_pru_status.sh # PRU status and debugging
└── cmd/                    # CLI commands
    └── pru-test.go         # PRU sampling test utility
```

## How It Works

1. **PRU Firmware**: Samples AIN1 at 48 kHz and writes to shared memory ring buffer
2. **Go Reader**: Memory-maps PRU shared memory and reads samples in real-time
3. **Processor**: Performs FFT analysis on sample buffers
4. **Manager**: Coordinates the pipeline and exposes results

## Configuration

### Sampling Rate

Current: **48 kHz** (configured in `sound/constants.go`)

To change, edit both:
- `sound/constants.go`: Update `SamplingRate` constant
- `pru/pru0_adc_sampler.c`: Update `CYCLES_PER_SAMPLE` calculation
- Rebuild both: `make build && cd pru && make && sudo ../scripts/deploy_pru.sh`

### Buffer Size

Adjust `BufferSize` in `sound/constants.go` based on your latency requirements.

## Documentation

- **[PRU Setup Guide](pru/PRU_SETUP.md)**: Detailed PRU installation and configuration
- **[PRU README](pru/README.md)**: PRU firmware architecture and development

## Troubleshooting

### Permission Denied
Run as root: `sudo ./bin/led-sound-light-control`

### PRU Not Running
```bash
./scripts/check_pru_status.sh
# If offline:
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state
```

### Buffer Overruns
Reduce sleep time in reader or increase buffer size. Monitor with `pru-test` command.

## Performance

- **Sampling Rate**: 48,000 samples/second
- **Timing Precision**: Microsecond-level (PRU deterministic execution)
- **Latency**: ~10-20ms (depends on buffer size)
- **CPU Usage**: Low (PRU handles sampling independently)

## License

See LICENSE file.
