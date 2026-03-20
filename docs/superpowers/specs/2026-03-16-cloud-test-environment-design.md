# Cloud Test Environment Design

**Date:** 2026-03-16
**Status:** Approved
**Server:** root@10.233.201.133 (4C/7.4G, Debian 12, k3s v1.34.5)

## Goal

Deploy AniLaunchpad on a cloud Linux server using AI agents (dev/ops/QA), verifying both the deployment pipeline and product functionality. All macOS-specific code has already been removed.

## Architecture

```
Local (macOS)
├── Dev Agent (worktree)  — code changes to setup.sh + templates
├── Ops Agent             — SSH deploy + monitor
└── QA Agent              — SSH curl/API validation + test report

Cloud Server (10.233.201.133)
├── /root/workspace/launchpad-deploy/deploy/   ← scripts + templates
├── /root/workspace/launchpad-deploy/deploy/generated/  ← output
├── k3s (running, kubeconfig: /etc/rancher/k3s/k3s.yaml)
└── Docker Compose services (11 containers)
```

**Access model:** Domain-based via hosts file.
- Server `/etc/hosts`: `127.0.0.1 launchpad.corp.testclaw.com launchpad-gitea.corp.testclaw.com`
- User's local `/etc/hosts`: `10.233.201.133 launchpad.corp.testclaw.com launchpad-gitea.corp.testclaw.com`
- SSL: self-signed with server IP `10.233.201.133` in SAN

## Changes Required

### 1. Non-interactive mode for setup.sh

**Problem:** `setup.sh` requires interactive input (domain, SSL mode, DB mode, etc.). AI agents cannot interact with terminal prompts.

**Solution:** Add `--config <file>` flag that sources a config file and skips all interactive prompts.

**File:** `deploy/setup.sh`

Argument parsing: `--config` takes a file argument (`--config <file>`), uses `shift 2`, and validates the file exists. It is only valid with the `install` action and can be combined with `--verbose`.

```bash
--config) [[ -z "${2:-}" ]] && { log_error "--config requires a file path"; exit 1; }
          CONFIG_FILE="$2"; shift 2 ;;
```

When `CONFIG_FILE` is set:
- Source the config file (same format as `.setup.conf`)
- Source the secrets file if it exists at `generated/.secrets` (same format as `.secrets`)
- Skip: `select_language`, `collect_basic_config`, `collect_advanced_config`, `show_summary`, `ask_confirm`
- Execute: load language → `check_environment` → `if ! load_secrets; then generate_secrets; fi` → `render_templates` → `save_config` → `save_secrets` → `setup_kubernetes` → `setup_certificates` → `deploy_services` → `register_cluster` → `show_result`

**Config file for test server** (`deploy/presets/test-server.conf`):
```bash
LANG_CHOICE="en"
DOMAIN="testclaw.com"
SUBDOMAIN="corp"
LAUNCHPAD_DOMAIN="launchpad.corp.testclaw.com"
GITEA_DOMAIN="launchpad-gitea.corp.testclaw.com"
IMAGE_REGISTRY="swr.ap-southeast-1.myhuaweicloud.com/ghisha"
IMAGE_VERSION="1.18.7"
IMAGE_VERSION_API="1.18.7"
IMAGE_VERSION_UI="1.18.6"
IMAGE_VERSION_ROUTER="1.16.2"
IMAGE_VERSION_GATEWAY="1.18.7"
IMAGE_VERSION_GITEA="1.25-rootless"
SSL_MODE="selfsigned"
DB_MODE="builtin"
REDIS_HOST="redis"
REDIS_PORT="6379"
K8S_MODE="builtin"
K8S_KUBECONFIG_PATH="/etc/rancher/k3s/k3s.yaml"
STORAGE_CLASS="local-path"
DEFAULT_BACKEND="localhost"
ADMIN_EMAIL="admin@testclaw.com"
API_REPLICAS="1"
ROUTER_REPLICAS="1"
GATEWAY_REPLICAS="1"
DB_CONNECTION_LIMIT="10"
```

### 2. Self-signed cert SAN includes server IP

**Problem:** If someone accesses via IP, the cert won't match. Also useful as a fallback.

**File:** `deploy/scripts/lib/certs.sh` — `setup_selfsigned_certs()`

**Change:** Add the detected internal IP to the SAN list:
```bash
local internal_ip
internal_ip=$(detect_internal_ip 2>/dev/null || echo "")
local san="DNS:*.${DOMAIN},DNS:${DOMAIN},DNS:${LAUNCHPAD_DOMAIN},DNS:${GITEA_DOMAIN},DNS:localhost,IP:127.0.0.1"
[[ -n "$internal_ip" ]] && san="${san},IP:${internal_ip}"
```

### 3. Server hosts file setup

**File:** `deploy/scripts/lib/deploy.sh` — add `setup_hosts()` function

