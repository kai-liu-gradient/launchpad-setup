#!/bin/bash
# Test: Clean install from scratch

test_start "01: Clean Install"

# Uninstall everything first
ssh_exec "cd ${TEST_DEPLOY_DIR} && echo 'Y' | ./setup.sh --uninstall-all" >/dev/null 2>&1 || true

# Generate a test config (unquoted CONF so local variables expand)
ssh_exec "cat > /tmp/test-setup.conf << CONF
LANG_CHOICE=en
DOMAIN=${TEST_DOMAIN}
SUBDOMAIN=${TEST_SUBDOMAIN}
LAUNCHPAD_DOMAIN=launchpad.${TEST_SUBDOMAIN}.${TEST_DOMAIN}
GITEA_DOMAIN=launchpad-gitea.${TEST_SUBDOMAIN}.${TEST_DOMAIN}
IMAGE_REGISTRY=${TEST_REGISTRY}
IMAGE_VERSION=${TEST_VERSION}
IMAGE_VERSION_API=${TEST_VERSION}
IMAGE_VERSION_UI=${TEST_VERSION}
IMAGE_VERSION_ROUTER=${TEST_VERSION}
IMAGE_VERSION_GATEWAY=${TEST_VERSION}
IMAGE_VERSION_GITEA=1.25-rootless
SSL_MODE=selfsigned
DB_MODE=builtin
DB_HOST=postgres
DB_PORT=5432
REDIS_HOST=redis
REDIS_PORT=6379
K8S_MODE=builtin
K8S_KUBECONFIG_PATH=/etc/rancher/k3s/k3s.yaml
STORAGE_CLASS=local-path
DEFAULT_BACKEND=localhost
ADMIN_EMAIL=admin@${TEST_DOMAIN}
ADMIN_PASSWORD=TestPassword123
CONF"

# Run install
install_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --config /tmp/test-setup.conf" 2>&1)
exit_code=$?

assert_eq "$exit_code" "0" "Install exits with code 0"
assert_contains "$install_output" "complete" "Install output contains 'complete'"
