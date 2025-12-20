#!/usr/bin/env python3
"""
Audio Parameter Optimization Script

Loads raw I2S samples captured from PRU and performs offline FFT analysis
with different parameter combinations to find optimal settings for LED control.

Usage:
    python optimize_audio_params.py baseline.bin music.bin

Requirements:
    pip install numpy scipy matplotlib pandas
"""

import numpy as np
import argparse
import struct
from pathlib import Path
from typing import Tuple, Dict, List
from dataclasses import dataclass
import json

# Try to import plotting libraries (optional)
try:
    import matplotlib.pyplot as plt
    import matplotlib
    matplotlib.use('Agg')  # Non-interactive backend
    PLOTTING_AVAILABLE = True
except ImportError:
    PLOTTING_AVAILABLE = False
    print("Warning: matplotlib not available, plots will be skipped")

try:
    import pandas as pd
    PANDAS_AVAILABLE = True
except ImportError:
    PANDAS_AVAILABLE = False
    print("Warning: pandas not available, CSV export will be skipped")


@dataclass
class AudioConfig:
    """Configuration for audio processing"""
    sample_rate: int = 32000  # Hz (I2S actual rate ~32 kHz)
    fft_size: int = 1024
    window_type: str = "hann"
    smoothing_alpha: float = 0.7
    bass_max: int = 150
    midlow_max: int = 1000
    midhigh_max: int = 2000

    def to_dict(self):
        return {
            'sample_rate': self.sample_rate,
            'fft_size': self.fft_size,
            'window_type': self.window_type,
            'smoothing_alpha': self.smoothing_alpha,
            'bass_max': self.bass_max,
            'midlow_max': self.midlow_max,
            'midhigh_max': self.midhigh_max,
        }


@dataclass
class AudioMetrics:
    """Metrics from audio analysis"""
    config: AudioConfig
    bass_snr_db: float
    midlow_snr_db: float
    midhigh_snr_db: float
    treble_snr_db: float
    overall_snr_db: float
    bass_cv: float
    midlow_cv: float
    midhigh_cv: float
    treble_cv: float

    def to_dict(self):
        d = self.config.to_dict()
        d.update({
            'bass_snr_db': float(self.bass_snr_db),
            'midlow_snr_db': float(self.midlow_snr_db),
            'midhigh_snr_db': float(self.midhigh_snr_db),
            'treble_snr_db': float(self.treble_snr_db),
            'overall_snr_db': float(self.overall_snr_db),
            'bass_cv': float(self.bass_cv),
            'midlow_cv': float(self.midlow_cv),
            'midhigh_cv': float(self.midhigh_cv),
            'treble_cv': float(self.treble_cv),
        })
        return d


def load_raw_samples(filepath: Path) -> np.ndarray:
    """
    Load raw I2S samples from binary file.

    Format: Little-endian int32 values (24-bit I2S data sign-extended to 32 bits)

    Args:
        filepath: Path to binary file

    Returns:
        numpy array of int32 samples
    """
    with open(filepath, 'rb') as f:
        data = f.read()

    # Unpack as little-endian int32
    sample_count = len(data) // 4
    samples = struct.unpack(f'<{sample_count}i', data)

    return np.array(samples, dtype=np.int32)


def apply_window(signal: np.ndarray, window_type: str) -> np.ndarray:
    """
    Apply windowing function to signal.

    Args:
        signal: Input signal
        window_type: Type of window ('hann', 'hamming', 'blackman', 'none')

    Returns:
        Windowed signal
    """
    if window_type == 'none':
        return signal
    elif window_type == 'hann':
        window = np.hanning(len(signal))
    elif window_type == 'hamming':
        window = np.hamming(len(signal))
    elif window_type == 'blackman':
        window = np.blackman(len(signal))
    else:
        raise ValueError(f"Unknown window type: {window_type}")

    return signal * window


