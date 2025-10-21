#!/bin/bash

# Workspace Corruption Scanner
# Scans all workspace subdirectories for OneDrive corruption
# Separates regular files from .git directories

set -euo pipefail

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
WORKSPACE_ROOT="${1:-/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIAGNOSE_SCRIPT="${SCRIPT_DIR}/diagnose-onedrive-corruption.sh"
REPORT_FILE="${SCRIPT_DIR}/workspace-corruption-report-$(date +%Y%m%d-%H%M%S).txt"
TEMP_DIR="/tmp/workspace-corruption-$$"

# Verify script exists
if [ ! -f "$DIAGNOSE_SCRIPT" ]; then
    echo -e "${RED}ERROR: diagnose-onedrive-corruption.sh not found at ${DIAGNOSE_SCRIPT}${NC}"
    exit 1
fi

# Create temp directory
mkdir -p "$TEMP_DIR"

# Cleanup on exit
trap "rm -rf $TEMP_DIR" EXIT

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      Workspace OneDrive Corruption Scanner                    ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Workspace Root: $WORKSPACE_ROOT"
echo "Report File: $REPORT_FILE"
echo "Start Time: $(date)"
echo ""

# Initialize report
cat > "$REPORT_FILE" << EOF
═══════════════════════════════════════════════════════════════
  WORKSPACE ONEDRIVE CORRUPTION SCAN REPORT
═══════════════════════════════════════════════════════════════

Scan Date: $(date)
Workspace Root: $WORKSPACE_ROOT
Scanner Version: 1.0

═══════════════════════════════════════════════════════════════

EOF

# Global counters
TOTAL_PROJECTS=0
PROJECTS_WITH_ISSUES=0
TOTAL_CORRUPTED_FILES=0
TOTAL_GIT_CORRUPTED=0

# Function to scan a single directory
scan_directory() {
    local dir="$1"
    local scan_type="$2"  # "regular" or "git"
    local project_name="$3"

    local temp_output="${TEMP_DIR}/scan_${project_name}_${scan_type}.txt"

    echo -e "${BLUE}  → Scanning ${scan_type} files in ${project_name}...${NC}"

    # Run diagnostic script and capture output
    if [ "$scan_type" = "git" ]; then
        # Only scan .git directory
        if [ -d "${dir}/.git" ]; then
            bash "$DIAGNOSE_SCRIPT" "${dir}/.git" > "$temp_output" 2>&1
        else
            echo "No .git directory found" > "$temp_output"
            return 0
        fi
    else
        # Scan everything except .git
        # Create temporary directory structure excluding .git
        local temp_scan_dir="${TEMP_DIR}/scan_${project_name}"
        mkdir -p "$temp_scan_dir"

        # Use rsync to copy structure excluding .git
        rsync -a --exclude='.git' "$dir/" "$temp_scan_dir/" 2>/dev/null || true

        bash "$DIAGNOSE_SCRIPT" "$temp_scan_dir" > "$temp_output" 2>&1

        # Cleanup temp scan dir
        rm -rf "$temp_scan_dir"
    fi

    # Parse results
    local unreadable=$(grep "Found .* unreadable files" "$temp_output" | grep -oE '[0-9]+' | head -1 || echo "0")
    local zero_byte=$(grep "Found .* zero-byte files" "$temp_output" | grep -oE '[0-9]+' | head -1 || echo "0")
    local slow_files=$(grep "Found .* slow/timeout files" "$temp_output" | grep -oE '[0-9]+' | head -1 || echo "0")
    local onedrive_attrs=$(grep "Found .* files with OneDrive attributes" "$temp_output" | grep -oE '[0-9]+' | head -1 || echo "0")

    local total_issues=$((unreadable + zero_byte + slow_files))

    if [ $total_issues -gt 0 ]; then
        echo -e "${RED}    ✗ Found ${total_issues} issues (${scan_type})${NC}"
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

    # Scan regular files (excluding .git)
    local regular_output="${TEMP_DIR}/regular_${project_name}.txt"
    scan_directory "$project_dir" "regular" "$project_name"
    local regular_issues=$?

    # Scan .git directory separately
    local git_output="${TEMP_DIR}/git_${project_name}.txt"
    scan_directory "$project_dir" "git" "$project_name"
    local git_issues=$?

    # Update global counters
    if [ $regular_issues -gt 0 ] || [ $git_issues -gt 0 ]; then
        PROJECTS_WITH_ISSUES=$((PROJECTS_WITH_ISSUES + 1))
        TOTAL_CORRUPTED_FILES=$((TOTAL_CORRUPTED_FILES + regular_issues))
        TOTAL_GIT_CORRUPTED=$((TOTAL_GIT_CORRUPTED + git_issues))
    fi

    # Generate project report
    cat >> "$REPORT_FILE" << EOF

───────────────────────────────────────────────────────────────
PROJECT: ${project_name}
───────────────────────────────────────────────────────────────

REGULAR FILES (excluding .git):
EOF

    if [ -f "${TEMP_DIR}/scan_${project_name}_regular.txt" ]; then
        cat "${TEMP_DIR}/scan_${project_name}_regular.txt" >> "$REPORT_FILE"
    fi

    cat >> "$REPORT_FILE" << EOF

.GIT DIRECTORY:
EOF

    if [ -f "${TEMP_DIR}/scan_${project_name}_git.txt" ]; then
        cat "${TEMP_DIR}/scan_${project_name}_git.txt" >> "$REPORT_FILE"
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
    # Skip if not a directory
    [ -d "$project_dir" ] || continue

    # Skip hidden directories
    [[ "$(basename "$project_dir")" == .* ]] && continue

    # Process the project
    process_project "$project_dir"

done < <(find "$WORKSPACE_ROOT" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null | sort -z)

# Generate final summary
echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      SCAN COMPLETE                                            ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Display summary
echo -e "${YELLOW}═══ SUMMARY ═══${NC}"
echo "Total Projects Scanned: ${TOTAL_PROJECTS}"
echo "Projects with Issues: ${PROJECTS_WITH_ISSUES}"
echo "Total Corrupted Files: ${TOTAL_CORRUPTED_FILES}"
echo "Corrupted Git Files: ${TOTAL_GIT_CORRUPTED}"
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
Total Corrupted Files (regular): ${TOTAL_CORRUPTED_FILES}
Total Corrupted Files (.git): ${TOTAL_GIT_CORRUPTED}

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
fi

# Open report in default editor if issues found
if [ $PROJECTS_WITH_ISSUES -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}Opening report in default editor...${NC}"
    open "$REPORT_FILE" 2>/dev/null || cat "$REPORT_FILE"
fi

exit 0
