#!/bin/bash

# OneDrive Corruption Root Finder
# Finds subdirectories containing corrupted files and reports their root paths
# Stops searching deeper once corruption is found in a directory

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SEARCH_ROOT="${1:-.}"
REPORT_FILE="corruption-roots-$(date +%Y%m%d-%H%M%S).txt"
TEMP_FILE=$(mktemp)

# Corruption patterns (OneDrive stub files and sync issues)
declare -a CORRUPTION_PATTERNS=(
    "*-Persönlich-*"
    "*-Personal-*"
    "*.tmp"
    "*~$*.tmp"
    "*conflict*.tmp"
    "*.partial"
)

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}OneDrive Corruption Root Directory Scanner${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""
echo -e "Search root: ${GREEN}${SEARCH_ROOT}${NC}"
echo -e "Report file: ${GREEN}${REPORT_FILE}${NC}"
echo ""

# Statistics
total_dirs=0
corrupted_roots=0
total_corrupt_files=0

# Function to check if directory contains corruption
check_directory_for_corruption() {
    local dir="$1"
    local found_corruption=false
    local corrupt_count=0

    # Check for each corruption pattern
    for pattern in "${CORRUPTION_PATTERNS[@]}"; do
        # Use find with maxdepth 1 to only check current directory
        while IFS= read -r -d '' file; do
            if [[ ! -d "$file" ]]; then
                found_corruption=true
                ((corrupt_count++))
                echo "$file" >> "$TEMP_FILE"
            fi
        done < <(find "$dir" -maxdepth 1 -name "$pattern" -print0 2>/dev/null)
    done

    if [[ "$found_corruption" == true ]]; then
        echo "$corrupt_count"
        return 0
    else
        echo "0"
        return 1
    fi
}

# Function to recursively scan directories
scan_directory() {
    local dir="$1"
    local depth="${2:-0}"

    ((total_dirs++))

    # Check if current directory contains corruption
    local corrupt_count
    corrupt_count=$(check_directory_for_corruption "$dir")

    if [[ $corrupt_count -gt 0 ]]; then
        # Found corruption - mark this as corrupted root and stop going deeper
        ((corrupted_roots++))
        ((total_corrupt_files+=corrupt_count))

        echo -e "${RED}[CORRUPTED]${NC} $dir ${YELLOW}(${corrupt_count} corrupt files)${NC}"
        echo "$dir|$corrupt_count" >> "$REPORT_FILE"

        # Don't recurse into subdirectories
        return 0
    fi

    # No corruption found - scan subdirectories
    while IFS= read -r -d '' subdir; do
        scan_directory "$subdir" $((depth + 1))
    done < <(find "$dir" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null | sort -z)
}

# Initialize report file
{
    echo "OneDrive Corruption Root Directory Report"
    echo "Generated: $(date)"
    echo "Search Root: $SEARCH_ROOT"
    echo "========================================"
    echo ""
    echo "Format: DIRECTORY_PATH|CORRUPT_FILE_COUNT"
    echo ""
} > "$REPORT_FILE"

# Start scanning
echo -e "${BLUE}Scanning directories...${NC}"
echo ""

scan_directory "$SEARCH_ROOT"

# Generate summary
echo ""
echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}Scan Complete - Summary${NC}"
echo -e "${BLUE}================================================${NC}"
echo -e "Total directories scanned: ${GREEN}${total_dirs}${NC}"
echo -e "Corrupted root directories: ${RED}${corrupted_roots}${NC}"
echo -e "Total corrupt files found: ${YELLOW}${total_corrupt_files}${NC}"
echo ""

# Add summary to report
{
    echo ""
    echo "========================================"
    echo "SUMMARY"
    echo "========================================"
    echo "Total directories scanned: $total_dirs"
    echo "Corrupted root directories: $corrupted_roots"
    echo "Total corrupt files found: $total_corrupt_files"
    echo ""
    echo "========================================"
    echo "CORRUPTED ROOT DIRECTORIES"
    echo "========================================"
} >> "$REPORT_FILE"

# Extract and list corrupted roots from temp file
if [[ $corrupted_roots -gt 0 ]]; then
    echo -e "${YELLOW}Corrupted root directories:${NC}"
    echo ""

    grep "|" "$REPORT_FILE" | while IFS='|' read -r dir_path file_count; do
        if [[ -n "$dir_path" && -n "$file_count" ]]; then
            echo -e "  ${RED}•${NC} $dir_path ${YELLOW}($file_count files)${NC}"
        fi
    done

    echo ""
    echo -e "${GREEN}Full report saved to: ${REPORT_FILE}${NC}"

    # Create detailed corruption file list
    if [[ -s "$TEMP_FILE" ]]; then
        DETAIL_FILE="corruption-details-$(date +%Y%m%d-%H%M%S).txt"
        {
            echo "Detailed Corruption File List"
            echo "Generated: $(date)"
            echo "========================================"
            echo ""
            sort -u "$TEMP_FILE"
        } > "$DETAIL_FILE"
        echo -e "${GREEN}Detailed file list saved to: ${DETAIL_FILE}${NC}"
    fi
else
    echo -e "${GREEN}✓ No corruption found!${NC}"
fi

# Cleanup
rm -f "$TEMP_FILE"

exit 0