def compute_fft_bands(samples: np.ndarray, config: AudioConfig) -> Dict[str, List[float]]:
    """
    Compute FFT and extract frequency band magnitudes.

    Args:
        samples: Raw ADC samples
        config: Audio configuration

    Returns:
        Dictionary with band names and lists of magnitude values
    """
    # Calculate number of complete buffers
    num_buffers = len(samples) // config.fft_size

    if num_buffers == 0:
        raise ValueError(f"Not enough samples for FFT. Need at least {config.fft_size}, got {len(samples)}")

    # Truncate to complete buffers
    samples = samples[:num_buffers * config.fft_size]

    # Reshape into buffers
    buffers = samples.reshape(num_buffers, config.fft_size)

    # Calculate frequency resolution
    freq_resolution = config.sample_rate / config.fft_size

    # Calculate bin boundaries
    bass_bin_max = int(config.bass_max / freq_resolution)
    midlow_bin_max = int(config.midlow_max / freq_resolution)
    midhigh_bin_max = int(config.midhigh_max / freq_resolution)
    nyquist_bin = config.fft_size // 2

    # Initialize band lists
    bass_values = []
    midlow_values = []
    midhigh_values = []
    treble_values = []

    # Process each buffer
    for buffer in buffers:
        # Remove DC offset
        buffer_float = buffer.astype(np.float32)
        buffer_float -= np.mean(buffer_float)

        # Apply window
        windowed = apply_window(buffer_float, config.window_type)

        # Compute FFT
        fft_result = np.fft.rfft(windowed)

        # Compute magnitude (skip DC bin 0)
        magnitudes = np.abs(fft_result[1:nyquist_bin])

        # Accumulate bands (using sum, as the PRU does)
        bass = np.sum(magnitudes[:bass_bin_max])
        midlow = np.sum(magnitudes[bass_bin_max:midlow_bin_max])
        midhigh = np.sum(magnitudes[midlow_bin_max:midhigh_bin_max])
        treble = np.sum(magnitudes[midhigh_bin_max:])

        bass_values.append(bass)
        midlow_values.append(midlow)
        midhigh_values.append(midhigh)
        treble_values.append(treble)

    # Apply temporal smoothing if configured
    if config.smoothing_alpha < 1.0:
        bass_values = apply_smoothing(bass_values, config.smoothing_alpha)
        midlow_values = apply_smoothing(midlow_values, config.smoothing_alpha)
        midhigh_values = apply_smoothing(midhigh_values, config.smoothing_alpha)
        treble_values = apply_smoothing(treble_values, config.smoothing_alpha)

    return {
        'bass': bass_values,
        'midlow': midlow_values,
        'midhigh': midhigh_values,
        'treble': treble_values,
    }


def apply_smoothing(values: List[float], alpha: float) -> List[float]:
    """
    Apply exponential moving average smoothing.

    Args:
        values: Input values
        alpha: Smoothing factor (0-1), where 1 = no smoothing

    Returns:
        Smoothed values
    """
    if alpha >= 1.0:
        return values

    smoothed = [values[0]]
    for i in range(1, len(values)):
        smoothed_val = alpha * values[i] + (1 - alpha) * smoothed[-1]
        smoothed.append(smoothed_val)

    return smoothed


def calculate_snr_db(music_values: List[float], baseline_values: List[float]) -> float:
    """
    Calculate Signal-to-Noise Ratio in dB.

    Args:
        music_values: Band values with music
        baseline_values: Band values with no music (noise floor)

    Returns:
        SNR in decibels
    """
    music_mean = np.mean(music_values)
    baseline_mean = np.mean(baseline_values)

    if baseline_mean <= 0:
        return float('inf')

    snr_linear = music_mean / baseline_mean
    snr_db = 10 * np.log10(snr_linear) if snr_linear > 0 else 0

    return snr_db


def calculate_cv(values: List[float]) -> float:
    """
    Calculate Coefficient of Variation (CV) as percentage.

    Args:
        values: Input values

    Returns:
        CV in percent
    """
    mean = np.mean(values)
    std = np.std(values)

    if mean == 0:
        return 0

    return (std / mean) * 100


