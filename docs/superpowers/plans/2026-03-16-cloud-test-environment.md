# Cloud Test Environment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy AniLaunchpad on cloud server (10.233.201.133) via AI agent feedback loop (Dev→Ops→QA, max 10 rounds).

**Architecture:** Dev agent modifies deploy scripts to support non-interactive mode, Ops agent runs deployment via SSH, QA agent validates via SSH+curl. Loop until all tests pass.

**Tech Stack:** Bash, Docker Compose, k3s, SSH, curl

**Spec:** `docs/superpowers/specs/2026-03-16-cloud-test-environment-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `deploy/setup.sh` | Add `--config` flag + non-interactive code path |
| Modify | `deploy/scripts/lib/certs.sh:79-117` | Add server IP to self-signed cert SAN |
| Modify | `deploy/scripts/lib/deploy.sh:45-105` | Add `setup_hosts()`, call after nginx healthy |
| Modify | `deploy/scripts/lib/detect.sh:4-46` | Add `envsubst` check in `check_environment()` |
| Create | `deploy/presets/test-server.conf` | Pre-filled config for test server |

---

## Task 1: Add `--config` non-interactive mode to setup.sh

**Files:**
- Modify: `deploy/setup.sh:15-28` (argument parsing)
- Modify: `deploy/setup.sh:35-83` (install/reconfigure case)

- [ ] **Step 1: Add `--config` to argument parser**

In `deploy/setup.sh`, add the `--config` case before the catch-all `*` case, and initialize `CONFIG_FILE=""` before the loop:

```bash
# Before the while loop (after ACTION="install"):
CONFIG_FILE=""

# In the case statement, before the *) line:
        --config)      [[ -z "${2:-}" ]] && { log_error "--config requires a file path"; exit 1; }
                       CONFIG_FILE="$2"; shift 2 ;;
```

After the `while` loop and before the `case` statement, add validation:
```bash
# Validate --config is only used with install/reconfigure
if [[ -n "$CONFIG_FILE" && "$ACTION" != "install" && "$ACTION" != "reconfigure" ]]; then
    log_error "--config is only valid with install/reconfigure"
    exit 1
fi
```

- [ ] **Step 2: Add non-interactive code path in install case**

Replace the `install|reconfigure)` case body with a conditional that checks `CONFIG_FILE`:

```bash
    install|reconfigure)
        # Banner
        print_banner "AniLaunchpad Setup" "$SETUP_VERSION"

        # Load remaining modules
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/render.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"

        if [[ -n "$CONFIG_FILE" ]]; then
            # Non-interactive: load config from file
            [[ ! -f "$CONFIG_FILE" ]] && { log_error "Config file not found: $CONFIG_FILE"; exit 1; }
            source "$CONFIG_FILE"
            source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
            check_environment
        else
            # Interactive: original flow
            select_language
            source "${DEPLOY_DIR}/scripts/lib/interact.sh"
            check_environment
            collect_basic_config
            collect_advanced_config
            show_summary
            if ! ask_confirm "$MSG_DEPLOY_CONFIRM" "Y"; then
                log_warn "Deployment cancelled."
                exit 0
            fi
        fi

        # Phase 4: Generate (shared by both modes)
        if ! load_secrets; then
            generate_secrets
        fi
        render_templates
        save_config
        save_secrets

        # Phase 5: Deploy
        setup_kubernetes
        setup_certificates
        deploy_services
        register_cluster || log_warn "Cluster registration failed — you can retry later with: ./deploy/setup.sh --resume"

        # Done
        show_result
        ;;
```

Key changes:
- `interact.sh` is only sourced in interactive mode (non-interactive doesn't need it)
- Language file loaded from `LANG_CHOICE` in config instead of `select_language`
- All interactive prompts skipped when `CONFIG_FILE` is set

- [ ] **Step 3: Update help text**

Add `--config` to the help output:

```bash
        echo "  --config FILE   Non-interactive install using config file"
