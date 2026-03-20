#!/bin/bash
# Test: K8s cluster registered and heartbeat working

test_start "06: Cluster Registration"

# Check k3s is running and has a ready node
k3s_nodes=$(ssh_exec "kubectl get nodes --kubeconfig=/etc/rancher/k3s/k3s.yaml --no-headers 2>/dev/null" || echo "")
assert_contains "$k3s_nodes" "Ready" "K3s node is Ready"

# Check heartbeat cron exists
has_cron=$(ssh_exec "crontab -l 2>/dev/null | grep -c k3s-heartbeat" || echo "0")
assert_ne "$has_cron" "0" "Heartbeat cron job exists"

# Check heartbeat script exists
has_script=$(ssh_exec "test -f /usr/local/bin/k3s-heartbeat.sh && echo yes || echo no")
assert_eq "$has_script" "yes" "Heartbeat script installed"

# Check dnsmasq routes *.DOMAIN to host IP (so gateway reaches nginx:443)
dnsmasq_conf=$(ssh_exec "cat /etc/dnsmasq.d/launchpad.conf 2>/dev/null" || echo "")
assert_contains "$dnsmasq_conf" "address=/${TEST_DOMAIN}/" "dnsmasq wildcard DNS configured"

# Verify dnsmasq resolves *.DOMAIN to the host IP
expected_ip=$(ssh_exec "ip addr show eth0 2>/dev/null | grep 'inet ' | awk '{print \$2}' | cut -d/ -f1" || echo "")
dns_ip=$(ssh_exec "nslookup test-check.${TEST_DOMAIN} 127.0.0.1 2>/dev/null | grep -A1 'Name:' | grep Address | tail -1 | awk '{print \$2}'" || echo "")
assert_eq "$dns_ip" "$expected_ip" "dnsmasq resolves *.${TEST_DOMAIN} to host IP"
