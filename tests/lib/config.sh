#!/bin/bash
# E2E test configuration — override via environment variables

TEST_SERVER="${TEST_SERVER:-10.233.201.133}"
TEST_SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
TEST_SSH="ssh ${TEST_SSH_OPTS} root@${TEST_SERVER}"
TEST_SCP="scp ${TEST_SSH_OPTS}"

TEST_DOMAIN="${TEST_DOMAIN:-testclaw.com}"
TEST_SUBDOMAIN="${TEST_SUBDOMAIN:-corp}"
TEST_REGISTRY="${TEST_REGISTRY:-swr.ap-southeast-1.myhuaweicloud.com/ghisha}"
TEST_VERSION="${TEST_VERSION:-2.0.3}"
TEST_DEPLOY_DIR="${TEST_DEPLOY_DIR:-~/workspace/launchpad-deploy}"

TEST_LAUNCHPAD_DOMAIN="launchpad.${TEST_SUBDOMAIN}.${TEST_DOMAIN}"
TEST_GITEA_DOMAIN="launchpad-gitea.${TEST_SUBDOMAIN}.${TEST_DOMAIN}"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0
FAILED_TESTS=()
