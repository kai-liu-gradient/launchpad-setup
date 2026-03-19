# Reliable Service Restart After .env Changes

**Date:** 2026-03-19
**Status:** Approved

## Problem

1. `docker compose restart` does not re-read `.env` files — containers keep stale env vars
2. `docker compose up -d --force-recreate` re-reads `.env` but changes container IPs, causing Nginx upstream DNS cache to point at old IPs → 502 errors
3. No health check after restart — callers don't know if services are ready

## Solution

Add a `recreate_services()` function in `deploy.sh` that:
1. Force-recreates specified services (re-reads .env)
2. Restarts nginx (refreshes DNS cache)
3. Waits for recreated services to become healthy

All `--setup-*` commands that modify `.env` use this function. `--restart` keeps its fast restart semantics.

## Changes

### 1. `deploy/scripts/lib/deploy.sh` — new function

```bash
recreate_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    docker compose -f "$compose_file" up -d --force-recreate "$@"
    docker compose -f "$compose_file" restart nginx
    for svc in "$@"; do
        wait_for_healthy "$svc" 60 || true
    done
    log_done "Recreated: $* (nginx restarted)"
}
```

Place after `restart_service()`.

### 2. `deploy/scripts/lib/deploy.sh` — fix `upgrade_services()`

Change `docker compose up -d` to `docker compose up -d --force-recreate`, add nginx restart and health checks.

### 3. `deploy/setup.sh` — simplify `--setup-telegram`

Replace inline docker compose commands with `recreate_services api gateway`.

## Impact

- All future `--setup-*` commands just call `recreate_services <affected-services>`
- `--restart` unchanged (fast restart, no .env reload)
- `--upgrade` now properly reloads env vars
