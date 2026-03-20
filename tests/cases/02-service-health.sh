#!/bin/bash
# Test: All services are healthy

test_start "02: Service Health"

for svc in postgres redis gitea api ui router gateway nginx; do
    assert_healthy "$svc"
done

# Verify NODE_TLS_REJECT_UNAUTHORIZED is set for services that make HTTPS calls
for svc in api gateway; do
    tls_env=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && awk '/^  ${svc}:/{found=1} found && /NODE_TLS_REJECT_UNAUTHORIZED/{print; exit} /^  [a-z]/ && !/^  ${svc}:/ && found{exit}' deploy/generated/docker-compose.yml" || echo "")
    if [[ -n "$tls_env" ]]; then
        test_pass "${svc} has NODE_TLS_REJECT_UNAUTHORIZED"
    else
        test_fail "${svc} has NODE_TLS_REJECT_UNAUTHORIZED" "missing from compose environment"
    fi
done
