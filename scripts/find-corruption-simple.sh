#!/bin/bash

# Simplified OneDrive Corruption Scanner
set -euo pipefail

SEARCH_ROOT="${1:-.}"
REPORT_FILE="corruption-scan-$(date +%Y%m%d-%H%M%S).txt"

echo "Scanning: $SEARCH_ROOT"
echo ""

# Find all corruption patterns
echo "Searching for OneDrive corruption patterns..."
echo ""

{
    echo "OneDrive Corruption Scan Report"
    echo "Generated: $(date)"
    echo "Search Root: $SEARCH_ROOT"
    echo "========================================"
    echo ""
} > "$REPORT_FILE"

# Track corrupted directories
declare -A corrupted_dirs

# Find files matching corruption patterns
while IFS= read -r file; do
    # Get parent directory
    dir=$(dirname "$file")

    # Mark this directory as corrupted
    if [[ -z "${corrupted_dirs[$dir]:-}" ]]; then
        corrupted_dirs[$dir]=1
        echo "CORRUPTED: $dir"
        echo "$dir" >> "$REPORT_FILE"
    else
        ((corrupted_dirs[$dir]++))
    fi
done < <(find "$SEARCH_ROOT" -type f \( \
    -name "*-Persönlich-*" -o \
    -name "*-Personal-*" -o \
    -name "*.tmp" -o \
    -name "*~\$*.tmp" -o \
    -name "*conflict*.tmp" -o \
    -name "*.partial" \
\) 2>/dev/null)

# Summary
echo ""
echo "========================================"
echo "SUMMARY"
echo "========================================"
total_corrupted=${#corrupted_dirs[@]}
total_files=0
for count in "${corrupted_dirs[@]}"; do
    ((total_files+=count))
done

echo "Corrupted directories: $total_corrupted"
echo "Total corrupt files: $total_files"

{
    echo ""
    echo "========================================"
    echo "SUMMARY"
    echo "========================================"
    echo "Corrupted directories: $total_corrupted"
    echo "Total corrupt files: $total_files"
} >> "$REPORT_FILE"

echo ""
echo "Report saved to: $REPORT_FILE"
