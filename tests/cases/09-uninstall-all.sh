#!/bin/bash
# Test: --uninstall-all cleans everything

test_start "09: Uninstall All"

output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && echo 'Y' | ./setup.sh --uninstall-all" 2>&1)
exit_code=$?

assert_eq "$exit_code" "0" "--uninstall-all exits 0"

# No Docker containers
containers=$(ssh_exec "docker ps -q --filter 'label=com.docker.compose.project'" 2>/dev/null | wc -l | tr -d ' ')
assert_eq "$containers" "0" "No Docker containers running"

# No k3s
k3s_running=$(ssh_exec "systemctl is-active k3s 2>/dev/null || echo inactive")
assert_eq "$k3s_running" "inactive" "k3s is not running"

# No generated directory
gen_exists=$(ssh_exec "test -d ${TEST_DEPLOY_DIR}/generated && echo yes || echo no")
assert_eq "$gen_exists" "no" "generated/ directory removed"
