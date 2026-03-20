# Template Import & Full Uninstall Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add template/repo auto-import during setup, full environment uninstall, granular CLI flags, and an E2E test suite.

**Architecture:** Extends existing bash module system. `import_templates()` added to `deploy.sh`, called after API is healthy (not after `bootstrap_gitea()` — API must be running for YAML import). `uninstall_all()` added alongside existing `uninstall_services()`. Granular CLI flags dispatch to existing functions via `load_saved_config` helper. E2E tests run remotely via SSH.

**Tech Stack:** Bash, Docker Compose, Gitea API (via gitea container curl), Launchpad Admin API port 6804 (via api container wget), git, SSH

**Status:** Implemented and tested. 37/37 E2E tests passing.

### Post-Implementation Fixes (applied during E2E testing)

The following critical bugs were discovered and fixed during testing. See `memory/project_deploy_pitfalls.md` for the full list.

1. **Git push subshell failures (exit 128):**
   - `local` keyword is invalid in subshells — removed, use plain vars
   - Git commit requires committer identity — added `GIT_AUTHOR/COMMITTER_NAME/EMAIL` env vars
   - EXIT trap inherited from common.sh — added `trap - EXIT` at subshell start

2. **Template YAML import returning false success (401 Unauthorized silently swallowed):**
   - API port 6802 requires JWT auth for `/api/v1/templates/from-yaml`
   - Switched to admin port 6804 `/api/templates/yaml/create` (no auth, port-based access)
   - Fixed error detection: check `"success":false` / `"valid":false` instead of grep for "error"
   - Fixed `wget` stderr swallowed by `2>/dev/null` — now uses `2>&1 || echo "WGET_FAILED"`

3. **`import_templates` called before API started:**
   - Moved from after `bootstrap_gitea()` to after API is healthy in `deploy_services()`

4. **Script exit on grep no-match (`set -eo pipefail`):**
   - `grep | tail | cut` pipeline returns exit 1 when grep finds nothing — added `|| true`
   - Affected `render.sh` when `.env` exists but has no `GITEA_ACCESS_TOKEN` line

5. **`kubectl wait` fails on non-existent resources:**
   - Added polling loop in `wait_for_traefik()` to wait for pod to appear before `kubectl wait`

6. **`uninstall_all` confirmation fails via SSH:**
   - `ask_confirm` reads from `/dev/tty` which doesn't exist in SSH — added stdin fallback

7. **Test config used wrong image versions:**
   - Router (1.16.2) and Gateway (1.18.7) have different versions from API/UI (2.0.3)

8. **`--import-templates` standalone didn't load GITEA_ACCESS_TOKEN:**
   - Token is in `.env` not `.secrets` — added explicit grep from `.env` in handler

**Spec:** `docs/superpowers/specs/2026-03-18-setup-enhancements-design.md`

---

## Chunk 1: Template & Repo Import

### Task 1: Remove ani-code creation from bootstrap_gitea

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh:174-179`

- [ ] **Step 1: Remove the ani-code repo creation block**

In `deploy/scripts/lib/deploy.sh`, delete lines 174-179 (the `# Create ani-code repo` block) from `bootstrap_gitea()`. Also update the log message on line 195.

Before:
```bash
        # Create ani-code repo
        docker compose -f "$compose_file" exec -T gitea \
            curl -s -X POST "http://localhost:3000/api/v1/orgs/launchpad/repos" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            -H "Content-Type: application/json" \
            -d '{"name":"ani-code","auto_init":true,"default_branch":"main"}' >/dev/null

        # Write token back to .env (API reads it on next start)
```

After:
```bash
        # Write token back to .env (API reads it on next start)
```

Also change the log line from:
```bash
        log_ok "Gitea bootstrapped (admin: $gitea_admin, org: launchpad, repo: ani-code)"
```
To:
```bash
        log_ok "Gitea bootstrapped (admin: $gitea_admin, org: launchpad)"
```

- [ ] **Step 2: Verify bootstrap_gitea still works**

Run on remote server:
```bash
ssh root@10.233.201.133 "cd ~/workspace/launchpad-deploy && ./setup.sh --config presets/test-server.conf"
```
Expected: Gitea bootstraps successfully, org `launchpad` created, but no `ani-code` repo yet.

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "refactor: remove ani-code repo creation from bootstrap_gitea