def analyze_configuration(baseline_samples: np.ndarray,
                         music_samples: np.ndarray,
                         config: AudioConfig) -> AudioMetrics:
    """
    Analyze a specific audio configuration.

    Args:
        baseline_samples: Baseline (no music) samples
        music_samples: Music samples
        config: Audio configuration to test

    Returns:
        AudioMetrics with SNR and CV values
    """
    # Compute bands for baseline and music
    baseline_bands = compute_fft_bands(baseline_samples, config)
    music_bands = compute_fft_bands(music_samples, config)

    # Calculate SNR for each band
    bass_snr = calculate_snr_db(music_bands['bass'], baseline_bands['bass'])
    midlow_snr = calculate_snr_db(music_bands['midlow'], baseline_bands['midlow'])
    midhigh_snr = calculate_snr_db(music_bands['midhigh'], baseline_bands['midhigh'])
    treble_snr = calculate_snr_db(music_bands['treble'], baseline_bands['treble'])

    # Calculate overall SNR (average)
    overall_snr = np.mean([bass_snr, midlow_snr, midhigh_snr, treble_snr])

    # Calculate CV for music (to measure dynamic range)
    bass_cv = calculate_cv(music_bands['bass'])
    midlow_cv = calculate_cv(music_bands['midlow'])
    midhigh_cv = calculate_cv(music_bands['midhigh'])
    treble_cv = calculate_cv(music_bands['treble'])

    return AudioMetrics(
        config=config,
        bass_snr_db=bass_snr,
        midlow_snr_db=midlow_snr,
        midhigh_snr_db=midhigh_snr,
        treble_snr_db=treble_snr,
        overall_snr_db=overall_snr,
        bass_cv=bass_cv,
        midlow_cv=midlow_cv,
        midhigh_cv=midhigh_cv,
        treble_cv=treble_cv,
    )


def parameter_sweep(baseline_samples: np.ndarray,
                    music_samples: np.ndarray,
                    output_dir: Path) -> List[AudioMetrics]:
    """
    Perform parameter sweep to find optimal configuration.

    Tests different combinations of:
    - Window types: hann, hamming, blackman, none
    - Smoothing alpha: 0.3, 0.5, 0.7, 0.9, 1.0
    - Frequency boundaries (limited variations)

    Args:
        baseline_samples: Baseline samples
        music_samples: Music samples
        output_dir: Directory to save results

    Returns:
        List of AudioMetrics for all configurations tested
    """
    results = []

    # Parameter ranges to test
    window_types = ['hann', 'hamming', 'blackman', 'none']
    smoothing_alphas = [0.3, 0.5, 0.7, 0.9, 1.0]

    # Frequency boundary variations (Hz)
    freq_configs = [
        {'bass_max': 150, 'midlow_max': 1000, 'midhigh_max': 2000},  # Current
        {'bass_max': 200, 'midlow_max': 1500, 'midhigh_max': 4000},  # Wider bands
        {'bass_max': 100, 'midlow_max': 500, 'midhigh_max': 2000},   # Narrower bass
        {'bass_max': 250, 'midlow_max': 2000, 'midhigh_max': 8000},  # Extended treble cutoff
    ]

    total_configs = len(window_types) * len(smoothing_alphas) * len(freq_configs)
    print(f"\nTesting {total_configs} configurations...")
    print(f"  Window types: {window_types}")
    print(f"  Smoothing alphas: {smoothing_alphas}")
    print(f"  Frequency configs: {len(freq_configs)} variations")
    print()

    config_num = 0
    for window in window_types:
        for alpha in smoothing_alphas:
            for freq_cfg in freq_configs:
                config_num += 1

                config = AudioConfig(
                    window_type=window,
                    smoothing_alpha=alpha,
                    **freq_cfg
                )

                try:
                    metrics = analyze_configuration(baseline_samples, music_samples, config)
                    results.append(metrics)

                    if config_num % 10 == 0:
                        print(f"Progress: {config_num}/{total_configs} configurations tested...")

                except Exception as e:
                    print(f"Error testing config {config_num}: {e}")
                    continue

    print(f"\nCompleted {len(results)} configurations successfully\n")

    return results