```

- [ ] **Step 4: Commit**

```bash
git add deploy/setup.sh
git commit -m "feat: add --config non-interactive mode to setup.sh"
```

---

## Task 2: Add server IP to self-signed cert SAN

**Files:**
- Modify: `deploy/scripts/lib/certs.sh:93-94`

- [ ] **Step 1: Modify SAN construction in `setup_selfsigned_certs()`**

Replace line 94 in `certs.sh`:

```bash
    # Before:
    local san="DNS:*.${DOMAIN},DNS:${DOMAIN},DNS:${LAUNCHPAD_DOMAIN},DNS:${GITEA_DOMAIN},DNS:localhost,IP:127.0.0.1"

    # After:
    local internal_ip
    internal_ip=$(detect_internal_ip 2>/dev/null || echo "")
    local san="DNS:*.${DOMAIN},DNS:${DOMAIN},DNS:${LAUNCHPAD_DOMAIN},DNS:${GITEA_DOMAIN},DNS:localhost,IP:127.0.0.1"
    [[ -n "$internal_ip" ]] && san="${san},IP:${internal_ip}"
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/certs.sh
git commit -m "feat: include server IP in self-signed cert SAN"
```

---

## Task 3: Add `setup_hosts()` and `envsubst` check

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh:98-104` (call setup_hosts after nginx)
- Modify: `deploy/scripts/lib/deploy.sh` (add function)
- Modify: `deploy/scripts/lib/detect.sh:34-40` (add envsubst check)

- [ ] **Step 1: Add `setup_hosts()` function to deploy.sh**

Add after the `deploy_fail()` function (after line 43), before `deploy_services()`:

```bash
setup_hosts() {
    local domains=("$LAUNCHPAD_DOMAIN" "$GITEA_DOMAIN")
    for domain in "${domains[@]}"; do
        grep -qF "$domain" /etc/hosts || echo "127.0.0.1  $domain" >> /etc/hosts || log_warn "Could not update /etc/hosts (run as root)"
    done
    log_ok "Hosts file configured"
}
```

- [ ] **Step 2: Call `setup_hosts()` in `deploy_services()` after nginx healthy**

In `deploy_services()`, between `wait_for_healthy nginx 30` (line 101) and `progress_done` (line 103), add:

```bash
    setup_hosts
```

- [ ] **Step 3: Call `setup_hosts()` in `resume_deploy()` before show_result**

In `resume_deploy()`, after `bootstrap_gitea` (line 317) and before `register_cluster` (line 320), add:

```bash
    setup_hosts
```

- [ ] **Step 4: Add `envsubst` check in `check_environment()`**

In `detect.sh`, between the jq check block and the `if [[ "$errors" -gt 0 ]]` block (between lines 41 and 42), add:

```bash
    # envsubst (required for template rendering)
    if ! command -v envsubst &>/dev/null; then
        log_error "envsubst is not installed (install gettext-base)"
        ((errors++))
    else
        log_ok "envsubst: available"
    fi
```

- [ ] **Step 5: Commit**

```bash
git add deploy/scripts/lib/deploy.sh deploy/scripts/lib/detect.sh
git commit -m "feat: add setup_hosts() and envsubst pre-flight check"
```

---

## Task 4: Create test server preset config

**Files:**
- Create: `deploy/presets/test-server.conf`

- [ ] **Step 1: Create presets directory and config file**

```bash
mkdir -p deploy/presets
```

Write `deploy/presets/test-server.conf`:

```bash
# AniLaunchpad Test Server Configuration
# Server: 10.233.201.133 (4C/7.4G, Debian 12, k3s)
# Usage: ./deploy/setup.sh --config deploy/presets/test-server.conf

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

- [ ] **Step 2: Commit**

```bash
git add deploy/presets/test-server.conf
git commit -m "feat: add test server preset config"
```

---

## Task 5: Sync code to remote server

- [ ] **Step 1: Sync all deploy files to remote**

```bash
scp -r deploy/setup.sh deploy/versions.conf deploy/.gitignore root@10.233.201.133:/root/workspace/launchpad-deploy/deploy/
scp deploy/scripts/lib/*.sh root@10.233.201.133:/root/workspace/launchpad-deploy/deploy/scripts/lib/
scp deploy/scripts/lang/*.sh root@10.233.201.133:/root/workspace/launchpad-deploy/deploy/scripts/lang/
scp deploy/templates/* root@10.233.201.133:/root/workspace/launchpad-deploy/deploy/templates/
ssh root@10.233.201.133 "mkdir -p /root/workspace/launchpad-deploy/deploy/presets"
scp deploy/presets/test-server.conf root@10.233.201.133:/root/workspace/launchpad-deploy/deploy/presets/
```

- [ ] **Step 2: Verify no macOS references remain**

```bash
ssh root@10.233.201.133 "grep -rn 'Darwin\|macOS\|host\.docker\.internal\|/Users/' /root/workspace/launchpad-deploy/deploy/ || echo 'Clean'"
```

Expected: `Clean`

---

## Task 6: Ops — Run deployment on remote server

- [ ] **Step 1: Clean any previous deployment state**

```bash
ssh root@10.233.201.133 "cd /root/workspace/launchpad-deploy && rm -rf deploy/generated"
```

- [ ] **Step 2: Execute non-interactive deployment**

```bash
ssh root@10.233.201.133 "cd /root/workspace/launchpad-deploy && bash deploy/setup.sh --config deploy/presets/test-server.conf --verbose"
```

Timeout: 10 minutes. Capture full stdout/stderr.

- [ ] **Step 3: Verify all containers healthy**

```bash
ssh root@10.233.201.133 "docker compose -f /root/workspace/launchpad-deploy/deploy/generated/docker-compose.yml ps"
```

Expected: all 11 services show "healthy" or "running" (cron/backup-worker have no healthcheck).

---

## Task 7: QA — Run test suite

All tests run via SSH on the remote server. Output structured report.

- [ ] **Step 1: Health checks**

```bash
# Nginx health
ssh root@10.233.201.133 "curl -sk -o /dev/null -w '%{http_code}' https://launchpad.corp.testclaw.com/nginx-health"
# Expected: 200

# API health
ssh root@10.233.201.133 "curl -sk -o /dev/null -w '%{http_code}' https://launchpad.corp.testclaw.com/api/health"
# Expected: non-5xx (200 or 401)

# Gitea home
ssh root@10.233.201.133 "curl -sk -o /dev/null -w '%{http_code}' https://launchpad-gitea.corp.testclaw.com/"
# Expected: 200
```

- [ ] **Step 2: API functionality**

```bash
# Login (single SSH session to avoid quoting issues with auto-generated passwords)
ssh root@10.233.201.133 'ADMIN_PASS=$(grep ADMIN_PASSWORD /root/workspace/launchpad-deploy/deploy/generated/.secrets | cut -d"\"" -f2) && curl -sk -X POST https://launchpad.corp.testclaw.com/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"admin@testclaw.com\",\"password\":\"$ADMIN_PASS\"}"'
# Expected: JSON with token

# Gitea org exists
ssh root@10.233.201.133 "curl -sk -o /dev/null -w '%{http_code}' https://launchpad-gitea.corp.testclaw.com/api/v1/orgs/launchpad"
# Expected: 200

# Gitea repo exists
ssh root@10.233.201.133 "curl -sk -o /dev/null -w '%{http_code}' https://launchpad-gitea.corp.testclaw.com/api/v1/repos/launchpad/ani-code"
# Expected: 200
```

- [ ] **Step 3: Infrastructure checks**

```bash
# PostgreSQL schemas (6 expected)
ssh root@10.233.201.133 "docker compose -f /root/workspace/launchpad-deploy/deploy/generated/docker-compose.yml exec -T postgres psql -U postgres -d launchpad -t -c \"SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'launchpad_%' ORDER BY 1\""
# Expected: launchpad_billing, launchpad_events, launchpad_gateway, launchpad_main, launchpad_monitoring, launchpad_stats

# Redis ping
ssh root@10.233.201.133 "docker compose -f /root/workspace/launchpad-deploy/deploy/generated/docker-compose.yml exec -T redis redis-cli -a \$(grep REDIS_PASSWORD /root/workspace/launchpad-deploy/deploy/generated/.secrets | cut -d'\"' -f2) ping"
# Expected: PONG
```

- [ ] **Step 4: Output structured report**

Format:
```
ROUND: 1
STATUS: PASS | FAIL
FAILURES:
  - [test_name] expected: X, got: Y, error: ...
PASSES:
  - [test_name] ok
```

---

## Task 8: Feedback loop (rounds 2-10)

This task is conditional — only executes if Task 7 reports failures.

- [ ] **Step 1: Dev agent analyzes QA report**

Read the failure report. For each failure:
- Determine root cause (code bug, config issue, infra problem)
- Apply targeted fix to the relevant file
- Sync changed files to remote server

- [ ] **Step 2: Ops agent re-deploys**

If template/config files changed:
```bash
ssh root@10.233.201.133 "cd /root/workspace/launchpad-deploy && bash deploy/setup.sh --config deploy/presets/test-server.conf --verbose"
```

If only a service needs restart:
```bash
ssh root@10.233.201.133 "docker compose -f /root/workspace/launchpad-deploy/deploy/generated/docker-compose.yml up -d <service>"
```

- [ ] **Step 3: QA agent re-runs full test suite**

Same as Task 7. Increment round number. If all pass → done. If failures remain and round < 10 → repeat from Step 1.

- [ ] **Step 4: Loop termination**

- All pass → output final success report to user
- Round 10 reached with failures → output summary of all rounds, list never-passed tests, suggest manual investigation
- Unrecoverable error (image pull fail, disk full) → stop immediately, report to user
