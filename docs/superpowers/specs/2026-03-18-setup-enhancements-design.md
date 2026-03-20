# Setup Enhancements — Design Spec

**Date:** 2026-03-18

**Scope:** 5 independent sub-projects for launchpad-setup improvements:
1. Template & repo auto-import
2. Full uninstall (`--uninstall-all`)
3. Wizard UI redesign (Style C)
4. Granular CLI flags (`--setup-*`, `--restart`)
5. QA automation (E2E test suite)

**Architecture:** Each sub-project is independently implementable. They share the existing module structure in `deploy/scripts/lib/` and the CLI entry point in `setup.sh`.

---

## Sub-project 1: Template & Repo Auto-Import

### Goal
During initial setup, automatically import template YAML definitions into the Launchpad API and push template git repos to Gitea's `launchpad` organization.

### Files in `files/`
| File | Type | Action |
|------|------|--------|
| `ani-code-main.tar.gz` | Git repo only | Push to Gitea `launchpad/ani-code` |
| `nodejs-helloworld-main.tar.gz` | Git repo + template | Push to Gitea + import YAML |
| `nodejs-helloworld.yaml` | Template definition | POST to API |
| `test-openclaw-main.tar.gz` | Git repo + template | Push to Gitea + import YAML |
| `test-openclaw.yaml` | Template definition | POST to API |

### New Function: `import_templates()`

**Location:** `deploy/scripts/lib/deploy.sh`

**Flow:**
1. Check prerequisites: Gitea healthy (via gitea container curl), `GITEA_ACCESS_TOKEN` exists, API healthy (via `wget` to `/health` on port 6802)
2. **Phase 1 — Git repos:** For each `files/*.tar.gz`:
   - Derive `repo_name` from filename (strip `-main.tar.gz`)
   - Check if repo exists via Gitea API (inside gitea container): `GET /api/v1/repos/launchpad/{repo_name}`
   - If not: `POST /api/v1/orgs/launchpad/repos` to create empty repo
   - Check if repo has commits: `GET /api/v1/repos/launchpad/{repo_name}/commits?limit=1`
   - If no commits: extract tar.gz to temp dir, push content via subshell
   - **Critical:** Subshell must: `trap - EXIT` (clear inherited tty trap), set `GIT_AUTHOR_NAME/EMAIL` + `GIT_COMMITTER_NAME/EMAIL` (server has no global git config), NOT use `local` keyword (invalid in subshells)
   - For self-signed SSL: `GIT_SSL_NO_VERIFY=1`
   - Clean up temp dir
3. **Phase 2 — YAML templates:** For each `files/*.yaml`:
   - Read content, substitute `$GITLAB_DOMAIN` → `https://${GITEA_DOMAIN}`
   - **Use admin port 6804** (no auth required), NOT API port 6802 (requires JWT)
   - Validate: `POST http://localhost:6804/api/templates/yaml/validate` with `{"yaml": "<content>"}`
   - Check `"valid":false` (not just presence of "errors" field, since `"errors":[]` is valid)
   - Import: `POST http://localhost:6804/api/templates/yaml/create` with `{"yaml": "<content>", "isOfficial": true}`
   - Check `"success":false` for failure detection (not grep for "error")
   - **Note:** API container has `wget` not `curl`; use `docker compose exec -T api wget -q -O-`
4. Idempotent: skip existing repos with commits

**Relationship with `bootstrap_gitea()`:** `bootstrap_gitea()` no longer creates the `ani-code` repo — all repo creation is delegated to `import_templates()`. `bootstrap_gitea()` retains: admin user creation, token generation, org creation.

**API authentication:**
- Gitea API: `GITEA_ACCESS_TOKEN` (passed as header inside gitea container)
- Template YAML import: admin port 6804 (no auth needed — port-based access control)
- Git push: `https://{gitea_admin}:{gitea_pass}@{GITEA_DOMAIN}/...`

**Integration:**
- Called **after API is healthy** in `deploy_services()` flow (NOT after bootstrap_gitea — API must be running for YAML import)
- Also called in `resume_deploy()` to handle resume-after-failure scenarios
- Standalone: `./setup.sh --import-templates` (explicitly loads `GITEA_ACCESS_TOKEN` from `.env`)

**`files/` directory path:** Differs by entry point. When running from root `setup.sh`, check `${DEPLOY_DIR}/files`; when from `deploy/setup.sh`, check `${PROJECT_DIR}/files`. Implementation checks both.

### i18n Messages
- `MSG_DEPLOY_TEMPLATES="Importing templates..." / "导入模版..."`
- `MSG_DEPLOY_TEMPLATES_DONE="Templates imported" / "模版导入完成"`

---

