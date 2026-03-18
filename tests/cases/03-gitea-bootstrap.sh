#!/bin/bash
# Test: Gitea is properly bootstrapped

test_start "03: Gitea Bootstrap"

# Check org exists
org_status=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && docker compose -f generated/docker-compose.yml exec -T gitea \
    curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/api/v1/orgs/launchpad")
assert_eq "$org_status" "200" "Gitea 'launchpad' org exists"

# Check token is in .env
has_token=$(ssh_exec "grep -c '^GITEA_ACCESS_TOKEN=' ${TEST_DEPLOY_DIR}/generated/launchpad/.env" || echo "0")
assert_ne "$has_token" "0" "GITEA_ACCESS_TOKEN is in .env"
