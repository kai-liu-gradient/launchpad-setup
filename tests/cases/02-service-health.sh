#!/bin/bash
# Test: All services are healthy

test_start "02: Service Health"

for svc in postgres redis gitea api ui router gateway nginx; do
    assert_healthy "$svc"
done
