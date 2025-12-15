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
