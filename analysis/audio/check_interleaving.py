#!/usr/bin/env python3
"""
Check if samples are interleaved or have stride issues
"""

import struct
import numpy as np
from pathlib import Path

def check_interleaving(filepath: Path):
    """Check if even/odd samples have different characteristics"""

    print(f"\n{'='*70}")
    print(f"INTERLEAVING ANALYSIS: {filepath.name}")
    print(f"{'='*70}\n")

    # Read raw bytes
    with open(filepath, 'rb') as f:
        data = f.read()

    # Parse as little-endian uint16
    sample_count = len(data) // 2
    samples = np.array(struct.unpack(f'<{sample_count}H', data), dtype=np.uint32)

    print(f"Total samples: {sample_count:,}")

    # Split into even and odd indexed samples
    even_samples = samples[0::2]
    odd_samples = samples[1::2]

    print(f"\n--- Even-indexed samples (0, 2, 4, ...) ---")
    print(f"Count: {len(even_samples):,}")
    print(f"First 10: {even_samples[:10].tolist()}")
    print(f"Min: {np.min(even_samples)}, Max: {np.max(even_samples)}")
    print(f"Mean: {np.mean(even_samples):.1f}, Std: {np.std(even_samples):.1f}")
    even_valid = np.sum(even_samples <= 4095)
    print(f"In valid range (0-4095): {even_valid:,} ({even_valid/len(even_samples)*100:.1f}%)")

    print(f"\n--- Odd-indexed samples (1, 3, 5, ...) ---")
    print(f"Count: {len(odd_samples):,}")
    print(f"First 10: {odd_samples[:10].tolist()}")
    print(f"Min: {np.min(odd_samples)}, Max: {np.max(odd_samples)}")
    print(f"Mean: {np.mean(odd_samples):.1f}, Std: {np.std(odd_samples):.1f}")
    odd_valid = np.sum(odd_samples <= 4095)
    print(f"In valid range (0-4095): {odd_valid:,} ({odd_valid/len(odd_samples)*100:.1f}%)")

    # Check byte-level patterns
    print(f"\n--- Byte-level Analysis ---")

    # Check if reading every 4th byte makes sense (potential uint32 misinterpretation)
    print(f"\nChecking if data is actually uint32 with only 12 bits used...")

    # Try reading as uint32 and check if only low 12 bits are used
    if len(data) % 4 == 0:
        sample_count_u32 = len(data) // 4
        samples_u32 = np.array(struct.unpack(f'<{sample_count_u32}I', data), dtype=np.uint32)

        print(f"As uint32: {sample_count_u32:,} samples")
        print(f"First 10: {samples_u32[:10].tolist()}")
        print(f"Min: {np.min(samples_u32)}, Max: {np.max(samples_u32)}")

        # Check if most significant 20 bits are zero
        valid_u32 = np.sum(samples_u32 <= 4095)
        print(f"In valid ADC range (0-4095): {valid_u32:,} ({valid_u32/len(samples_u32)*100:.1f}%)")

        if valid_u32 / len(samples_u32) > 0.8:
            print("✅ Data appears to be uint32 with 12-bit ADC values!")
        else:
            print("❌ Not uint32 format")

    # Check consecutive byte patterns
    print(f"\n--- Consecutive Bytes Pattern ---")
    print("First 40 bytes grouped by 4:")
    for i in range(0, min(40, len(data)), 4):
        byte_group = data[i:i+4]
        as_u32 = struct.unpack('<I', byte_group)[0] if len(byte_group) == 4 else 0
        as_u16_le = (struct.unpack('<H', byte_group[:2])[0], struct.unpack('<H', byte_group[2:4])[0]) if len(byte_group) == 4 else (0, 0)
        print(f"  Bytes {i:3d}-{i+3:3d}: {' '.join(f'{b:02x}' for b in byte_group)} | u32={as_u32:6d} | u16={as_u16_le[0]:5d},{as_u16_le[1]:5d}")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python check_interleaving.py <file.bin> [<file2.bin> ...]")
        sys.exit(1)

    for filepath in sys.argv[1:]:
        path = Path(filepath)
        if not path.exists():
            print(f"ERROR: File not found: {filepath}")
            continue

        check_interleaving(path)

if __name__ == '__main__':
    main()