def generate_report(results: List[AudioMetrics], output_dir: Path):
    """
    Generate analysis report with recommendations.

    Args:
        results: List of AudioMetrics from parameter sweep
        output_dir: Directory to save report
    """
    output_dir.mkdir(parents=True, exist_ok=True)

    # Sort by overall SNR (descending)
    results_by_snr = sorted(results, key=lambda x: x.overall_snr_db, reverse=True)

    # Sort by CV (descending) - for dynamic range
    results_by_cv = sorted(results, key=lambda x: (x.bass_cv + x.midlow_cv + x.midhigh_cv + x.treble_cv) / 4, reverse=True)

    # Generate text report
    report_path = output_dir / 'optimization_report.txt'
    with open(report_path, 'w') as f:
        f.write("=" * 80 + "\n")
        f.write("AUDIO PARAMETER OPTIMIZATION REPORT\n")
        f.write("=" * 80 + "\n\n")

        f.write(f"Total configurations tested: {len(results)}\n\n")

        f.write("=" * 80 + "\n")
        f.write("TOP 5 CONFIGURATIONS BY SIGNAL-TO-NOISE RATIO\n")
        f.write("=" * 80 + "\n\n")

        for i, metrics in enumerate(results_by_snr[:5], 1):
            f.write(f"#{i} - Overall SNR: {metrics.overall_snr_db:.2f} dB\n")
            f.write(f"  Window: {metrics.config.window_type}\n")
            f.write(f"  Smoothing Alpha: {metrics.config.smoothing_alpha}\n")
            f.write(f"  Bass Max: {metrics.config.bass_max} Hz\n")
            f.write(f"  Mid-Low Max: {metrics.config.midlow_max} Hz\n")
            f.write(f"  Mid-High Max: {metrics.config.midhigh_max} Hz\n")
            f.write(f"  Band SNRs: Bass={metrics.bass_snr_db:.2f}dB, MidLow={metrics.midlow_snr_db:.2f}dB, " +
                   f"MidHigh={metrics.midhigh_snr_db:.2f}dB, Treble={metrics.treble_snr_db:.2f}dB\n")
            f.write(f"  Band CVs: Bass={metrics.bass_cv:.1f}%, MidLow={metrics.midlow_cv:.1f}%, " +
                   f"MidHigh={metrics.midhigh_cv:.1f}%, Treble={metrics.treble_cv:.1f}%\n")
            f.write("\n")

        f.write("=" * 80 + "\n")
        f.write("TOP 5 CONFIGURATIONS BY DYNAMIC RANGE (CV)\n")
        f.write("=" * 80 + "\n\n")

        for i, metrics in enumerate(results_by_cv[:5], 1):
            avg_cv = (metrics.bass_cv + metrics.midlow_cv + metrics.midhigh_cv + metrics.treble_cv) / 4
            f.write(f"#{i} - Average CV: {avg_cv:.1f}%\n")
            f.write(f"  Window: {metrics.config.window_type}\n")
            f.write(f"  Smoothing Alpha: {metrics.config.smoothing_alpha}\n")
            f.write(f"  Overall SNR: {metrics.overall_snr_db:.2f} dB\n")
            f.write(f"  Band CVs: Bass={metrics.bass_cv:.1f}%, MidLow={metrics.midlow_cv:.1f}%, " +
                   f"MidHigh={metrics.midhigh_cv:.1f}%, Treble={metrics.treble_cv:.1f}%\n")
            f.write("\n")

        f.write("=" * 80 + "\n")
        f.write("RECOMMENDED CONFIGURATION\n")
        f.write("=" * 80 + "\n\n")

        # Find best balance: high SNR and high CV
        best = max(results, key=lambda x: x.overall_snr_db * ((x.bass_cv + x.midlow_cv + x.midhigh_cv + x.treble_cv) / 400))

        f.write("Best balance of signal quality and dynamic range:\n\n")
        f.write(f"  Window Type: {best.config.window_type}\n")
        f.write(f"  Smoothing Alpha: {best.config.smoothing_alpha}\n")
        f.write(f"  Bass Max: {best.config.bass_max} Hz\n")
        f.write(f"  Mid-Low Max: {best.config.midlow_max} Hz\n")
        f.write(f"  Mid-High Max: {best.config.midhigh_max} Hz\n\n")
        f.write(f"  Overall SNR: {best.overall_snr_db:.2f} dB\n")
        f.write(f"  Band SNRs: Bass={best.bass_snr_db:.2f}dB, MidLow={best.midlow_snr_db:.2f}dB, " +
               f"MidHigh={best.midhigh_snr_db:.2f}dB, Treble={best.treble_snr_db:.2f}dB\n")
        f.write(f"  Band CVs: Bass={best.bass_cv:.1f}%, MidLow={best.midlow_cv:.1f}%, " +
               f"MidHigh={best.midhigh_cv:.1f}%, Treble={best.treble_cv:.1f}%\n\n")

        f.write("To apply these settings to PRU firmware:\n")
        f.write(f"  1. Update pru/audio/pru1_audio_i2s.c:\n")
        f.write(f"     - BASS_MAX_HZ = {best.config.bass_max}\n")
        f.write(f"     - MIDLOW_MAX_HZ = {best.config.midlow_max}\n")
        f.write(f"     - MIDHIGH_MAX_HZ = {best.config.midhigh_max}\n")
        f.write(f"     - smoothing_alpha_x1000 = {int(best.config.smoothing_alpha * 1000)}\n")
        f.write(f"     - Window function: {best.config.window_type}\n")
        f.write(f"  2. Rebuild and deploy: cd pru/audio && make i2s-deploy-all\n\n")

    print(f"Report saved to: {report_path}")

    # Export to JSON
    json_path = output_dir / 'optimization_results.json'
    with open(json_path, 'w') as f:
        json.dump({
            'recommended': best.to_dict(),
            'top_snr': [m.to_dict() for m in results_by_snr[:5]],
            'top_cv': [m.to_dict() for m in results_by_cv[:5]],
            'all_results': [m.to_dict() for m in results],
        }, f, indent=2)
    print(f"JSON results saved to: {json_path}")

    # Export to CSV if pandas available
    if PANDAS_AVAILABLE:
        csv_path = output_dir / 'optimization_results.csv'
        df = pd.DataFrame([m.to_dict() for m in results])
        df.to_csv(csv_path, index=False)
        print(f"CSV results saved to: {csv_path}")

    # Generate plots if matplotlib available
    if PLOTTING_AVAILABLE:
        generate_plots(results, output_dir)