## Sub-project 2: Full Uninstall

### Goal
Provide `--uninstall-all` that completely removes all deployed components including k3s, leaving the machine clean.

### Two-tier Uninstall

**`--uninstall` (existing, unchanged):**
- Warns about data loss
- Confirms (default: N)
- `docker compose down -v`
- Removes containers and volumes

**`--uninstall-all` (new):**
- Stronger warning: "COMPLETE REMOVAL — services, data, k3s, certificates, all config"
- **Confirmation:** Uses `ask_confirm` if tty available; for non-interactive (SSH/piped input), reads "Y" from stdin via `[ -t 0 ] || [ -t 1 ]` check
- Steps (order matters — compose file must be used before generated/ is deleted):
  1. Resolve compose file path: `compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"`
  2. `docker compose -f "$compose_file" down -v` (services + volumes)
  3. Remove heartbeat: `crontab -l | grep -v k3s-heartbeat | crontab -`, `rm -f /usr/local/bin/k3s-heartbeat.sh`, `rm -f /etc/default/k3s-heartbeat`
  4. If `K8S_MODE=builtin` and `/usr/local/bin/k3s-uninstall.sh` exists: run `k3s-uninstall.sh`
  5. `rm -rf "${DEPLOY_DIR}/generated/"` (configs, secrets, certs, compose file — safe now since compose is no longer needed)
  6. Log: "Environment fully cleaned"

**Config loading:** Sources `generated/.setup.conf` to determine `K8S_MODE`. If missing, checks for `k3s-uninstall.sh` existence as fallback.

**CLI handler note:** Must source language file before calling `uninstall_all()` (the function uses i18n variables). Source `.setup.conf` if it exists, then `scripts/lang/${LANG_CHOICE:-en}.sh`.

### New Function: `uninstall_all()`

**Location:** `deploy/scripts/lib/deploy.sh`

### CLI Change
```bash
--uninstall-all) ACTION="uninstall-all"; shift ;;
```

### i18n Messages
- `MSG_UNINSTALL_ALL_WARN="COMPLETE REMOVAL — ..." / "完全卸载 — ..."`
- `MSG_UNINSTALL_ALL_DONE="Environment fully cleaned" / "环境已完全清理"`

---

## Sub-project 3: Wizard UI Redesign (Style C)

### Goal
Replace the current wizard UI with a polished "Style C" design: centered brand header, tab-style step navigation, color-coded accent blocks, and a deploy progress display.

### Visual Design Reference
See mockups in `.superpowers/brainstorm/97105-1773797594/wizard-all-pages.html`

### Page Structure (6 pages, maps 1:1 to current wizard)

| Tab | Page Function | Inputs |
|-----|--------------|--------|
| 1.Basic | `_page_basic()` | Language, domain, subdomain, confirm domains, registry, image versions (API/UI/Router/Gateway/Gitea) |
| 2.SSL | `_page_ssl()` | Cert method (select), DNS provider (select, conditional), API token (conditional) |
| 3.DB | `_page_database()` | DB mode (select), external: 7 connection URLs + Redis host/port/password, builtin: auto-configured info |
| 4.K8s | `_page_k8s()` | K8s mode (select), kubeconfig path, external: context/ingress/storage class |
| 5.Adv | `_page_advanced()` | Enter advanced? (confirm), module picker (multi-select), per-module inputs |
| 6.Deploy | `_page_summary()` | Read-only summary with deploy confirmation |

### Brand Header
```
        ■ ■ ■ ■           (4 colored squares: red, yellow, green, blue)
      AniLaunchpad         (bold white, centered)
 Production-grade ...      (dim, centered)

  1.Basic  2.SSL  3.DB  4.K8s  5.Adv  6.Deploy    (active = green+bold)
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━       (separator)
```

### Input Display Patterns
- **Answered field:** `  ▸ Label         value` (green ▸)
- **Select menu:** `  ❯ Active option` (green), `    Inactive option` (default)
- **Accent block:** Colored left border `┃` with bold colored header
  - Green: domains, cluster info, success states
  - Blue: infrastructure, images, Redis
  - Yellow: connection URLs, SMTP, images, DNS warnings
  - Red: admin/credentials, sensitive info

### Navigation
- **Page gate (page 2+):** `_wizard_page_gate()` renders header + tabs, then waits for Enter (continue) or b (back). Returns "back" or "continue" via stdout. Same mechanism as current implementation — each page function checks the return and does `return 1` for back, which `run_wizard()` catches to decrement page index.
- **Bottom bar:** `[ Enter: next ]  [ b: back ]  [ q: quit ]`
- **q support (new):** Exits with "Setup cancelled" message