Repo creation will be handled by import_templates() instead."
```

---

### Task 2: Add i18n messages for template import

**Files:**
- Modify: `deploy/scripts/lang/en.sh`
- Modify: `deploy/scripts/lang/zh.sh`

- [ ] **Step 1: Add English messages**

Append to `deploy/scripts/lang/en.sh` after the `# Deploy` section (after line 128):

```bash
# Templates
MSG_DEPLOY_TEMPLATES="Importing templates..."
MSG_DEPLOY_TEMPLATES_DONE="Templates imported"
MSG_DEPLOY_TEMPLATES_REPO="Pushing repo"
MSG_DEPLOY_TEMPLATES_YAML="Importing template"
MSG_DEPLOY_TEMPLATES_SKIP="already exists, skipping"
```

- [ ] **Step 2: Add Chinese messages**

Append to `deploy/scripts/lang/zh.sh` after the `# Deploy` section (same relative position):

```bash
# Templates
MSG_DEPLOY_TEMPLATES="导入模版..."
MSG_DEPLOY_TEMPLATES_DONE="模版导入完成"
MSG_DEPLOY_TEMPLATES_REPO="推送仓库"
MSG_DEPLOY_TEMPLATES_YAML="导入模版"
MSG_DEPLOY_TEMPLATES_SKIP="已存在，跳过"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lang/en.sh deploy/scripts/lang/zh.sh
git commit -m "feat: add i18n messages for template import"
```

---

