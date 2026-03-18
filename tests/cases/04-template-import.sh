#!/bin/bash
# Test: Templates imported and repos pushed

test_start "04: Template Import"

# Check repos exist in Gitea
for repo in ani-code nodejs-helloworld test-openclaw; do
    repo_status=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && docker compose -f generated/docker-compose.yml exec -T gitea \
        curl -s -o /dev/null -w '%{http_code}' \
        -H \"Authorization: token \$(grep GITEA_ACCESS_TOKEN generated/launchpad/.env | cut -d= -f2)\" \
        http://localhost:3000/api/v1/repos/launchpad/${repo}")
    assert_eq "$repo_status" "200" "Repo launchpad/${repo} exists"
done

# Check repos have commits
for repo in ani-code nodejs-helloworld test-openclaw; do
    has_commits=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && docker compose -f generated/docker-compose.yml exec -T gitea \
        curl -s \
        -H \"Authorization: token \$(grep GITEA_ACCESS_TOKEN generated/launchpad/.env | cut -d= -f2)\" \
        http://localhost:3000/api/v1/repos/launchpad/${repo}/commits?limit=1" | grep -c '"sha"' || echo "0")
    assert_ne "$has_commits" "0" "Repo launchpad/${repo} has commits"
done
