#!/bin/bash
# Test: --restart flag works

test_start "07: Restart Service"

# Restart API (exit code may be non-zero due to /dev/tty trap, check output)
restart_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart api" 2>&1)
assert_contains "$restart_output" "Restarted" "--restart api outputs 'Restarted'"

# Wait for health
sleep 10
assert_healthy "api"

# Test invalid target
bad_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart badname" 2>&1)
assert_contains "$bad_output" "Unknown service" "--restart badname shows error"