### Task 3: Implement import_templates()

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh` (add function after `bootstrap_gitea`)

- [ ] **Step 1: Add the import_templates function**

Add this function after `bootstrap_gitea()` (after line 199) in `deploy/scripts/lib/deploy.sh`:

```bash
import_templates() {
    log_info "$MSG_DEPLOY_TEMPLATES"

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local files_dir="${PROJECT_DIR}/files"
    local gitea_admin="${ADMIN_EMAIL%%@*}"
    local gitea_pass="${ADMIN_PASSWORD}"
    local gitea_api="http://localhost:3000/api/v1"
    local api_base="http://localhost:6802"  # API container internal port, matches docker-compose config

    # Check prerequisites
    if [[ -z "${GITEA_ACCESS_TOKEN:-}" ]]; then
        log_warn "No Gitea token — skipping template import"
        return 1
    fi

    if [[ ! -d "$files_dir" ]]; then
        log_warn "No files/ directory found — skipping template import"
        return 0
    fi

    # Helper: run curl inside gitea container (defined early for health check)
    _gitea_curl() {
        docker compose -f "$compose_file" exec -T gitea curl -s "$@"
    }

    # Check Gitea is healthy
    local gitea_status
    gitea_status=$(_gitea_curl -o /dev/null -w "%{http_code}" "${gitea_api}/version" 2>/dev/null || echo "000")
    if [[ "$gitea_status" != "200" ]]; then
        log_warn "Gitea not healthy (HTTP $gitea_status) — skipping template import"
        return 1
    fi

    # Check API is healthy
    local api_status
    api_status=$(docker compose -f "$compose_file" exec -T api curl -s -o /dev/null -w "%{http_code}" "${api_base}/health" 2>/dev/null || echo "000")
    if [[ "$api_status" != "200" ]]; then
        log_warn "API not healthy (HTTP $api_status) — skipping template import"
        return 1
    fi

    # Phase 1: Push git repos from tar.gz files
    local tar_file repo_name
    for tar_file in "$files_dir"/*.tar.gz; do
        [[ -f "$tar_file" ]] || continue
        repo_name=$(basename "$tar_file" | sed 's/-main\.tar\.gz$//')

        # Check if repo exists
        local repo_status
        repo_status=$(_gitea_curl -o /dev/null -w "%{http_code}" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            "${gitea_api}/repos/launchpad/${repo_name}")

        if [[ "$repo_status" == "200" ]]; then
            # Check if repo has commits (not empty)
            local commits
            commits=$(_gitea_curl \
                -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
                "${gitea_api}/repos/launchpad/${repo_name}/commits?limit=1")
            if echo "$commits" | grep -q '"sha"'; then
                log_ok "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name} — $MSG_DEPLOY_TEMPLATES_SKIP"
                continue
            fi
        else
            # Create empty repo
            _gitea_curl -X POST "${gitea_api}/orgs/launchpad/repos" \
                -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
                -H "Content-Type: application/json" \
                -d "{\"name\":\"${repo_name}\",\"auto_init\":false,\"default_branch\":\"main\"}" >/dev/null
        fi

        # Extract and push
        local tmp_dir
        tmp_dir=$(mktemp -d)
        tar xzf "$tar_file" -C "$tmp_dir" --strip-components=1 2>/dev/null || tar xzf "$tar_file" -C "$tmp_dir"

        (
            cd "$tmp_dir"
            git init -b main >/dev/null 2>&1
            git add -A >/dev/null 2>&1
            git commit -m "Initial import" --author="Launchpad <launchpad@${DOMAIN}>" >/dev/null 2>&1

            local remote_url="https://${gitea_admin}:${gitea_pass}@${GITEA_DOMAIN}/launchpad/${repo_name}.git"
            if [[ "${SSL_MODE}" == "selfsigned" ]]; then
                GIT_SSL_NO_VERIFY=1 git push -f "$remote_url" main >/dev/null 2>&1
            else
                git push -f "$remote_url" main >/dev/null 2>&1
            fi
        )
        rm -rf "$tmp_dir"

        log_ok "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name}"
    done

    # Phase 2: Import YAML templates via API
    local yaml_file yaml_content
    for yaml_file in "$files_dir"/*.yaml; do
        [[ -f "$yaml_file" ]] || continue
        local template_name
        template_name=$(basename "$yaml_file" .yaml)

        # Read and substitute $GITLAB_DOMAIN
        yaml_content=$(cat "$yaml_file" | sed "s|\\\$GITLAB_DOMAIN|https://${GITEA_DOMAIN}|g")

        # Escape for JSON (handle newlines, quotes, backslashes)
        local json_yaml
        json_yaml=$(printf '%s' "$yaml_content" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null \
            || printf '%s' "$yaml_content" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')

        # Validate
        local validate_resp
        validate_resp=$(docker compose -f "$compose_file" exec -T api \
            curl -s -X POST "${api_base}/api/v1/templates/validate-yaml" \
            -H "Content-Type: application/json" \
            -d "{\"yaml\": ${json_yaml}}" 2>/dev/null)

        if echo "$validate_resp" | grep -qi "error\|invalid"; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — validation failed: $validate_resp"
            continue
        fi

        # Import
        local import_resp
        import_resp=$(docker compose -f "$compose_file" exec -T api \
            curl -s -X POST "${api_base}/api/v1/templates/from-yaml" \
            -H "Content-Type: application/json" \
            -d "{\"yaml\": ${json_yaml}}" 2>/dev/null)

        if echo "$import_resp" | grep -qi "error"; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — import failed: $import_resp"
        else
            log_ok "$MSG_DEPLOY_TEMPLATES_YAML ${template_name}"
        fi
    done

    log_done "$MSG_DEPLOY_TEMPLATES_DONE"
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat: add import_templates() for auto-importing template repos and YAMLs"
```

---

### Task 4: Integrate import_templates into deploy flow

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh:93-94` (deploy_services)
- Modify: `deploy/scripts/lib/deploy.sh:344` (resume_deploy)
- Modify: `deploy/setup.sh:85-86`

- [ ] **Step 1: Add import_templates call in deploy_services**

In `deploy_services()`, after line 94 (`progress_update "Gitea bootstrapped"`), add:

```bash
    _draw_progress "$MSG_DEPLOY_TEMPLATES"
    import_templates || log_warn "Template import failed — you can retry with: ./setup.sh --import-templates"
    progress_update "$MSG_DEPLOY_TEMPLATES_DONE"
```

Also update the total steps on line 57 from `11` to `12`:
```bash
    # Total steps: infra(2) + db(2) + gitea(2) + templates(1) + services(4) + nginx(1) = 12
    progress_start 12
```

- [ ] **Step 2: Add import_templates call in resume_deploy**

In `resume_deploy()`, after line 344 (`bootstrap_gitea`), add:

```bash
    # Phase 5b: Template import (idempotent)
    import_templates || log_warn "Template import failed — retry with: ./setup.sh --import-templates"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat: integrate import_templates into deploy and resume flows"
```

---

### Task 5: Add --import-templates CLI flag

**Files:**
- Modify: `deploy/setup.sh`

- [ ] **Step 1: Add CLI parsing for --import-templates**

In `deploy/setup.sh`, add to the `case` statement inside the `while` loop (after line 24):

```bash
        --import-templates) ACTION="import-templates"; shift ;;
```

- [ ] **Step 2: Add load_saved_config helper function**

Add this before the `case "$ACTION"` line (before line 44):

```bash
# Helper: load saved config for granular commands
load_saved_config() {
    if [[ ! -d "${DEPLOY_DIR}/generated" ]]; then
        log_error "No deployment found. Run ./setup.sh first."
        exit 1
    fi
    source "${DEPLOY_DIR}/versions.conf"
    source "${DEPLOY_DIR}/generated/.setup.conf"
    source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    load_secrets || { log_error "Cannot load secrets."; exit 1; }
}
```

- [ ] **Step 3: Add case handler for import-templates**

In the `case "$ACTION"` block, add before the `help)` case:

```bash
    import-templates)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        import_templates
        ;;
```

- [ ] **Step 4: Update help output**

Add to the help section:

```bash
        echo "  --import-templates  Import templates to API & Gitea"
```

- [ ] **Step 5: Commit**

```bash
git add deploy/setup.sh
git commit -m "feat: add --import-templates CLI flag for standalone template import"
```

---

## Chunk 2: Full Uninstall & Granular CLI Flags

> **Prerequisite:** Chunk 1 Task 5 must be completed first (defines `load_saved_config` used by Task 8).

### Task 6: Add i18n messages for uninstall-all

**Files:**
- Modify: `deploy/scripts/lang/en.sh`
- Modify: `deploy/scripts/lang/zh.sh`

- [ ] **Step 1: Add English messages**

Append to `deploy/scripts/lang/en.sh` in the Errors section:

```bash
MSG_UNINSTALL_ALL_WARN="COMPLETE REMOVAL: services, data, k3s, certificates, and all configuration"
MSG_UNINSTALL_ALL_CONFIRM="Are you absolutely sure? This cannot be undone."
MSG_UNINSTALL_ALL_DONE="Environment fully cleaned"
```

- [ ] **Step 2: Add Chinese messages**

Append to `deploy/scripts/lang/zh.sh`:

```bash
MSG_UNINSTALL_ALL_WARN="完全卸载：服务、数据、k3s、证书和所有配置"
MSG_UNINSTALL_ALL_CONFIRM="确定要完全卸载吗？此操作不可恢复。"
MSG_UNINSTALL_ALL_DONE="环境已完全清理"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lang/en.sh deploy/scripts/lang/zh.sh
git commit -m "feat: add i18n messages for --uninstall-all"
```

---

### Task 7: Implement uninstall_all()

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh` (add after `uninstall_services`)

- [ ] **Step 1: Add uninstall_all function**

Add after `uninstall_services()` in `deploy/scripts/lib/deploy.sh`:

```bash
uninstall_all() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"

    # Load config to know K8S_MODE
    if [[ -f "${DEPLOY_DIR}/generated/.setup.conf" ]]; then
        source "${DEPLOY_DIR}/generated/.setup.conf"
    fi

    log_warn "$MSG_UNINSTALL_ALL_WARN"
    echo ""
    if ! ask_confirm "$MSG_UNINSTALL_ALL_CONFIRM" "N"; then
        return 0
    fi

    # Step 1: Stop services and remove volumes
    if [[ -f "$compose_file" ]]; then
        log_info "Stopping services..."
        docker compose -f "$compose_file" down -v 2>/dev/null || true
        log_ok "Services and volumes removed"
    fi

    # Step 2: Remove heartbeat cron, script, env
    if crontab -l 2>/dev/null | grep -q 'k3s-heartbeat'; then
        crontab -l 2>/dev/null | grep -v 'k3s-heartbeat' | crontab - 2>/dev/null || true
        log_ok "Heartbeat cron removed"
    fi
    rm -f /usr/local/bin/k3s-heartbeat.sh
    rm -f /etc/default/k3s-heartbeat

    # Step 3: Uninstall k3s if builtin
    if [[ -f /usr/local/bin/k3s-uninstall.sh ]]; then
        if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
            log_info "Uninstalling k3s..."
            /usr/local/bin/k3s-uninstall.sh 2>/dev/null || true
            log_ok "k3s uninstalled"
        fi
    fi

    # Step 4: Remove generated directory
    if [[ -d "${DEPLOY_DIR}/generated" ]]; then
        rm -rf "${DEPLOY_DIR}/generated"
        log_ok "Generated config removed"
    fi

    log_done "$MSG_UNINSTALL_ALL_DONE"
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat: add uninstall_all() for complete environment cleanup"
```

---

### Task 8: Add --uninstall-all and granular CLI flags to setup.sh

**Files:**
- Modify: `deploy/setup.sh`

- [ ] **Step 1: Add new CLI argument parsing**

In the `while` loop `case` block, add these entries:

```bash
        --uninstall-all)    ACTION="uninstall-all";    shift ;;
        --setup-k3s)        ACTION="setup-k3s";        shift ;;
        --setup-certs)      ACTION="setup-certs";      shift ;;
        --setup-db)         ACTION="setup-db";         shift ;;
        --restart)          [[ -z "${2:-}" ]] && { log_error "--restart requires a service name"; exit 1; }
                            ACTION="restart"; RESTART_TARGET="$2"; shift 2 ;;
```

- [ ] **Step 2: Add restart_service function to deploy.sh**

Add to `deploy/scripts/lib/deploy.sh` after `uninstall_all()`:

```bash
restart_service() {
    local target="$1"
    local valid_targets="api ui router gateway nginx gitea all"
    if [[ ! " $valid_targets " =~ " $target " ]]; then
        log_error "Unknown service: $target (valid: $valid_targets)"
        exit 1
    fi
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_error "No deployment found. Run ./setup.sh first."
        exit 1
    fi
    if [[ "$target" == "all" ]]; then
        docker compose -f "$compose_file" restart
    else
        docker compose -f "$compose_file" restart "$target"
    fi
    log_done "Restarted: $target"
}
```

- [ ] **Step 3: Add case handlers for all new actions**

In the `case "$ACTION"` block, add before `help)`:

```bash
    uninstall-all)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        uninstall_all
        ;;
    setup-k3s)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        load_saved_config
        setup_kubernetes
        ;;
    setup-certs)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        load_saved_config
        setup_certificates
        ;;
    setup-db)
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        load_saved_config
        init_database
        ;;
    restart)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        restart_service "$RESTART_TARGET"
        ;;
```

- [ ] **Step 4: Update help output**

Replace the help section with:

```bash
    help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Install:"
        echo "  (none)              Fresh install (interactive wizard)"
        echo "  --config FILE       Non-interactive install using config file"
        echo "  --reconfigure       Re-enter configuration wizard"
        echo ""
        echo "Manage:"
        echo "  --status            Show service status"
        echo "  --restart <svc>     Restart service (api|ui|router|gateway|nginx|gitea|all)"
        echo "  --upgrade           Update image versions"
        echo ""
        echo "Setup Phases:"
        echo "  --setup-k3s         Install/reconfigure k3s only"
        echo "  --setup-certs       Regenerate SSL certificates only"
        echo "  --setup-db          Run database initialization only"
        echo "  --import-templates  Import templates to API & Gitea"
        echo ""
        echo "Teardown:"
        echo "  --uninstall         Stop services, remove containers and data"
        echo "  --uninstall-all     Complete removal (services + k3s + config)"
        echo "  --resume            Resume from failed deploy phase"
        echo ""
        echo "General:"
        echo "  --verbose, -v       Show detailed output"
        echo "  --help              Show this help"
        ;;
```

- [ ] **Step 5: Commit**

```bash
git add deploy/setup.sh deploy/scripts/lib/deploy.sh
git commit -m "feat: add --uninstall-all, --setup-*, --restart CLI flags"
```

---

## Chunk 3: E2E Test Suite

### Task 9: Create test infrastructure

**Files:**
- Create: `tests/lib/config.sh`
- Create: `tests/lib/helpers.sh`

- [ ] **Step 1: Create test config**

Create `tests/lib/config.sh`:

```bash
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
```

- [ ] **Step 2: Create test helpers**

Create `tests/lib/helpers.sh`:

```bash
#!/bin/bash
# E2E test helper functions

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BOLD='\033[1m'; NC='\033[0m'

ssh_exec() {
    $TEST_SSH "$@" 2>/dev/null
}

sync_files() {
    local local_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
    echo -e "${YELLOW}Syncing files to ${TEST_SERVER}...${NC}"
    $TEST_SCP -r "${local_dir}/deploy/" "root@${TEST_SERVER}:${TEST_DEPLOY_DIR}/deploy/" >/dev/null
    $TEST_SCP -r "${local_dir}/files/" "root@${TEST_SERVER}:${TEST_DEPLOY_DIR}/files/" >/dev/null
    # Also sync root setup.sh
    $TEST_SCP "${local_dir}/deploy/setup.sh" "root@${TEST_SERVER}:${TEST_DEPLOY_DIR}/setup.sh" >/dev/null 2>&1 || true
    echo -e "${GREEN}Files synced${NC}"
}

test_start() {
    local name="$1"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo ""
    echo -e "${BOLD}━━━ TEST: ${name} ━━━${NC}"
}

test_pass() {
    local name="$1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "  ${GREEN}✓ PASS${NC}: ${name}"
}

test_fail() {
    local name="$1" reason="${2:-}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    FAILED_TESTS+=("$name")
    echo -e "  ${RED}✗ FAIL${NC}: ${name}"
    [[ -n "$reason" ]] && echo -e "    ${RED}Reason: ${reason}${NC}"
}

assert_eq() {
    local actual="$1" expected="$2" msg="$3"
    if [[ "$actual" == "$expected" ]]; then
        test_pass "$msg"
    else
        test_fail "$msg" "expected='${expected}' actual='${actual}'"
    fi
}

assert_ne() {
    local actual="$1" unexpected="$2" msg="$3"
    if [[ "$actual" != "$unexpected" ]]; then
        test_pass "$msg"
    else
        test_fail "$msg" "got unexpected value='${unexpected}'"
    fi
}

assert_contains() {
    local haystack="$1" needle="$2" msg="$3"
    if echo "$haystack" | grep -q "$needle"; then
        test_pass "$msg"
    else
        test_fail "$msg" "output does not contain '${needle}'"
    fi
}

assert_http() {
    local url="$1" expected_code="$2" msg="${3:-HTTP ${url} returns ${expected_code}}"
    local actual_code
    actual_code=$(ssh_exec "curl -sk -o /dev/null -w '%{http_code}' --max-time 10 '${url}'" 2>/dev/null || echo "000")
    assert_eq "$actual_code" "$expected_code" "$msg"
}

assert_healthy() {
    local service="$1"
    local status
    status=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && docker compose -f generated/docker-compose.yml ps --format json ${service}" 2>/dev/null \
        | grep -o '"Health":"[^"]*"' | cut -d'"' -f4)
    if [[ "$status" == "healthy" ]]; then
        test_pass "${service} is healthy"
    else
        test_fail "${service} is healthy" "status='${status}'"
    fi
}

print_summary() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
    echo -e "${BOLD}  Test Summary${NC}"
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
    echo -e "  Total:  ${TESTS_TOTAL}"
    echo -e "  ${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "  ${RED}Failed: ${TESTS_FAILED}${NC}"
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo ""
        echo -e "  ${RED}Failed tests:${NC}"
        for t in "${FAILED_TESTS[@]}"; do
            echo -e "    ${RED}✗${NC} $t"
        done
    fi
    echo -e "${BOLD}═══════════════════════════════════════${NC}"
}
```

- [ ] **Step 3: Commit**

```bash
git add tests/lib/config.sh tests/lib/helpers.sh
git commit -m "feat: add E2E test infrastructure (config + helpers)"
```

---

### Task 10: Create test cases

**Files:**
- Create: `tests/cases/01-clean-install.sh`
- Create: `tests/cases/02-service-health.sh`
- Create: `tests/cases/03-gitea-bootstrap.sh`
- Create: `tests/cases/04-template-import.sh`
- Create: `tests/cases/05-endpoints.sh`
- Create: `tests/cases/06-cluster-register.sh`
- Create: `tests/cases/07-restart-service.sh`
- Create: `tests/cases/08-idempotent.sh`
- Create: `tests/cases/09-uninstall-all.sh`

- [ ] **Step 1: Create 01-clean-install.sh**

```bash
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
```

- [ ] **Step 2: Create 02-service-health.sh**

```bash
#!/bin/bash
# Test: All services are healthy

test_start "02: Service Health"

for svc in postgres redis gitea api ui router gateway nginx; do
    assert_healthy "$svc"
done
```

- [ ] **Step 3: Create 03-gitea-bootstrap.sh**

```bash
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
```

- [ ] **Step 4: Create 04-template-import.sh**

```bash
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
```

- [ ] **Step 5: Create 05-endpoints.sh**

```bash
#!/bin/bash
# Test: All web endpoints respond

test_start "05: Endpoints"

assert_http "https://${TEST_LAUNCHPAD_DOMAIN}" "200" "Dashboard returns 200"
assert_http "https://${TEST_GITEA_DOMAIN}" "200" "Gitea returns 200"
assert_http "https://${TEST_LAUNCHPAD_DOMAIN}/admin" "200" "Admin returns 200"
```

- [ ] **Step 6: Create 06-cluster-register.sh**

```bash
#!/bin/bash
# Test: K8s cluster registered and heartbeat working

test_start "06: Cluster Registration"

# Check cluster registered via admin API
cluster_status=$(ssh_exec "curl -sk https://${TEST_LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/clusters" 2>/dev/null)
assert_contains "$cluster_status" "local-k3s" "Cluster 'local-k3s' found in API"

# Check heartbeat cron exists
has_cron=$(ssh_exec "crontab -l 2>/dev/null | grep -c k3s-heartbeat" || echo "0")
assert_ne "$has_cron" "0" "Heartbeat cron job exists"

# Check heartbeat script exists
has_script=$(ssh_exec "test -f /usr/local/bin/k3s-heartbeat.sh && echo yes || echo no")
assert_eq "$has_script" "yes" "Heartbeat script installed"
```

- [ ] **Step 7: Create 07-restart-service.sh**

```bash
#!/bin/bash
# Test: --restart flag works

test_start "07: Restart Service"

# Restart API
restart_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart api" 2>&1)
exit_code=$?
assert_eq "$exit_code" "0" "--restart api exits 0"

# Wait for health
sleep 10
assert_healthy "api"

# Test invalid target
bad_output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --restart badname" 2>&1)
bad_code=$?
assert_ne "$bad_code" "0" "--restart badname exits non-zero"
```

- [ ] **Step 8: Create 08-idempotent.sh**

```bash
#!/bin/bash
# Test: Re-running install is idempotent

test_start "08: Idempotent Re-deploy"

output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && ./setup.sh --config /tmp/test-setup.conf" 2>&1)
exit_code=$?

assert_eq "$exit_code" "0" "Re-install exits with code 0"

# All services still healthy
for svc in api ui router gateway nginx gitea; do
    assert_healthy "$svc"
done
```

- [ ] **Step 9: Create 09-uninstall-all.sh**

```bash
#!/bin/bash
# Test: --uninstall-all cleans everything

test_start "09: Uninstall All"

output=$(ssh_exec "cd ${TEST_DEPLOY_DIR} && echo 'Y' | ./setup.sh --uninstall-all" 2>&1)
exit_code=$?

assert_eq "$exit_code" "0" "--uninstall-all exits 0"

# No Docker containers
containers=$(ssh_exec "docker ps -q --filter 'label=com.docker.compose.project'" 2>/dev/null | wc -l | tr -d ' ')
assert_eq "$containers" "0" "No Docker containers running"

# No k3s
k3s_running=$(ssh_exec "systemctl is-active k3s 2>/dev/null || echo inactive")
assert_eq "$k3s_running" "inactive" "k3s is not running"

# No generated directory
gen_exists=$(ssh_exec "test -d ${TEST_DEPLOY_DIR}/generated && echo yes || echo no")
assert_eq "$gen_exists" "no" "generated/ directory removed"
```

- [ ] **Step 10: Commit**

```bash
git add tests/cases/
git commit -m "feat: add E2E test cases (01-09)"
```

---

### Task 11: Create main test runner

**Files:**
- Create: `tests/e2e.sh`

- [ ] **Step 1: Create the test runner**

Create `tests/e2e.sh`:

```bash
#!/bin/bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load config and helpers
source "${SCRIPT_DIR}/lib/config.sh"
source "${SCRIPT_DIR}/lib/helpers.sh"

# Parse args
DO_SYNC=false
SELECTED_CASES=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --sync) DO_SYNC=true; shift ;;
        --help|-h)
            echo "Usage: $0 [--sync] [case_numbers...]"
            echo ""
            echo "  --sync          Sync local files to server before testing"
            echo "  case_numbers    Run specific cases (e.g., 03 04)"
            echo ""
            echo "Examples:"
            echo "  $0              Run all tests"
            echo "  $0 --sync       Sync files, then run all tests"
            echo "  $0 03 04        Run only cases 03 and 04"
            exit 0
            ;;
        *)  SELECTED_CASES+=("$1"); shift ;;
    esac
done

echo ""
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo -e "${BOLD}  AniLaunchpad E2E Tests${NC}"
echo -e "${BOLD}═══════════════════════════════════════${NC}"
echo "  Server: ${TEST_SERVER}"
echo "  Domain: ${TEST_DOMAIN}"
echo ""

# Verify SSH connectivity
if ! ssh_exec "echo ok" >/dev/null 2>&1; then
    echo -e "${RED}Cannot connect to ${TEST_SERVER}${NC}"
    exit 1
fi

# Sync files if requested
if [[ "$DO_SYNC" == "true" ]]; then
    sync_files
fi

# Collect test cases
ALL_CASES=()
for f in "${SCRIPT_DIR}"/cases/*.sh; do
    [[ -f "$f" ]] || continue
    ALL_CASES+=("$f")
done

# Filter if specific cases requested
RUN_CASES=()
if [[ ${#SELECTED_CASES[@]} -gt 0 ]]; then
    for num in "${SELECTED_CASES[@]}"; do
        for f in "${ALL_CASES[@]}"; do
            if [[ "$(basename "$f")" == "${num}-"* ]]; then
                RUN_CASES+=("$f")
            fi
        done
    done
else
    RUN_CASES=("${ALL_CASES[@]}")
fi

if [[ ${#RUN_CASES[@]} -eq 0 ]]; then
    echo -e "${RED}No test cases found${NC}"
    exit 1
fi

echo "Running ${#RUN_CASES[@]} test case(s)..."

# Run test cases
for case_file in "${RUN_CASES[@]}"; do
    source "$case_file"
done

# Summary
print_summary

# Exit code
[[ $TESTS_FAILED -eq 0 ]] && exit 0 || exit 1
```

- [ ] **Step 2: Make executable**

```bash
chmod +x tests/e2e.sh
```

- [ ] **Step 3: Commit**

```bash
git add tests/e2e.sh
git commit -m "feat: add E2E test runner"
```

---

### Task 12: Sync all changes to remote and run tests

- [ ] **Step 1: Sync files to remote server**

```bash
scp -r deploy/ root@10.233.201.133:~/workspace/launchpad-deploy/deploy/
scp -r tests/ root@10.233.201.133:~/workspace/launchpad-deploy/tests/
scp -r files/ root@10.233.201.133:~/workspace/launchpad-deploy/files/
# Also sync root-level copies of scripts that the root setup.sh uses
scp deploy/scripts/lib/deploy.sh root@10.233.201.133:~/workspace/launchpad-deploy/scripts/lib/deploy.sh
scp deploy/scripts/lang/en.sh root@10.233.201.133:~/workspace/launchpad-deploy/scripts/lang/en.sh
scp deploy/scripts/lang/zh.sh root@10.233.201.133:~/workspace/launchpad-deploy/scripts/lang/zh.sh
scp deploy/setup.sh root@10.233.201.133:~/workspace/launchpad-deploy/setup.sh
```

- [ ] **Step 2: Run E2E tests**

```bash
./tests/e2e.sh
```

Expected: All 9 test cases pass.

- [ ] **Step 3: If failures, fix and re-run specific cases**

```bash
./tests/e2e.sh <failed_case_number>
```

- [ ] **Step 4: Final commit after all tests pass**

```bash
git add tests/ deploy/ docs/
git commit -m "test: all E2E tests passing"
```