def generate_plots(results: List[AudioMetrics], output_dir: Path):
    """
    Generate visualization plots.

    Args:
        results: List of AudioMetrics
        output_dir: Directory to save plots
    """
    # Plot 1: SNR by window type and smoothing
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    fig.suptitle('SNR by Configuration Parameters', fontsize=16, fontweight='bold')

    window_types = sorted(set(m.config.window_type for m in results))
    smoothing_alphas = sorted(set(m.config.smoothing_alpha for m in results))

    for window in window_types:
        window_results = [m for m in results if m.config.window_type == window]
        snrs_by_alpha = {alpha: [] for alpha in smoothing_alphas}

        for alpha in smoothing_alphas:
            alpha_results = [m for m in window_results if m.config.smoothing_alpha == alpha]
            if alpha_results:
                snrs_by_alpha[alpha] = [m.overall_snr_db for m in alpha_results]

        axes[0, 0].plot(smoothing_alphas, [np.mean(snrs_by_alpha[a]) if snrs_by_alpha[a] else 0 for a in smoothing_alphas],
                       marker='o', label=window)

    axes[0, 0].set_xlabel('Smoothing Alpha')
    axes[0, 0].set_ylabel('Overall SNR (dB)')
    axes[0, 0].set_title('SNR vs Smoothing Alpha by Window Type')
    axes[0, 0].legend()
    axes[0, 0].grid(True, alpha=0.3)

    # Plot 2: SNR by frequency band
    bands = ['bass_snr_db', 'midlow_snr_db', 'midhigh_snr_db', 'treble_snr_db']
    band_names = ['Bass', 'Mid-Low', 'Mid-High', 'Treble']

    for window in window_types:
        window_results = [m for m in results if m.config.window_type == window]
        band_snrs = [np.mean([getattr(m, band) for m in window_results]) for band in bands]
        axes[0, 1].plot(band_names, band_snrs, marker='o', label=window)

    axes[0, 1].set_xlabel('Frequency Band')
    axes[0, 1].set_ylabel('SNR (dB)')
    axes[0, 1].set_title('SNR by Frequency Band and Window Type')
    axes[0, 1].legend()
    axes[0, 1].grid(True, alpha=0.3)

    # Plot 3: CV by window type
    for window in window_types:
        window_results = [m for m in results if m.config.window_type == window]
        cvs_by_alpha = {alpha: [] for alpha in smoothing_alphas}

        for alpha in smoothing_alphas:
            alpha_results = [m for m in window_results if m.config.smoothing_alpha == alpha]
            if alpha_results:
                avg_cvs = [(m.bass_cv + m.midlow_cv + m.midhigh_cv + m.treble_cv) / 4 for m in alpha_results]
                cvs_by_alpha[alpha] = avg_cvs

        axes[1, 0].plot(smoothing_alphas, [np.mean(cvs_by_alpha[a]) if cvs_by_alpha[a] else 0 for a in smoothing_alphas],
                       marker='o', label=window)

    axes[1, 0].set_xlabel('Smoothing Alpha')
    axes[1, 0].set_ylabel('Average CV (%)')
    axes[1, 0].set_title('Dynamic Range (CV) vs Smoothing Alpha')
    axes[1, 0].legend()
    axes[1, 0].grid(True, alpha=0.3)

    # Plot 4: SNR vs CV scatter
    for window in window_types:
        window_results = [m for m in results if m.config.window_type == window]
        snrs = [m.overall_snr_db for m in window_results]
        cvs = [(m.bass_cv + m.midlow_cv + m.midhigh_cv + m.treble_cv) / 4 for m in window_results]
        axes[1, 1].scatter(snrs, cvs, label=window, alpha=0.6, s=50)

    axes[1, 1].set_xlabel('Overall SNR (dB)')
    axes[1, 1].set_ylabel('Average CV (%)')
    axes[1, 1].set_title('SNR vs Dynamic Range Trade-off')
    axes[1, 1].legend()
    axes[1, 1].grid(True, alpha=0.3)

    plt.tight_layout()
    plot_path = output_dir / 'optimization_analysis.png'
    plt.savefig(plot_path, dpi=150)
    plt.close()
    print(f"Plots saved to: {plot_path}")


