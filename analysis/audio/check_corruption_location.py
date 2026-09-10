#!/usr/bin/env python3
"""
Check where in the file corruption starts
"""

import struct
import numpy as np
from pathlib import Path

def check_corruption_location(filepath: Path):
    """Check where corruption begins in the file"""

    print(f"\n{'='*70}")
    print(f"CORRUPTION LOCATION ANALYSIS: {filepath.name}")
    print(f"{'='*70}\n")

    # Read raw bytes
    with open(filepath, 'rb') as f:
        data = f.read()

    # Parse as little-endian uint16
    sample_count = len(data) // 2
    samples = np.array(struct.unpack(f'<{sample_count}H', data), dtype=np.uint32)

    print(f"Total samples: {sample_count:,}")

    # Check samples in chunks
    chunk_size = 10000
    print(f"\nChecking in chunks of {chunk_size:,} samples:")
    print(f"{'Chunk':>8} {'Start':>10} {'End':>10} {'Valid':>8} {'%Valid':>8} {'MinVal':>8} {'MaxVal':>8} {'MeanVal':>10}")
    print("-" * 70)

    for start_idx in range(0, sample_count, chunk_size):
        end_idx = min(start_idx + chunk_size, sample_count)
        chunk = samples[start_idx:end_idx]

        valid = np.sum(chunk <= 4095)
        pct_valid = valid / len(chunk) * 100
        min_val = np.min(chunk)
        max_val = np.max(chunk)
        mean_val = np.mean(chunk)

        chunk_num = start_idx // chunk_size + 1
        print(f"{chunk_num:8d} {start_idx:10d} {end_idx:10d} {valid:8d} {pct_valid:7.1f}% {min_val:8d} {max_val:8d} {mean_val:10.1f}")

    # Check first and last 100 valid samples
    valid_indices = np.where(samples <= 4095)[0]
    print(f"\n\nFirst 100 valid sample indices:")
    print(valid_indices[:100].tolist() if len(valid_indices) >= 100 else valid_indices.tolist())

    print(f"\nFirst 100 valid sample values:")
    first_valid_samples = samples[valid_indices[:100]] if len(valid_indices) >= 100 else samples[valid_indices]
    print(f"Values: min={np.min(first_valid_samples)}, max={np.max(first_valid_samples)}, mean={np.mean(first_valid_samples):.1f}")

    # Check for runs of valid vs invalid
    is_valid = samples <= 4095
    print(f"\n\nPattern of valid/invalid samples (first 200):")
    for i in range(0, min(200, len(samples)), 20):
        chunk_validity = ['V' if is_valid[j] else 'X' for j in range(i, min(i+20, len(samples)))]
        print(f"  Samples {i:5d}-{min(i+19, len(samples)-1):5d}: {''.join(chunk_validity)}")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python check_corruption_location.py <file.bin>")
        sys.exit(1)

    filepath = Path(sys.argv[1])
    if not filepath.exists():
        print(f"ERROR: File not found: {filepath}")
        sys.exit(1)

    check_corruption_location(filepath)

if __name__ == '__main__':
    main()
