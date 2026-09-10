# PRU Audio Sampler Analysis

This directory contains analysis tools for evaluating PRU audio sampling data to optimize LED control parameters.

## Setup

This project uses Poetry for dependency management.

### Prerequisites

- Python 3.14 or higher
- Poetry

### Installation

```bash
# Install dependencies
poetry install

# Activate the virtual environment
poetry shell
```

## Usage

### Launch Jupyter Notebook

```bash
poetry run jupyter notebook
```

Then open `audio_analysis.ipynb` in your browser.

### Run Analysis

The notebook includes:

1. **Data Loading**: Loads baseline, Crab Rave, and Techno audio samples
2. **Data Quality Assessment**: Validates sampling quality and FFT performance
3. **Frequency Band Distribution**: Compares band responses across datasets
4. **Signal-to-Noise Ratio**: Measures music signal vs baseline noise
5. **Time Series Visualization**: Plots frequency bands over time
6. **Dynamic Range Analysis**: Evaluates variability for LED responsiveness
7. **Correlation Analysis**: Checks independence of frequency bands
8. **Statistical Comparison**: Tests significance of music vs baseline
9. **Balance Analysis**: Examines relative frequency band contributions
10. **LED Tuning Recommendations**: Provides specific tuning parameters

### Output Files

After running the analysis, the following files are generated:

- `analysis_summary.csv`: Statistical summary for all datasets and bands
- `snr_analysis.csv`: Signal-to-Noise ratio results

## Dataset Files

- `baseline.csv`: Background noise (no music playing)
- `crab_rave.csv`: Crab Rave song recording
- `techno.csv`: Techno song recording

### CSV Format

Each CSV file contains the following columns:

- `Sample`: Sample number
- `Timestamp (s)`: Time in seconds
- `FFT Count`: Total FFT operations performed
- `FFT Rate (Hz)`: FFTs per second
- `FFT Time (ms)`: FFT processing time
- `Bass Sum`: Bass frequency band total magnitude
- `Mid-Low Sum`: Mid-low frequency band total magnitude
- `Mid-High Sum`: Mid-high frequency band total magnitude
- `Treble Sum`: Treble frequency band total magnitude
- `Bass Avg`: Bass average magnitude per bin
- `Mid-Low Avg`: Mid-low average magnitude per bin
- `Mid-High Avg`: Mid-high average magnitude per bin
- `Treble Avg`: Treble average magnitude per bin

## Analysis Results

The notebook provides:

### Key Metrics

- **SNR (Signal-to-Noise Ratio)**: Measures how well music stands out from baseline
- **Dynamic Range**: Shows variability for responsive LED effects
- **Coefficient of Variation**: Indicates how much values change over time
- **Statistical Significance**: Confirms music differs from baseline noise

### Recommendations

The analysis generates specific recommendations for:

1. **Scaling Factors**: How to normalize values for 0-255 LED range
2. **Noise Thresholds**: Minimum values to filter out background noise
3. **LED Mappings**: Which frequency bands to use for different LED effects
4. **Tuning Parameters**: Optimal settings for frequency band boundaries

## Next Steps

After reviewing the analysis:

1. Implement recommended scaling factors in LED controller
2. Add noise filtering thresholds
3. Test with additional music genres
4. Fine-tune frequency band boundaries if needed
5. Add smoothing for stable LED transitions

---

# Raw Sample Capture and Parameter Optimization

## Overview

If the processed FFT data shows no difference between baseline and music, you can use the raw sample capture system to perform offline parameter optimization. This allows testing hundreds of configurations without recompiling/deploying firmware.

## Workflow

### 1. Deploy Raw Capture Firmware

The raw capture firmware samples ADC at 40 kHz without any processing:

```bash
cd pru/audio
make raw-deploy-all
```

### 2. Capture Baseline Samples

Capture audio with **no music** (room noise/silence):

```bash
record-pru-raw -o baseline.bin -t 60s
```

### 3. Capture Music Samples

Play music at **loud volume** and capture:

```bash
record-pru-raw -o music.bin -t 60s
```

**Tips:**
- Use music with strong bass and varied dynamics
- Play at typical volume level for LED use
- Ensure audio input is connected properly

### 4. Copy Samples to Development Machine

```bash
scp bbb1.wegman:~/baseline.bin analysis/audio/
scp bbb1.wegman:~/music.bin analysis/audio/
```

### 5. Run Parameter Optimization

```bash
python optimize_audio_params.py baseline.bin music.bin
```

This tests:
- Window functions: Hann, Hamming, Blackman, none
- Smoothing alpha: 0.3, 0.5, 0.7, 0.9, 1.0
- Frequency band boundaries: 4 configurations

**Requirements:**
```bash
pip install numpy scipy matplotlib pandas
```

### 6. Review Results

Check `optimization_results/`:
- `optimization_report.txt` - Recommendations
- `optimization_results.json` - Machine-readable data
- `optimization_results.csv` - Spreadsheet data
- `optimization_analysis.png` - Visualization

### 7. Apply Recommended Settings

Update `pru/audio/pru1_audio.c` with optimal values:

```c
#define BASS_MAX_HZ         200   // From optimization
#define MIDLOW_MAX_HZ       1500
#define MIDHIGH_MAX_HZ      4000

// In main():
ctrl->smoothing_alpha_x1000 = 500; // alpha = 0.5
```

### 8. Deploy Updated Firmware

```bash
cd pru/audio
make deploy-all  # Switches back to audio processing firmware
```

## Key Metrics

- **SNR (Signal-to-Noise Ratio)**:
  - \>20 dB = Excellent
  - 10-20 dB = Good
  - <10 dB = Poor (music doesn't stand out)

- **CV (Coefficient of Variation)**:
  - \>50% = High variability (dynamic LEDs)
  - 20-50% = Moderate
  - <20% = Low (static appearance)

## Troubleshooting

### Raw capture firmware not detected
```bash
cd pru/audio && make raw-deploy-all
cat /sys/class/remoteproc/remoteproc2/state  # Should show "running"
```

### Buffer overruns
Decrease sampling interval:
```bash
record-pru-raw -o samples.bin -i 10ms
```

### Not enough samples
Capture longer or check PRU status:
```bash
dmesg | grep pru
```

## Files

- `optimize_audio_params.py` - Parameter optimization script
- `pru/audio/pru1_raw_capture.c` - Raw capture PRU firmware
- `api/audio/pru_raw_sampler.go` - Raw sample reader
- `api/audio/cmd.go` - CLI commands