def main():
    parser = argparse.ArgumentParser(
        description='Optimize audio processing parameters for LED control',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='''
Examples:
  # Deploy raw I2S capture firmware first:
  cd pru/audio && make i2s-raw-deploy-all

  # Capture samples on BeagleBone:
  ssh bbb1.wegman
  cd ~/led-sound-light-control
  sudo ./raw_i2s_capture 10 baseline.bin    # 10 seconds of silence
  sudo ./raw_i2s_capture 30 music.bin       # 30 seconds of music

  # Copy back and optimize:
  scp bbb1.wegman:~/led-sound-light-control/*.bin .
  python optimize_audio_params.py baseline.bin music.bin

  # Or specify output directory:
  python optimize_audio_params.py baseline.bin music.bin -o results/
        '''
    )

    parser.add_argument('baseline_file', type=Path,
                       help='Binary file with baseline (no music) samples')
    parser.add_argument('music_file', type=Path,
                       help='Binary file with music samples')
    parser.add_argument('-o', '--output', type=Path, default=Path('optimization_results'),
                       help='Output directory for results (default: optimization_results/)')
    parser.add_argument('--quick', action='store_true',
                       help='Quick mode: test fewer configurations')

    args = parser.parse_args()

    print("=" * 80)
    print("AUDIO PARAMETER OPTIMIZATION")
    print("=" * 80)
    print()

    # Load samples
    sample_rate = 32000  # I2S actual rate
    print(f"Loading baseline samples from: {args.baseline_file}")
    baseline_samples = load_raw_samples(args.baseline_file)
    print(f"  Loaded {len(baseline_samples)} samples ({len(baseline_samples) / sample_rate:.2f} seconds)")

    print(f"\nLoading music samples from: {args.music_file}")
    music_samples = load_raw_samples(args.music_file)
    print(f"  Loaded {len(music_samples)} samples ({len(music_samples) / sample_rate:.2f} seconds)")

    # Run parameter sweep
    results = parameter_sweep(baseline_samples, music_samples, args.output)

    # Generate report
    print("\nGenerating report...")
    generate_report(results, args.output)

    print("\n" + "=" * 80)
    print("OPTIMIZATION COMPLETE")
    print("=" * 80)
    print(f"\nResults saved to: {args.output}/")
    print("\nNext steps:")
    print("  1. Review optimization_report.txt for recommended settings")
    print("  2. Update PRU firmware with optimal parameters")
    print("  3. Rebuild and deploy: cd pru/audio && make i2s-deploy-all")
    print()


if __name__ == '__main__':
    main()
