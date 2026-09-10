#!/usr/bin/env python3
"""
Check if corruption follows a pattern related to buffer size
"""

import struct
import numpy as np
from pathlib import Path

def check_pattern(filepath: Path):
    """Check corruption pattern"""

    # Read raw bytes
    with open(filepath, 'rb') as f:
        data = f.read()

    # Parse as little-endian uint16
    sample_count = len(data) // 2
    samples = np.array(struct.unpack(f'<{sample_count}H', data), dtype=np.uint32)

    # Find all runs of valid/invalid
    is_valid = samples <= 4095

    # Find transitions
    transitions = []
    current_state = is_valid[0]
    run_start = 0

    for i in range(1, len(is_valid)):
        if is_valid[i] != current_state:
            # Transition found
            transitions.append({
                'start': run_start,
                'end': i - 1,
                'length': i - run_start,
                'valid': current_state
            })
            current_state = is_valid[i]
            run_start = i

    # Add final run
    transitions.append({
        'start': run_start,
        'end': len(is_valid) - 1,
        'length': len(is_valid) - run_start,
        'valid': current_state
    })

    print(f"Total transitions: {len(transitions)}")
    print(f"\nFirst 50 runs:")
    print(f"{'Run#':>5} {'Type':>10} {'Start':>10} {'End':>10} {'Length':>10} {'Time(ms)':>10}")
    print("-" * 65)

    for i, run in enumerate(transitions[:50]):
        run_type = "VALID" if run['valid'] else "CORRUPTED"
        time_ms = run['start'] / 40.0  # 40 kHz = 40 samples per ms
        print(f"{i+1:5d} {run_type:>10} {run['start']:10d} {run['end']:10d} {run['length']:10d} {time_ms:10.1f}")

    # Check if run lengths have a pattern
    print(f"\n\nRun length statistics:")
    valid_runs = [r['length'] for r in transitions if r['valid']]
    invalid_runs = [r['length'] for r in transitions if not r['valid']]

    print(f"\nValid runs: {len(valid_runs)}")
    if valid_runs:
        print(f"  Min: {min(valid_runs)}, Max: {max(valid_runs)}, Mean: {np.mean(valid_runs):.1f}, Median: {np.median(valid_runs):.1f}")

    print(f"\nCorrupted runs: {len(invalid_runs)}")
    if invalid_runs:
        print(f"  Min: {min(invalid_runs)}, Max: {max(invalid_runs)}, Mean: {np.mean(invalid_runs):.1f}, Median: {np.median(invalid_runs):.1f}")

def main():
    import sys

    if len(sys.argv) < 2:
        print("Usage: python check_corruption_pattern.py <file.bin>")
        sys.exit(1)

    filepath = Path(sys.argv[1])
    if not filepath.exists():
        print(f"ERROR: File not found: {filepath}")
        sys.exit(1)

    check_pattern(filepath)

if __name__ == '__main__':
    main()
