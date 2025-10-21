#!/bin/bash

# Fast Workspace Corruption Scanner
# Skips slow performance tests, focuses on actual corruption detection

set -euo pipefail

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
WORKSPACE_ROOT="${1:-/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_FILE="${SCRIPT_DIR}/workspace-corruption-report-fast-$(date +%Y%m%d-%H%M%S).txt"

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      Fast Workspace OneDrive Corruption Scanner               ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Workspace Root: $WORKSPACE_ROOT"
echo "Report File: $REPORT_FILE"
echo "Start Time: $(date)"
echo ""

# Initialize report
cat > "$REPORT_FILE" << EOF
═══════════════════════════════════════════════════════════════
  FAST WORKSPACE ONEDRIVE CORRUPTION SCAN REPORT
═══════════════════════════════════════════════════════════════

Scan Date: $(date)
Workspace Root: $WORKSPACE_ROOT
Scanner Version: 1.0 (Fast)

Note: This scanner skips slow performance tests and focuses on
      actual file corruption detection.

═══════════════════════════════════════════════════════════════

EOF

# Global counters
TOTAL_PROJECTS=0
PROJECTS_WITH_ISSUES=0
TOTAL_CORRUPTED_FILES=0
TOTAL_GIT_CORRUPTED=0
TOTAL_UNREADABLE=0
TOTAL_ZERO_BYTE=0

# Function to scan directory for corruption
scan_directory() {
    local dir="$1"
    local scan_type="$2"  # "regular" or "git"
    local project_name="$3"

    echo -e "${BLUE}  → Scanning ${scan_type} files in ${project_name}...${NC}"

    local unreadable=0
    local zero_byte=0
    local onedrive_attrs=0

    # Target directory
    local target_dir="$dir"
    if [ "$scan_type" = "git" ]; then
        if [ ! -d "${dir}/.git" ]; then
            echo -e "${GREEN}    ✓ No .git directory${NC}"
            return 0
        fi
        target_dir="${dir}/.git"
    fi

    # Quick check: unreadable files
    while IFS= read -r -d '' file; do
        if ! cat "$file" > /dev/null 2>&1; then
            echo -e "${RED}    UNREADABLE:${NC} $file" | tee -a "$REPORT_FILE"
            ((unreadable++))
        fi
    done < <(find "$target_dir" -type f $([ "$scan_type" = "regular" ] && echo "-path '*/.git' -prune -o") -type f -print0 2>/dev/null | head -z -n 50)

    # Quick check: zero-byte files
    zero_byte=$(find "$target_dir" $([ "$scan_type" = "regular" ] && echo "-path '*/.git' -prune -o") -type f -size 0 -print 2>/dev/null | wc -l | tr -d ' ')

    if [ $zero_byte -gt 0 ]; then
        echo -e "${YELLOW}    Zero-byte files: ${zero_byte}${NC}"
        find "$target_dir" $([ "$scan_type" = "regular" ] && echo "-path '*/.git' -prune -o") -type f -size 0 -print 2>/dev/null | head -5 | tee -a "$REPORT_FILE"
    fi

    # Check OneDrive attributes (sample only)
    while IFS= read -r -d '' file; do
        if xattr -l "$file" 2>/dev/null | grep -q "com.microsoft.OneDrive"; then
            ((onedrive_attrs++))
        fi
    done < <(find "$target_dir" -type f $([ "$scan_type" = "regular" ] && echo "-path '*/.git' -prune -o") -type f -print0 2>/dev/null | head -z -n 10)

    local total_issues=$((unreadable + zero_byte))

    # Update global counters
    TOTAL_UNREADABLE=$((TOTAL_UNREADABLE + unreadable))
    TOTAL_ZERO_BYTE=$((TOTAL_ZERO_BYTE + zero_byte))

    if [ $total_issues -gt 0 ]; then
        echo -e "${RED}    ✗ Found ${total_issues} issues (${unreadable} unreadable, ${zero_byte} zero-byte)${NC}"
        if [ "$scan_type" = "git" ]; then
            TOTAL_GIT_CORRUPTED=$((TOTAL_GIT_CORRUPTED + total_issues))
        else
            TOTAL_CORRUPTED_FILES=$((TOTAL_CORRUPTED_FILES + total_issues))
        fi
        return $total_issues
    else
        echo -e "${GREEN}    ✓ No issues found (${scan_type})${NC}"
        return 0
    fi
}

