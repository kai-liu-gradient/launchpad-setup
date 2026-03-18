#!/bin/bash
# Test: --restart flag works

test_start "07: Restart Service"

# Restart API
restart_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart api" 2>&1)
exit_code=$?
assert_eq "$exit_code" "0" "--restart api exits 0"

# Wait for health
sleep 10
assert_healthy "api"

# Test invalid target
bad_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart badname" 2>&1)
bad_code=$?
assert_ne "$bad_code" "0" "--restart badname exits non-zero"