After deployment completes, idempotently append entries to `/etc/hosts` using domain variables:

```bash
setup_hosts() {
    local domains=("$LAUNCHPAD_DOMAIN" "$GITEA_DOMAIN")
    for domain in "${domains[@]}"; do
        grep -qF "$domain" /etc/hosts || echo "127.0.0.1  $domain" >> /etc/hosts
    done
    log_ok "Hosts file configured"
}
```

Called from `deploy_services()` after nginx is healthy (between health check and `progress_done`). Also called in `resume_deploy()` at the end.

**Note:** `envsubst` (from `gettext-base` package) must be available on the server. Dev agent should add a check in `check_environment()` or ensure it's installed.

## Agent Workflow — Feedback Loop

The three agents form a closed loop that iterates until all tests pass or the maximum of **10 rounds** is reached.

```
Round N (max 10)
┌→ Dev Agent ──→ Ops Agent ──→ QA Agent ─┐
│                                         │
│   ┌─── All pass? ───Yes───→ DONE        │
│   │                                     │
│   └─── Has failures ───────────────────┘
│         (structured report)
└─────────────────────────────────────────┘
```

### Round 1: Initial deployment

**Dev Agent** (worktree isolation):
1. Add `--config <file>` non-interactive mode to `setup.sh`
2. Add server IP to self-signed cert SAN in `certs.sh`
3. Add `setup_hosts()` to `deploy.sh`
4. Create `deploy/presets/test-server.conf`
5. Sync updated code to remote server via scp

**Ops Agent**:
1. Execute `./deploy/setup.sh --config deploy/presets/test-server.conf` on remote server
2. Monitor deployment progress, capture logs on failure
3. Verify all 11 containers reach healthy status:
   - postgres, redis (infrastructure)
   - api, ui, router, gateway (application)
   - gitea, cron, backup-worker, heartbeat, nginx (supporting)
4. Report deployed service versions and health status

**QA Agent**:
Tests (all via SSH + curl on the server):

*Health checks:*
- `curl -sk https://launchpad.corp.testclaw.com/nginx-health` → 200
- `curl -sk https://launchpad.corp.testclaw.com/api/health` → non-5xx
- `curl -sk https://launchpad-gitea.corp.testclaw.com/` → 200

*API functionality:*
- Login with admin credentials → get JWT token
- List projects → 200
- Check Gitea org `launchpad` exists
- Check Gitea repo `ani-code` exists

*Infrastructure:*
- PostgreSQL: 6 schemas exist (`launchpad_main`, `_monitoring`, `_events`, `_billing`, `_stats`, `_gateway`)
- Redis: ping responds
- K3s: cluster registered (check via admin API)

*Output:* Structured test report in this format:
```
ROUND: N
STATUS: PASS | FAIL
FAILURES:
  - [test_name] expected: X, got: Y, error: ...
  - ...
PASSES:
  - [test_name] ok
  - ...
```

### Round 2-10: Fix loop

If QA report has failures:

**Dev Agent** receives the QA report and:
1. Analyzes each failure — determine if it's a code bug, config issue, or infra problem
2. Applies targeted fix to the relevant file(s)
3. Syncs updated code to remote server

**Ops Agent**:
1. If code changed: re-deploy affected services (`docker compose up -d <service>`) or full re-deploy if config templates changed
2. Wait for services to stabilize

**QA Agent**:
1. Re-runs full test suite
2. Outputs updated report with round number
3. If all pass → loop ends
4. If still failures → next round

### Loop termination

- **Success:** All QA tests pass → output final report + notify user
- **Max rounds reached (10):** Output summary of all rounds showing which tests never passed, remaining failure details, and suggested manual investigation steps
- **Unrecoverable error:** Ops deployment completely fails (e.g., image pull error, disk full) → stop loop, report to user

## Resource Estimates

| Service | Memory | Notes |
|---------|--------|-------|
| K3s | ~750MB | Already running |
| PostgreSQL | ~300MB | |
| Redis | ~300MB | 256MB max configured |
| API | ~512MB-1GB | Largest consumer |
| UI | ~300MB | |
| Router | ~256MB | |
| Gateway | ~256MB | |
| Gitea | ~256MB | |
| Cron + Backup | ~400MB | Share API image |
| Nginx + Heartbeat | ~128MB | |
| **Total** | **~3.5-5GB** | Fits in 7.4GB with headroom |

## Success Criteria

1. `setup.sh --config` runs to completion without interactive prompts
2. All 11 containers reach healthy status
3. QA health checks all pass
4. Admin can log in via browser at `https://launchpad.corp.testclaw.com`
5. Gitea accessible at `https://launchpad-gitea.corp.testclaw.com`
6. Feedback loop converges (all tests pass) within 10 rounds

## Out of Scope

- CI/CD pipeline integration
- Persistent/repeatable deployment (future improvement)
- Load testing
- Production hardening
