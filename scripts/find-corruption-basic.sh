#!/bin/bash

# Basic OneDrive Corruption Scanner
set -euo pipefail

SEARCH_ROOT="${1:-.}"
REPORT_FILE="corruption-scan-$(date +%Y%m%d-%H%M%S).txt"
TEMP_DIRS=$(mktemp)

echo "================================================"
echo "OneDrive Corruption Scanner"
echo "================================================"
echo ""
echo "Scanning: $SEARCH_ROOT"
echo ""

{
    echo "OneDrive Corruption Scan Report"
    echo "Generated: $(date)"
    echo "Search Root: $SEARCH_ROOT"
    echo "========================================"
    echo ""
    echo "Corrupted Directories:"
    echo ""
} > "$REPORT_FILE"

# Find all corruption files and extract unique directories
echo "Searching for corruption patterns..."
find "$SEARCH_ROOT" -type f \( \
    -name "*-Persönlich-*" -o \
    -name "*-Personal-*" -o \
    -name "*.tmp" -o \
    -name "*~\$*.tmp" -o \
    -name "*conflict*.tmp" -o \
    -name "*.partial" \
\) 2>/dev/null | while read -r file; do
    dirname "$file"
done | sort -u > "$TEMP_DIRS"

# Display and count results
if [[ -s "$TEMP_DIRS" ]]; then
    echo ""
    echo "Corrupted directories found:"
    echo ""

    while read -r dir; do
        count=$(find "$dir" -maxdepth 1 -type f \( \
            -name "*-Persönlich-*" -o \
            -name "*-Personal-*" -o \
            -name "*.tmp" -o \
            -name "*~\$*.tmp" -o \
            -name "*conflict*.tmp" -o \
            -name "*.partial" \
        \) 2>/dev/null | wc -l | tr -d ' ')

        echo "  • $dir ($count files)"
        echo "$dir|$count" >> "$REPORT_FILE"
    done < "$TEMP_DIRS"

    total_dirs=$(wc -l < "$TEMP_DIRS" | tr -d ' ')
    total_files=$(find "$SEARCH_ROOT" -type f \( \
        -name "*-Persönlich-*" -o \
        -name "*-Personal-*" -o \
        -name "*.tmp" -o \
        -name "*~\$*.tmp" -o \
        -name "*conflict*.tmp" -o \
        -name "*.partial" \
    \) 2>/dev/null | wc -l | tr -d ' ')

    echo ""
    echo "========================================"
    echo "SUMMARY"
    echo "========================================"
    echo "Corrupted directories: $total_dirs"
    echo "Total corrupt files: $total_files"

    {
        echo ""
        echo "========================================"
        echo "SUMMARY"
        echo "========================================"
        echo "Corrupted directories: $total_dirs"
        echo "Total corrupt files: $total_files"
    } >> "$REPORT_FILE"

    echo ""
    echo "Report saved to: $REPORT_FILE"
else
    echo "✓ No corruption found!"
    echo "" >> "$REPORT_FILE"
    echo "No corruption detected." >> "$REPORT_FILE"
fi

# Cleanup
rm -f "$TEMP_DIRS"
