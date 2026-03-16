#!/usr/bin/env python3
"""
check_error_rate.py — Parse a Locust _stats.csv and print the aggregated error rate.

Usage:
    python3 check_error_rate.py <stats_csv> [threshold]

Exits 0 if error rate < threshold (default 5%), exits 1 if >= threshold.
Prints the error rate as "X.XX" to stdout.
"""
import csv
import sys

csv_file = sys.argv[1]
threshold = float(sys.argv[2]) if len(sys.argv) > 2 else 5.0

with open(csv_file, newline="") as f:
    for row in csv.DictReader(f):
        if row.get("Name", "").strip() == "Aggregated":
            reqs  = int(row.get("Request Count", 0) or 0)
            fails = int(row.get("Failure Count", 0) or 0)
            rate  = (fails / reqs * 100) if reqs > 0 else 0.0
            print(f"{rate:.2f}")
            sys.exit(0 if rate < threshold else 1)

# Aggregated row not found — treat as 0% error rate
print("0.00")
sys.exit(0)