### Deploy Progress
```
  ━━━━━━━━━━━━━━━━━━━━━━━━━━ ━━━━━━━━━━━━━━  65%

  ✓ Infrastructure started
  ✓ PostgreSQL ready
  ⠋ Starting API...
  ○ UI
  ○ Nginx
```
- Green filled bar + dark unfilled, percentage right-aligned
- `✓` completed (green), `⠋` spinner (yellow), `○` pending (dim)

### New Helper Functions in `common.sh`
- `print_centered(text)` — center text to terminal width
- `print_brand_header()` — the 4-squares + title + subtitle
- `print_tab_bar(active_index)` — numbered step tabs with 6 positions
- `print_accent_block(color, title, lines...)` — left-border accent block
- `print_hotkey_bar(page)` — bottom navigation legend
- `_draw_deploy_status(phase, total, steps_status[])` — progress bar + phase list

### Files Modified
- `deploy/scripts/lib/common.sh` — new helper functions
- `deploy/scripts/lib/interact.sh` — rewrite all 6 page functions + `_wizard_header` + `_wizard_page_gate` to use Style C
- `deploy/scripts/lib/deploy.sh` — rewrite `deploy_services()` progress display, `show_result()`
- `deploy/scripts/lang/en.sh` — new i18n keys
- `deploy/scripts/lang/zh.sh` — new i18n keys

---

## Sub-project 4: Granular CLI Flags

### Goal
Support running individual setup phases or restarting specific services without a full install flow.

### New CLI Arguments

**Setup flags** (run a specific setup phase):
```
--setup-k3s          Install/reconfigure k3s only
--setup-certs        Regenerate SSL certificates only
--setup-db           Run database initialization only
--import-templates   Import templates & push repos to Gitea only
```

**Service flags** (manage running services):
```
--restart <target>   Restart service: api|ui|router|gateway|nginx|gitea|all
```

### Implementation

Each granular flag:
1. Calls `load_saved_config` which: validates `generated/` exists, sources `versions.conf`, `.setup.conf`, `.secrets`, language file
2. Sources required library modules per flag (see table below)
3. Calls the specific function
4. Exits

**Module dependencies per flag:**
| Flag | Modules sourced |
|------|----------------|
| `--setup-k3s` | `common.sh`, `detect.sh`, `k3s.sh` |
| `--setup-certs` | `common.sh`, `detect.sh`, `certs.sh` |
| `--setup-db` | `common.sh`, `database.sh` |
| `--import-templates` | `common.sh`, `detect.sh`, `deploy.sh` |
| `--restart` | `common.sh`, `deploy.sh` |

**CLI parsing:**
```bash
--setup-k3s)        ACTION="setup-k3s";        shift ;;
--setup-certs)      ACTION="setup-certs";       shift ;;
--setup-db)         ACTION="setup-db";          shift ;;
--import-templates) ACTION="import-templates";  shift ;;
--restart)          ACTION="restart"; RESTART_TARGET="${2:-all}"; shift 2 ;;
```

**Case handler:**
```bash
setup-k3s)        load_saved_config; setup_kubernetes ;;
setup-certs)      load_saved_config; setup_certificates ;;
setup-db)         load_saved_config; init_databases ;;
import-templates) load_saved_config; import_templates ;;
restart)          load_saved_config; restart_service "$RESTART_TARGET" ;;
```

### New Function: `restart_service(target)`

**Location:** `deploy/scripts/lib/deploy.sh`

```bash
restart_service() {
    local target="$1"
    local valid_targets="api ui router gateway nginx gitea all"
    if [[ ! " $valid_targets " =~ " $target " ]]; then
        log_error "Unknown service: $target (valid: $valid_targets)"
        exit 1
    fi
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ "$target" == "all" ]]; then
        docker compose -f "$compose_file" restart
    else
        docker compose -f "$compose_file" restart "$target"
    fi
    log_done "Restarted: $target"
}
```

### New Function: `load_saved_config()`

**Location:** `deploy/setup.sh` (inline)

Loads `.setup.conf` + `.secrets` + `versions.conf` + language file. Validates that `generated/` exists.

### Updated Help Output
```
Usage: ./setup.sh [OPTIONS]

Install:
  (none)              Fresh install (interactive wizard)
  --config FILE       Non-interactive install using config file
  --reconfigure       Re-enter configuration wizard

Manage:
  --status            Show service status
  --restart <svc>     Restart service (api|ui|router|gateway|nginx|gitea|all)
  --upgrade           Update image versions

Setup Phases:
  --setup-k3s         Install/reconfigure k3s only
  --setup-certs       Regenerate SSL certificates only
  --setup-db          Run database initialization only
  --import-templates  Import templates to API & Gitea

Teardown:
  --uninstall         Stop services, remove containers and data
  --uninstall-all     Complete removal (services + k3s + config)
  --resume            Resume from failed deploy phase

General:
  --verbose, -v       Show detailed output
  --help              Show this help
```