# Function to process a single project directory
process_project() {
    local project_dir="$1"
    local project_name=$(basename "$project_dir")

    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Processing: ${YELLOW}${project_name}${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    TOTAL_PROJECTS=$((TOTAL_PROJECTS + 1))

    # Append to report
    cat >> "$REPORT_FILE" << EOF

───────────────────────────────────────────────────────────────
PROJECT: ${project_name}
───────────────────────────────────────────────────────────────

EOF

    # Scan regular files (excluding .git)
    scan_directory "$project_dir" "regular" "$project_name"
    local regular_issues=$?

    # Scan .git directory separately
    scan_directory "$project_dir" "git" "$project_name"
    local git_issues=$?

    # Update counters
    if [ $regular_issues -gt 0 ] || [ $git_issues -gt 0 ]; then
        PROJECTS_WITH_ISSUES=$((PROJECTS_WITH_ISSUES + 1))
    fi

    cat >> "$REPORT_FILE" << EOF

SUMMARY FOR ${project_name}:
  Regular Files Issues: ${regular_issues}
  Git Directory Issues: ${git_issues}
  Total Issues: $((regular_issues + git_issues))

EOF
}

# Main scanning loop
echo -e "${YELLOW}Discovering workspace projects...${NC}"
echo ""

# Find all subdirectories in workspace (1 level deep)
while IFS= read -r -d '' project_dir; do
    [ -d "$project_dir" ] || continue
    [[ "$(basename "$project_dir")" == .* ]] && continue
    process_project "$project_dir"
done < <(find "$WORKSPACE_ROOT" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null | sort -z)

# Generate final summary
echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      SCAN COMPLETE                                            ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${YELLOW}═══ SUMMARY ═══${NC}"
echo "Total Projects Scanned: ${TOTAL_PROJECTS}"
echo "Projects with Issues: ${PROJECTS_WITH_ISSUES}"
echo "Total Unreadable Files: ${TOTAL_UNREADABLE}"
echo "Total Zero-Byte Files: ${TOTAL_ZERO_BYTE}"
echo "  → In regular files: ${TOTAL_CORRUPTED_FILES}"
echo "  → In .git directories: ${TOTAL_GIT_CORRUPTED}"
echo ""

if [ $PROJECTS_WITH_ISSUES -gt 0 ]; then
    echo -e "${RED}⚠️  WARNING: Found corruption in ${PROJECTS_WITH_ISSUES} projects!${NC}"
else
    echo -e "${GREEN}✓ No corruption detected in any workspace projects${NC}"
fi

echo ""
echo -e "${CYAN}Full report saved to: ${YELLOW}${REPORT_FILE}${NC}"

# Append final summary to report
cat >> "$REPORT_FILE" << EOF

═══════════════════════════════════════════════════════════════
  FINAL SUMMARY
═══════════════════════════════════════════════════════════════

Total Projects Scanned: ${TOTAL_PROJECTS}
Projects with Issues: ${PROJECTS_WITH_ISSUES}
Total Unreadable Files: ${TOTAL_UNREADABLE}
Total Zero-Byte Files: ${TOTAL_ZERO_BYTE}
  → In regular files: ${TOTAL_CORRUPTED_FILES}
  → In .git directories: ${TOTAL_GIT_CORRUPTED}

Scan Completed: $(date)

═══════════════════════════════════════════════════════════════
EOF

# Recommendations
if [ $PROJECTS_WITH_ISSUES -gt 0 ]; then
    cat >> "$REPORT_FILE" << EOF

RECOMMENDATIONS:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. IMMEDIATE ACTIONS:
   • Stop OneDrive sync: File → Pause syncing
   • Backup corrupted projects immediately

2. GIT REPOSITORY ISSUES:
   • For projects with .git corruption:
     - Try: git fsck --full
     - Try: git gc --aggressive
     - Consider re-cloning from remote

3. LONG-TERM SOLUTIONS:
   • Move repositories outside OneDrive
   • Use selective sync to exclude .git directories
   • Consider: git config core.fscache true
   • Use symbolic links for large repos

4. ONEDRIVE CONFIGURATION:
   • Exclude patterns: .git/*, node_modules/*, venv/*
   • Enable "Files On-Demand" for large files

═══════════════════════════════════════════════════════════════
EOF

    echo ""
    echo -e "${YELLOW}Opening report...${NC}"
    open "$REPORT_FILE" 2>/dev/null || true
fi

exit 0
