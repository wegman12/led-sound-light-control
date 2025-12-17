#!/usr/bin/env python3
"""
Analyze only valid ADC samples (filter out zeros and out-of-range values)
"""

import numpy as np
import struct
from pathlib import Path

def analyze_valid_samples(filepath: Path):
    """Analyze only valid ADC samples"""

    print(f"\n{'='*70}")
    print(f"VALID SAMPLE ANALYSIS: {filepath.name}")
    print(f"{'='*70}\n")

    # Load samples
    with open(filepath, 'rb') as f:
        data = f.read()

    sample_count = len(data) // 2
    samples = struct.unpack(f'<{sample_count}H', data)
    samples = np.array(samples, dtype=np.uint16)

    # Filter to valid ADC range (1-4095, excluding 0 and out-of-range)
    valid_samples = samples[(samples > 0) & (samples <= 4095)]

    print(f"Total samples: {len(samples):,}")
    print(f"Valid samples (1-4095): {len(valid_samples):,} ({len(valid_samples)/len(samples)*100:.1f}%)")
    print(f"Zeros: {np.sum(samples == 0):,} ({np.sum(samples == 0)/len(samples)*100:.1f}%)")
    print(f"Out of range (>4095): {np.sum(samples > 4095):,} ({np.sum(samples > 4095)/len(samples)*100:.1f}%)\n")

    if len(valid_samples) == 0:
        print("No valid samples found!")
        return

    stats = {
        'min': int(np.min(valid_samples)),
        'max': int(np.max(valid_samples)),
        'mean': float(np.mean(valid_samples)),
        'std': float(np.std(valid_samples)),
        'median': float(np.median(valid_samples)),
        'range': int(np.max(valid_samples) - np.min(valid_samples)),
    }

    print(f"Valid Sample Statistics:")
    print(f"  Min value:         {stats['min']} / 4095 ({stats['min']/4095*1.8:.3f}V)")
    print(f"  Max value:         {stats['max']} / 4095 ({stats['max']/4095*1.8:.3f}V)")
    print(f"  Mean (DC bias):    {stats['mean']:.1f} / 4095 ({stats['mean']/4095*1.8:.3f}V)")
    print(f"  Expected DC bias:  2048 / 4095 (0.900V)")
    print(f"  DC bias error:     {stats['mean']-2048:+.1f} counts ({(stats['mean']-2048)/4095*1.8:+.3f}V)")
    print(f"  Std deviation:     {stats['std']:.1f}")
    print(f"  Median:            {stats['median']:.1f}")
    print(f"  Dynamic range:     {stats['range']} counts ({stats['range']/4095*1.8:.3f}V)")

    # Clipping analysis
    samples_at_4095 = np.sum(valid_samples == 4095)
    samples_above_3995 = np.sum(valid_samples > 3995)

    print(f"\nClipping Analysis (valid samples only):")
    print(f"  Samples at 4095 (max):     {samples_at_4095:,} ({samples_at_4095/len(valid_samples)*100:.2f}%)")
    print(f"  Samples > 3995 (near max): {samples_above_3995:,} ({samples_above_3995/len(valid_samples)*100:.2f}%)")

    print(f"\n{'='*70}")
    print("DIAGNOSIS:")
    print(f"{'='*70}")

    issues = []

    if samples_at_4095 > len(valid_samples) * 0.01:
        issues.append(f"⚠️  Frequent clipping at 4095 ({samples_at_4095/len(valid_samples)*100:.2f}% of samples)")
        issues.append("   -> Input voltage occasionally exceeds 1.8V")
        issues.append("   -> Reduce microphone gain or input level")
    elif stats['max'] == 4095:
        issues.append("⚠️  Peak value at absolute maximum (4095)")
        issues.append("   -> Signal is clipping occasionally")

    if abs(stats['mean'] - 2048) > 500:
        issues.append(f"⚠️  DC bias off center: {stats['mean']:.0f} (expected ~2048)")
        issues.append(f"   -> Measured: {stats['mean']/4095*1.8:.3f}V, Expected: ~0.9V")
        if stats['mean'] < 1500:
            issues.append("   -> DC bias too low - check power supply and biasing circuit")

    if stats['range'] < 200:
        issues.append(f"⚠️  Very narrow dynamic range: {stats['range']} counts")
        issues.append("   -> Signal amplitude is very low")
        issues.append("   -> May need more gain or different mic positioning")

    if stats['std'] < 50:
        issues.append(f"⚠️  Very low variation: std = {stats['std']:.1f}")
        issues.append("   -> Signal is mostly flat/DC")

    if not issues:
        print("✅ No major issues detected")
        print("   Valid samples appear to be within acceptable range")
        print(f"   DC bias: {stats['mean']:.0f} (~{stats['mean']/4095*1.8:.2f}V)")
        print(f"   Range: {stats['min']} - {stats['max']}")
    else:
        for issue in issues:
            print(issue)

    print(f"{'='*70}\n")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python analyze_valid_only.py <file.bin> [<file2.bin> ...]")
        sys.exit(1)

    for filepath in sys.argv[1:]:
        path = Path(filepath)
        if not path.exists():
            print(f"ERROR: File not found: {filepath}")
            continue

        analyze_valid_samples(path)

if __name__ == '__main__':
    main()
