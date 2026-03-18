#!/bin/bash
# Test: K8s cluster registered and heartbeat working

test_start "06: Cluster Registration"

# Check cluster registered via admin API
cluster_status=$(ssh_exec "curl -sk https://${TEST_LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/clusters" 2>/dev/null)
assert_contains "$cluster_status" "local-k3s" "Cluster 'local-k3s' found in API"

# Check heartbeat cron exists
has_cron=$(ssh_exec "crontab -l 2>/dev/null | grep -c k3s-heartbeat" || echo "0")
assert_ne "$has_cron" "0" "Heartbeat cron job exists"

# Check heartbeat script exists
has_script=$(ssh_exec "test -f /usr/local/bin/k3s-heartbeat.sh && echo yes || echo no")
assert_eq "$has_script" "yes" "Heartbeat script installed"
