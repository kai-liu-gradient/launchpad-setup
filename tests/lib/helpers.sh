#!/bin/bash
# E2E test helper functions

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BOLD='\033[1m'; NC='\033[0m'

ssh_exec() {
    $TEST_SSH "$@" 2>/dev/null
}

sync_files() {
    local local_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
    echo -e "${YELLOW}Syncing files to ${TEST_SERVER}...${NC}"
    ssh_exec "mkdir -p ${TEST_DEPLOY_DIR}/deploy ${TEST_DEPLOY_DIR}/files" >/dev/null 2>&1 || true
    $TEST_SCP -r "${local_dir}/deploy/"* "root@${TEST_SERVER}:${TEST_DEPLOY_DIR}/deploy/" >/dev/null
    $TEST_SCP -r "${local_dir}/files/"* "root@${TEST_SERVER}:${TEST_DEPLOY_DIR}/files/" >/dev/null
    echo -e "${GREEN}Files synced${NC}"
}

test_start() {
    local name="$1"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo ""
    echo -e "${BOLD}━━━ TEST: ${name} ━━━${NC}"
}

test_pass() {
    local name="$1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "  ${GREEN}✓ PASS${NC}: ${name}"
}

test_fail() {
    local name="$1" reason="${2:-}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    FAILED_TESTS+=("$name")
    echo -e "  ${RED}✗ FAIL${NC}: ${name}"
    [[ -n "$reason" ]] && echo -e "    ${RED}Reason: ${reason}${NC}"
}

assert_eq() {
    local actual="$1" expected="$2" msg="$3"
    if [[ "$actual" == "$expected" ]]; then
        test_pass "$msg"
    else
        test_fail "$msg" "expected='${expected}' actual='${actual}'"
    fi
}

assert_ne() {
    local actual="$1" unexpected="$2" msg="$3"
    if [[ "$actual" != "$unexpected" ]]; then
        test_pass "$msg"
    else
        test_fail "$msg" "got unexpected value='${unexpected}'"
    fi
}

assert_contains() {
    local haystack="$1" needle="$2" msg="$3"
    if echo "$haystack" | grep -q "$needle"; then
        test_pass "$msg"
    else
        test_fail "$msg" "output does not contain '${needle}'"
    fi
}

assert_http() {
    local url="$1" expected_code="$2" msg="${3:-HTTP ${url} returns ${expected_code}}"
    local actual_code
    actual_code=$(ssh_exec "curl -sk -o /dev/null -w '%{http_code}' --max-time 10 '${url}'" 2>/dev/null || echo "000")
    assert_eq "$actual_code" "$expected_code" "$msg"
}

assert_healthy() {
    local service="$1"
    local status
    status=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && docker compose -f deploy/generated/docker-compose.yml ps --format json ${service}" 2>/dev/null \
        | grep -o '"Health":"[^"]*"' | cut -d'"' -f4)
    if [[ "$status" == "healthy" ]]; then
        test_pass "${service} is healthy"
    else
        test_fail "${service} is healthy" "status='${status}'"
    fi
}

print_summary() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
    echo -e "${BOLD}  Test Summary${NC}"
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
    echo -e "  Total:  ${TESTS_TOTAL}"
    echo -e "  ${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "  ${RED}Failed: ${TESTS_FAILED}${NC}"
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo ""
        echo -e "  ${RED}Failed tests:${NC}"
        for t in "${FAILED_TESTS[@]}"; do
            echo -e "    ${RED}✗${NC} $t"
        done
    fi
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
}