---

## Sub-project 5: QA Automation

### Goal
Automated E2E test suite that runs against the remote test server (10.233.201.133), verifying the complete setup flow and all features.

### Directory Structure
```
tests/
├── e2e.sh                 # Main runner
├── lib/
│   ├── helpers.sh         # SSH exec, assertions, logging
│   └── config.sh          # Server IP, credentials, timeouts
└── cases/
    ├── 01-clean-install.sh     # Full uninstall + fresh --config install
    ├── 02-service-health.sh    # All containers healthy
    ├── 03-gitea-bootstrap.sh   # Org, repos, token validation
    ├── 04-template-import.sh   # Templates in API, repos in Gitea
    ├── 05-endpoints.sh         # HTTP status checks on all URLs
    ├── 06-cluster-register.sh  # Cluster healthy, heartbeat working
    ├── 07-restart-service.sh   # --restart api, verify recovery
    ├── 08-idempotent.sh        # Re-run --config, everything still works
    └── 09-uninstall-all.sh     # Full cleanup verification
```

### Test Runner: `e2e.sh`

```bash
./tests/e2e.sh              # Run all tests
./tests/e2e.sh 03 04        # Run specific cases
./tests/e2e.sh --sync       # Sync files to server before running
```

**Flow:**
1. Sync local files to remote server via scp
2. Generate test config file (non-interactive)
3. Run each test case in order
4. Collect PASS/FAIL results
5. Print summary, exit 0 (all pass) or 1 (any fail)

### Helper Functions (`lib/helpers.sh`)

```bash
ssh_exec(cmd)                    # Run command on remote via SSH
assert_eq(actual, expected, msg) # Equality assertion
assert_ne(actual, expected, msg) # Not-equal assertion
assert_http(url, code)           # Check HTTP response code (curl -sk)
assert_healthy(service)          # Docker compose health check
assert_contains(haystack, needle, msg)  # Substring check
test_start(name)                 # Log test start
test_pass(name)                  # Log PASS
test_fail(name, reason)          # Log FAIL with context
```

### Test Config (`lib/config.sh`)

All values have defaults but can be overridden via environment variables:

```bash
TEST_SERVER="${TEST_SERVER:-10.233.201.133}"
TEST_SSH="ssh -o StrictHostKeyChecking=no root@${TEST_SERVER}"
TEST_DOMAIN="${TEST_DOMAIN:-testclaw.com}"
TEST_SUBDOMAIN="${TEST_SUBDOMAIN:-corp}"
TEST_REGISTRY="${TEST_REGISTRY:-swr.ap-southeast-1.myhuaweicloud.com/ghisha}"
TEST_VERSION="${TEST_VERSION:-2.0.3}"
DEPLOY_DIR="${TEST_DEPLOY_DIR:-~/workspace/launchpad-deploy}"
```

### Test Cases Summary

| Case | Tests | Verifies |
|------|-------|----------|
| 01-clean-install | Uninstall-all, then --config install | Clean state → successful deploy (depends on sub-project 2: `--uninstall-all`) |
| 02-service-health | All containers running + healthy | postgres, redis, gitea, api, ui, router, gateway, nginx |
| 03-gitea-bootstrap | Gitea API checks | `launchpad` org exists, repos created, token works |
| 04-template-import | API + Gitea checks | Templates in API, repos have commits in Gitea |
| 05-endpoints | HTTP status codes | Dashboard 200, Gitea 200, Admin 200/302 |
| 06-cluster-register | Admin API check | Cluster status healthy, heartbeat recent |
| 07-restart-service | --restart api | Service restarts, returns healthy |
| 08-idempotent | Re-run --config | No errors, all services still healthy |
| 09-uninstall-all | --uninstall-all | No containers, no k3s, no generated/ |

### Dev-QA Loop
- Test failures include: test name, expected value, actual value, relevant logs
- Developer fixes locally, re-runs failed case: `./tests/e2e.sh 04`
- Full suite before declaring done: `./tests/e2e.sh`

---

## Implementation Order

1. **Template import + Full uninstall** (sub-projects 1 & 2) — smallest scope, high value
2. **Granular CLI flags** (sub-project 4) — extends CLI, needed by tests
3. **QA automation** (sub-project 5) — validates everything built so far
4. **Wizard UI redesign** (sub-project 3) — largest scope, visual polish, tested manually

Sub-projects 1-2 and 4 can be tested by the QA suite (sub-project 5). Sub-project 3 (wizard UI) requires manual/interactive testing since it's a TUI.
