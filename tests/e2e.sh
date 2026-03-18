#!/bin/bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load config and helpers
source "${SCRIPT_DIR}/lib/config.sh"
source "${SCRIPT_DIR}/lib/helpers.sh"

# Parse args
DO_SYNC=false
SELECTED_CASES=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --sync) DO_SYNC=true; shift ;;
        --help|-h)
            echo "Usage: $0 [--sync] [case_numbers...]"
            echo ""
            echo "  --sync          Sync local files to server before testing"
            echo "  case_numbers    Run specific cases (e.g., 03 04)"
            echo ""
            echo "Examples:"
            echo "  $0              Run all tests"
            echo "  $0 --sync       Sync files, then run all tests"
            echo "  $0 03 04        Run only cases 03 and 04"
            exit 0
            ;;
        *)  SELECTED_CASES+=("$1"); shift ;;
    esac
done

echo ""
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo -e "${BOLD}  AniLaunchpad E2E Tests${NC}"
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo "  Server: ${TEST_SERVER}"
echo "  Domain: ${TEST_DOMAIN}"
echo ""

# Verify SSH connectivity
if ! ssh_exec "echo ok" >/dev/null 2>&1; then
    echo -e "${RED}Cannot connect to ${TEST_SERVER}${NC}"
    exit 1
fi

# Sync files if requested
if [[ "$DO_SYNC" == "true" ]]; then
    sync_files
fi

# Collect test cases
ALL_CASES=()
for f in "${SCRIPT_DIR}"/cases/*.sh; do
    [[ -f "$f" ]] || continue
    ALL_CASES+=("$f")
done

# Filter if specific cases requested
RUN_CASES=()
if [[ ${#SELECTED_CASES[@]} -gt 0 ]]; then
    for num in "${SELECTED_CASES[@]}"; do
        for f in "${ALL_CASES[@]}"; do
            if [[ "$(basename "$f")" == "${num}-"* ]]; then
                RUN_CASES+=("$f")
            fi
        done
    done
else
    RUN_CASES=("${ALL_CASES[@]}")
fi

if [[ ${#RUN_CASES[@]} -eq 0 ]]; then
    echo -e "${RED}No test cases found${NC}"
    exit 1
fi

echo "Running ${#RUN_CASES[@]} test case(s)..."

# Run test cases
for case_file in "${RUN_CASES[@]}"; do
    source "$case_file"
done

# Summary
print_summary

# Exit code
[[ $TESTS_FAILED -eq 0 ]] && exit 0 || exit 1
