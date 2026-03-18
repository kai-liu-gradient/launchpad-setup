#!/bin/bash
# Test: All web endpoints respond

test_start "05: Endpoints"

assert_http "https://${TEST_LAUNCHPAD_DOMAIN}" "200" "Dashboard returns 200"
assert_http "https://${TEST_GITEA_DOMAIN}" "200" "Gitea returns 200"
assert_http "https://${TEST_LAUNCHPAD_DOMAIN}/admin" "200" "Admin returns 200"
