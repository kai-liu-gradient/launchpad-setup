#!/bin/bash
# Test: Pod-internal connectivity to gitea and gateway domains (as node user)
# Validates: CoreDNS resolution + ingress routing + CA trust chain under non-root user

test_start "09: Pod Connectivity (node user)"

POD_NAME="test-conn-$$"
KUBECONFIG_FLAG="--kubeconfig=/etc/rancher/k3s/k3s.yaml"

# Create test pod with node:24-alpine (has node user uid=1000)
# Install curl inside since busybox wget doesn't support --ca-certificate
ssh_exec "kubectl ${KUBECONFIG_FLAG} run ${POD_NAME} --image=node:24-alpine --restart=Never --command -- sleep 300" >/dev/null 2>&1

# Wait for pod ready
pod_ready=$(ssh_exec "kubectl ${KUBECONFIG_FLAG} wait --for=condition=ready pod/${POD_NAME} --timeout=120s 2>&1" || echo "timeout")
assert_contains "$pod_ready" "condition met" "Test pod is ready"

# Install curl in the pod (alpine's busybox wget has limited SSL support)
ssh_exec "kubectl ${KUBECONFIG_FLAG} exec ${POD_NAME} -- apk add -q --no-cache curl" >/dev/null 2>&1

# Test gitea HTTPS as node user (validates DNS + routing + CA trust)
gitea_result=$(ssh_exec "kubectl ${KUBECONFIG_FLAG} exec ${POD_NAME} -- su -s /bin/sh node -c 'curl -sf https://${TEST_GITEA_DOMAIN}/api/v1/version'" 2>/dev/null || echo "FAIL")
assert_contains "$gitea_result" "version" "Gitea HTTPS reachable from pod (node user)"

# Test gateway HTTPS as node user
gateway_code=$(ssh_exec "kubectl ${KUBECONFIG_FLAG} exec ${POD_NAME} -- su -s /bin/sh node -c 'curl -so /dev/null -w \"%{http_code}\" https://${TEST_LAUNCHPAD_DOMAIN}/gatewayproxy/'" 2>/dev/null || echo "000")
assert_ne "$gateway_code" "000" "Gateway HTTPS reachable from pod (node user)"

# Cleanup
ssh_exec "kubectl ${KUBECONFIG_FLAG} delete pod ${POD_NAME} --grace-period=0 --force" >/dev/null 2>&1
