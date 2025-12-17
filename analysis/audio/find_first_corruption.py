#!/usr/bin/env python3
"""
Find the first corrupted sample
"""

import struct
import numpy as np
from pathlib import Path

def find_first_corruption(filepath: Path):
    """Find the first sample that exceeds 4095"""

    # Read raw bytes
    with open(filepath, 'rb') as f:
        data = f.read()

    # Parse as little-endian uint16
    sample_count = len(data) // 2
    samples = np.array(struct.unpack(f'<{sample_count}H', data), dtype=np.uint32)

    # Find first invalid sample
    invalid_indices = np.where(samples > 4095)[0]

    if len(invalid_indices) == 0:
        print("No corrupted samples found!")
        return

    first_bad_idx = invalid_indices[0]

    print(f"First corrupted sample at index: {first_bad_idx}")
    print(f"Time: {first_bad_idx / 40000:.3f} seconds")
    print(f"\nSamples around first corruption:")
    print(f"{'Index':>10} {'Value':>8} {'Valid':>6} {'Hex (LE)':>12}")
    print("-" * 40)

    start = max(0, first_bad_idx - 10)
    end = min(len(samples), first_bad_idx + 10)

    for i in range(start, end):
        value = samples[i]
        valid = "OK" if value <= 4095 else "BAD"
        # Get hex representation
        byte_offset = i * 2
        low_byte = data[byte_offset]
        high_byte = data[byte_offset + 1]
        hex_str = f"{high_byte:02x} {low_byte:02x}"

        marker = " <--" if i == first_bad_idx else ""
        print(f"{i:10d} {value:8d} {valid:>6} {hex_str:>12}{marker}")

    # Check pattern of corruption
    print(f"\n\nPattern around first corruption (V=valid, X=corrupted):")
    for i in range(max(0, first_bad_idx - 50), min(len(samples), first_bad_idx + 50), 10):
        chunk_validity = ['V' if samples[j] <= 4095 else 'X' for j in range(i, min(i+10, len(samples)))]
        marker = " <--" if i <= first_bad_idx < i + 10 else ""
        print(f"  Samples {i:6d}-{min(i+9, len(samples)-1):6d}: {''.join(chunk_validity)}{marker}")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python find_first_corruption.py <file.bin>")
        sys.exit(1)

    filepath = Path(sys.argv[1])
    if not filepath.exists():
        print(f"ERROR: File not found: {filepath}")
        sys.exit(1)

    find_first_corruption(filepath)

if __name__ == '__main__':
    main()
