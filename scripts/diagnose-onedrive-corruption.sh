#!/bin/bash

# OneDrive Corruption Detection Script
# Identifies files that may be corrupted or blocked by OneDrive

echo "=== OneDrive Corruption Diagnostics ==="
echo "Date: $(date)"
echo ""

TARGET_DIR="${1:-.}"

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "Scanning directory: $TARGET_DIR"
echo ""

# 1. Check for OneDrive extended attributes
echo "1. Checking for OneDrive sync attributes..."
ONEDRIVE_ATTR_COUNT=0
while IFS= read -r -d '' file; do
    if xattr -l "$file" 2>/dev/null | grep -q "com.microsoft.OneDrive"; then
        echo -e "${YELLOW}OneDrive attribute found:${NC} $file"
        ((ONEDRIVE_ATTR_COUNT++))
    fi
done < <(find "$TARGET_DIR" -type f -print0 2>/dev/null)
echo "Found $ONEDRIVE_ATTR_COUNT files with OneDrive attributes"
echo ""

# 2. Check for files that can't be read
echo "2. Checking for unreadable files..."
UNREADABLE_COUNT=0
while IFS= read -r -d '' file; do
    if ! cat "$file" > /dev/null 2>&1; then
        echo -e "${RED}UNREADABLE:${NC} $file"
        ls -lh "$file" 2>/dev/null
        ((UNREADABLE_COUNT++))
    fi
done < <(find "$TARGET_DIR" -type f -print0 2>/dev/null | head -n 1000)
echo "Found $UNREADABLE_COUNT unreadable files"
echo ""

# 3. Check for zero-byte files (often corruption indicator)
echo "3. Checking for zero-byte files..."
ZERO_BYTE_COUNT=$(find "$TARGET_DIR" -type f -size 0 2>/dev/null | wc -l)
echo "Found $ZERO_BYTE_COUNT zero-byte files"
if [ $ZERO_BYTE_COUNT -gt 0 ]; then
    echo "Sample zero-byte files:"
    find "$TARGET_DIR" -type f -size 0 2>/dev/null | head -5
fi
echo ""

# 4. Check for files with "Operation timed out" errors
echo "4. Testing file access speed..."
SLOW_FILES=0
TEST_COUNT=0
while IFS= read -r -d '' file; do
    TEST_COUNT=$((TEST_COUNT + 1))
    if [ $TEST_COUNT -gt 100 ]; then
        break
    fi

    START=$(date +%s%N)
    timeout 2 cat "$file" > /dev/null 2>&1
    EXIT_CODE=$?
    END=$(date +%s%N)
    DURATION=$(( (END - START) / 1000000 )) # Convert to milliseconds

    if [ $EXIT_CODE -eq 124 ] || [ $DURATION -gt 1000 ]; then
        echo -e "${RED}SLOW/TIMEOUT:${NC} $file (${DURATION}ms)"
        ((SLOW_FILES++))
    fi
done < <(find "$TARGET_DIR" -type f -print0 2>/dev/null)
echo "Found $SLOW_FILES slow/timeout files out of $TEST_COUNT tested"
echo ""

# 5. Check OneDrive process status
echo "5. OneDrive Process Status..."
if pgrep -x "OneDrive" > /dev/null; then
    echo -e "${GREEN}OneDrive is RUNNING${NC}"
    ps aux | grep "[O]neDrive" | head -3
else
    echo -e "${YELLOW}OneDrive is NOT RUNNING${NC}"
fi
echo ""

# 6. Check for .cloud files (OneDrive placeholder files)
echo "6. Checking for OneDrive placeholder files..."
CLOUD_FILES=$(find "$TARGET_DIR" -name "*.cloud" 2>/dev/null | wc -l)
echo "Found $CLOUD_FILES .cloud placeholder files"
if [ $CLOUD_FILES -gt 0 ]; then
    echo "Sample .cloud files:"
    find "$TARGET_DIR" -name "*.cloud" 2>/dev/null | head -5
fi
echo ""

# 7. Summary
echo "=== SUMMARY ==="
echo "OneDrive attributes: $ONEDRIVE_ATTR_COUNT"
echo "Unreadable files: $UNREADABLE_COUNT"
echo "Zero-byte files: $ZERO_BYTE_COUNT"
echo "Slow/timeout files: $SLOW_FILES"
echo "Cloud placeholder files: $CLOUD_FILES"
echo ""

# 8. Recommendations
if [ $UNREADABLE_COUNT -gt 0 ] || [ $SLOW_FILES -gt 5 ]; then
    echo -e "${RED}=== RECOMMENDATIONS ===${NC}"
    echo "• Stop OneDrive sync: File → Pause syncing"
    echo "• Move repository outside OneDrive"
    echo "• Exclude .git directories from OneDrive sync"
    echo "• Consider using: git config core.fscache true"
fi
