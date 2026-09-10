#!/usr/bin/env python3
"""
Diagnose binary format issues in captured ADC samples
"""

import struct
import numpy as np
from pathlib import Path

def analyze_binary_format(filepath: Path):
    """Analyze binary file to determine actual format"""

    print(f"\n{'='*70}")
    print(f"BINARY FORMAT ANALYSIS: {filepath.name}")
    print(f"{'='*70}\n")

    # Read raw bytes
    with open(filepath, 'rb') as f:
        data = f.read()

    file_size = len(data)
    print(f"File size: {file_size:,} bytes")

    # Try different interpretations
    print(f"\nFirst 32 bytes (hex):")
    print(' '.join(f'{b:02x}' for b in data[:32]))

    # Interpretation 1: Little-endian uint16
    print(f"\n--- Interpretation 1: Little-Endian uint16 ---")
    if len(data) >= 20:
        samples_u16_le = struct.unpack('<10H', data[:20])
        print(f"First 10 samples: {samples_u16_le}")
        print(f"Min: {min(samples_u16_le)}, Max: {max(samples_u16_le)}")

        # Check if any are > 4095
        over_range = [s for s in samples_u16_le if s > 4095]
        print(f"Samples > 4095: {len(over_range)} / 10")

    # Interpretation 2: Big-endian uint16
    print(f"\n--- Interpretation 2: Big-Endian uint16 ---")
    if len(data) >= 20:
        samples_u16_be = struct.unpack('>10H', data[:20])
        print(f"First 10 samples: {samples_u16_be}")
        print(f"Min: {min(samples_u16_be)}, Max: {max(samples_u16_be)}")

        # Check if any are > 4095
        over_range = [s for s in samples_u16_be if s > 4095]
        print(f"Samples > 4095: {len(over_range)} / 10")

    # Interpretation 3: uint8 (bytes)
    print(f"\n--- Interpretation 3: uint8 (individual bytes) ---")
    bytes_first = data[:20]
    print(f"First 20 bytes as decimal: {[b for b in bytes_first]}")
    print(f"Min: {min(bytes_first)}, Max: {max(bytes_first)}")

    # Interpretation 4: Check for patterns
    print(f"\n--- Pattern Analysis ---")

    # Count samples in valid range (0-4095) for little-endian uint16
    sample_count = len(data) // 2
    samples = struct.unpack(f'<{sample_count}H', data)

    valid = sum(1 for s in samples if s <= 4095)
    print(f"Total samples (LE uint16): {sample_count:,}")
    print(f"In valid range (0-4095): {valid:,} ({valid/sample_count*100:.1f}%)")
    print(f"Out of range: {sample_count - valid:,} ({(sample_count-valid)/sample_count*100:.1f}%)")

    # Histogram of high byte values
    high_bytes = [data[i+1] for i in range(0, min(len(data), 10000), 2)]
    print(f"\nHigh byte histogram (first 5000 samples):")
    unique, counts = np.unique(high_bytes, return_counts=True)
    # Show most common high byte values
    sorted_idx = np.argsort(counts)[::-1]
    for i in sorted_idx[:10]:
        print(f"  0x{unique[i]:02x} ({unique[i]:3d}): {counts[i]:5d} occurrences ({counts[i]/len(high_bytes)*100:5.1f}%)")

    # Statistical analysis
    print(f"\n--- Statistical Analysis (as LE uint16) ---")
    samples_arr = np.array(samples, dtype=np.uint32)
    print(f"Mean: {np.mean(samples_arr):.1f}")
    print(f"Median: {np.median(samples_arr):.1f}")
    print(f"Std: {np.std(samples_arr):.1f}")
    print(f"Min: {np.min(samples_arr)}")
    print(f"Max: {np.max(samples_arr)}")

    # Check if low byte looks reasonable
    low_bytes = [data[i] for i in range(0, min(len(data), 10000), 2)]
    print(f"\n--- Low Byte Analysis ---")
    print(f"Low byte mean: {np.mean(low_bytes):.1f}")
    print(f"Low byte std: {np.std(low_bytes):.1f}")
    print(f"Low byte range: {min(low_bytes)} - {max(low_bytes)}")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python diagnose_binary_format.py <file.bin> [<file2.bin> ...]")
        sys.exit(1)

    for filepath in sys.argv[1:]:
        path = Path(filepath)
        if not path.exists():
            print(f"ERROR: File not found: {filepath}")
            continue

        analyze_binary_format(path)

if __name__ == '__main__':
    main()
