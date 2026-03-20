#!/bin/bash
# Test: Re-running install is idempotent

test_start "08: Idempotent Re-deploy"

output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --config /tmp/test-setup.conf" 2>&1)

assert_contains "$output" "complete" "Re-install output contains 'complete'"

# All services still healthy
for svc in api ui router gateway nginx gitea; do
    assert_healthy "$svc"
done
