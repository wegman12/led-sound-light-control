# LED Sound Light Control

Control LED lights based on real-time audio frequency analysis using BeagleBone Black's high-speed ADC sampling.

## Features

- **48 kHz ADC Sampling**: High-speed Linux IIO sampling (~200 kHz) with intelligent decimation to 48 kHz
- **Real-time FFT Analysis**: Frequency domain analysis of audio signals
- **Anti-Aliasing Filter**: Software filtering prevents aliasing during decimation
- **Low Latency**: Direct IIO buffer access with minimal overhead

## Hardware Requirements

- BeagleBone Black
- Audio/analog signal source
- Connection to P9_40 (AIN1) and P9_34 (AGND)
- **Input voltage range: 0-1.8V** (BeagleBone ADC limit)

## Quick Start

### Build and Test

```bash
# Build the application
make build

# Test sound sampling (run as root for IIO access)
sudo ./bin/led-sound-light-control test-sound

# Test IIO sampling directly
sudo ./bin/led-sound-light-control iio-test
```

## Project Structure

```
.
├── sound/                  # Audio processing package
│   ├── iio_fast_reader.go  # High-speed IIO buffered reader
│   ├── decimator.go        # Sample rate decimation with anti-aliasing
│   ├── reader.go           # Sample reader with IIO integration
│   ├── processor.go        # FFT frequency analysis
│   └── manager.go          # Audio processing manager
├── cmd/                    # CLI commands
│   ├── sound-tester.go     # Sound sampling test with CSV export
│   └── iio-test.go         # IIO sampling test utility
└── utilities/              # Helper functions
```

## How It Works

1. **IIO Reader**: Samples AIN1 at ~200 kHz using Linux IIO buffered mode
2. **Decimator**: Applies anti-aliasing filter and decimates to 48 kHz
3. **Processor**: Performs FFT analysis on 48 kHz sample buffers
4. **Manager**: Coordinates the pipeline and exposes results

## Configuration

### Sampling Rate

Current: **48 kHz** (configured in `sound/constants.go`)

The system samples at ~200 kHz from IIO and decimates to 48 kHz. To change the target rate:
- Edit `sound/constants.go`: Update `SamplingRate` constant
- Rebuild: `make build`

The decimator will automatically adjust the decimation ratio.

### Buffer Size

Adjust `BufferSize` in `sound/constants.go` based on your latency requirements.

## Troubleshooting

### Permission Denied
Run as root for IIO access: `sudo ./bin/led-sound-light-control test-sound`

### No ADC Device
Check if IIO device exists:
```bash
ls -la /sys/bus/iio/devices/iio:device0/
```

### Low Sample Rate
If you're not getting the expected sample rate, check:
```bash
# View actual IIO sample rate
cat /sys/bus/iio/devices/iio:device0/sampling_frequency 2>/dev/null
```

## Performance

- **Input Sampling Rate**: ~200,000 samples/second (IIO buffered mode)
- **Output Sampling Rate**: 48,000 samples/second (after decimation)
- **Timing Precision**: Good (hardware-triggered IIO with software decimation)
- **Latency**: ~10-20ms (depends on buffer size)
- **CPU Usage**: Low (IIO uses DMA, minimal CPU for decimation)

## License

See LICENSE file.
