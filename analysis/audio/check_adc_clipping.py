#!/usr/bin/env python3
"""
Check captured ADC samples for voltage clipping and range issues
"""

import numpy as np
import struct
from pathlib import Path

def analyze_adc_samples(filepath: Path) -> dict:
    """Analyze raw ADC samples for clipping and voltage range issues"""

    # Load samples
    with open(filepath, 'rb') as f:
        data = f.read()

    sample_count = len(data) // 2
    samples = struct.unpack(f'<{sample_count}H', data)
    samples = np.array(samples, dtype=np.uint16)

    # ADC is 12-bit: range 0-4095
    # Expected for proper operation: mostly in middle range (1000-3000)
    # Clipping indicators:
    #   - Many samples at 0 or 4095
    #   - Very narrow range
    #   - DC bias way off from 2048 (middle)

    stats = {
        'filename': filepath.name,
        'total_samples': len(samples),
        'min': int(np.min(samples)),
        'max': int(np.max(samples)),
        'mean': float(np.mean(samples)),
        'std': float(np.std(samples)),
        'median': float(np.median(samples)),
        'range': int(np.max(samples) - np.min(samples)),
    }

    # Count samples at extremes
    stats['samples_at_0'] = int(np.sum(samples == 0))
    stats['samples_at_4095'] = int(np.sum(samples == 4095))
    stats['samples_below_100'] = int(np.sum(samples < 100))
    stats['samples_above_3995'] = int(np.sum(samples > 3995))

    # Calculate percentages
    stats['pct_at_0'] = (stats['samples_at_0'] / stats['total_samples']) * 100
    stats['pct_at_4095'] = (stats['samples_at_4095'] / stats['total_samples']) * 100
    stats['pct_clipping'] = ((stats['samples_below_100'] + stats['samples_above_3995']) / stats['total_samples']) * 100

    # Expected DC bias should be around 2048 (1.8V / 2 = 0.9V = 2048 counts)
    # If powered from 5V with DC bias at 2.5V, ADC would clip at 1.8V = 4095 counts
    stats['dc_bias_offset'] = float(stats['mean'] - 2048)

    return stats

def print_analysis(stats: dict):
    """Print analysis results"""
    print(f"\n{'='*70}")
    print(f"ADC SAMPLE ANALYSIS: {stats['filename']}")
    print(f"{'='*70}")
    print(f"\nBasic Statistics:")
    print(f"  Total samples:     {stats['total_samples']:,}")
    print(f"  Duration:          {stats['total_samples'] / 40000:.2f} seconds (@ 40 kHz)")
    print(f"  Min value:         {stats['min']} / 4095")
    print(f"  Max value:         {stats['max']} / 4095")
    print(f"  Mean (DC bias):    {stats['mean']:.1f} / 4095 ({stats['mean']/4095*1.8:.3f}V)")
    print(f"  Expected DC bias:  2048 / 4095 (0.900V)")
    print(f"  DC bias error:     {stats['dc_bias_offset']:+.1f} counts ({stats['dc_bias_offset']/4095*1.8:+.3f}V)")
    print(f"  Std deviation:     {stats['std']:.1f}")
    print(f"  Dynamic range:     {stats['range']} counts ({stats['range']/4095*1.8:.3f}V)")

    print(f"\nClipping Analysis:")
    print(f"  Samples at 0:      {stats['samples_at_0']:,} ({stats['pct_at_0']:.2f}%)")
    print(f"  Samples at 4095:   {stats['samples_at_4095']:,} ({stats['pct_at_4095']:.2f}%)")
    print(f"  Samples < 100:     {stats['samples_below_100']:,}")
    print(f"  Samples > 3995:    {stats['samples_above_3995']:,}")
    print(f"  Total clipping:    {stats['pct_clipping']:.2f}%")

    print(f"\n{'='*70}")
    print("DIAGNOSIS:")
    print(f"{'='*70}")

    # Diagnosis
    issues = []

    if stats['pct_at_4095'] > 1.0:
        issues.append("🚨 CRITICAL: Constant clipping at 4095 (>1% of samples)")
        issues.append("   -> Input voltage exceeds 1.8V maximum")
        issues.append("   -> ADC may be damaged")
        issues.append("   -> DISCONNECT MICROPHONE IMMEDIATELY")
    elif stats['pct_at_4095'] > 0.1:
        issues.append("⚠️  WARNING: Frequent clipping at 4095")
        issues.append("   -> Input voltage occasionally exceeds 1.8V")

    if stats['max'] == 4095:
        issues.append("⚠️  Peak value at absolute maximum (4095)")
        issues.append("   -> Signal is being hard-clipped")

    if abs(stats['dc_bias_offset']) > 1000:
        issues.append(f"⚠️  DC bias way off center: {stats['mean']:.0f} (expected ~2048)")
        issues.append(f"   -> Measured: {stats['mean']/4095*1.8:.3f}V, Expected: ~0.9V")
        if stats['mean'] > 3000:
            issues.append("   -> DC bias too high - likely from 5V-powered mic at 2.5V DC")

    if stats['range'] < 200:
        issues.append(f"⚠️  Very narrow dynamic range: {stats['range']} counts")
        issues.append("   -> Signal amplitude is very low")
        issues.append("   -> May need more gain or different mic positioning")

    if stats['std'] < 50:
        issues.append(f"⚠️  Very low variation: std = {stats['std']:.1f}")
        issues.append("   -> Signal is mostly flat/DC")

    if not issues:
        print("✅ No major issues detected")
        print("   Signal appears to be within acceptable range")
    else:
        for issue in issues:
            print(issue)

    print(f"{'='*70}\n")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python check_adc_clipping.py <sample_file.bin> [<sample_file2.bin> ...]")
        sys.exit(1)

    for filepath in sys.argv[1:]:
        path = Path(filepath)
        if not path.exists():
            print(f"ERROR: File not found: {filepath}")
            continue

        stats = analyze_adc_samples(path)
        print_analysis(stats)

if __name__ == '__main__':
    main()
