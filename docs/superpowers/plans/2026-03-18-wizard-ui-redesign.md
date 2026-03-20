# Wizard UI Redesign (Style C) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current plain wizard UI with the polished "Style C" design — centered brand header with colored squares, tab-style step navigation, color-coded accent blocks, and a deploy progress display with spinner/checkmarks.

**Architecture:** Pure bash — no external dependencies. New rendering helpers added to `common.sh`, wizard page functions rewritten in `interact.sh`, deploy progress display rewritten in `deploy.sh`. All text output goes to `/dev/tty` (interactive) to preserve stdout for return values. Existing i18n pattern (MSG_* variables in lang files) extended with new keys.

**Tech Stack:** Bash 4.3+ (requires namerefs via `local -n`), ANSI escape codes, `/dev/tty` for display

**Design Reference:** `.superpowers/brainstorm/97105-1773797594/wizard-all-pages.html`

**Spec:** `docs/superpowers/specs/2026-03-18-setup-enhancements-design.md` (Sub-project 3, lines 118-188)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `deploy/scripts/lib/common.sh` | Modify | Add 6 new rendering helpers at end of file |
| `deploy/scripts/lib/interact.sh` | Rewrite | Rewrite all 6 page functions + `_wizard_header` + `_wizard_page_gate` + `run_wizard` to use Style C |
| `deploy/scripts/lib/deploy.sh` | Modify | Rewrite `deploy_services()` progress display and `show_result()` |
| `deploy/scripts/lang/en.sh` | Modify | Add new i18n keys for Style C elements |
| `deploy/scripts/lang/zh.sh` | Modify | Add new i18n keys for Style C elements |
| `deploy/scripts/lib/i18n.sh` | Modify | Restyle `select_language()` with brand header |

---

## Chunk 1: Rendering Helpers & i18n

### Task 1: Add i18n keys for Style C elements

**Files:**
- Modify: `deploy/scripts/lang/en.sh`
- Modify: `deploy/scripts/lang/zh.sh`

- [ ] **Step 1: Add new i18n keys to en.sh**

Append after the existing `MSG_ERR_UNINSTALL` line (line 181):

```bash
# Style C UI
MSG_BRAND_SUBTITLE="Production-grade deployment in minutes"
MSG_TAB_BASIC="1.Basic"
MSG_TAB_SSL="2.SSL"
MSG_TAB_DB="3.DB"
MSG_TAB_K8S="4.K8s"
MSG_TAB_ADV="5.Adv"
MSG_TAB_DEPLOY="6.Deploy"
MSG_NAV_ENTER_NEXT="Enter: next"
MSG_NAV_BACK="b: back"
MSG_NAV_QUIT="q: quit"
MSG_NAV_ENTER_DEPLOY="Enter: deploy"
MSG_WIZARD_CANCELLED="Setup cancelled."
MSG_DEPLOY_DEPLOYING="Deploying..."
MSG_DEPLOY_PROGRESS_INFRA="Infrastructure started"
MSG_DEPLOY_PROGRESS_PG="PostgreSQL ready"
MSG_DEPLOY_PROGRESS_REDIS="Redis ready"
MSG_DEPLOY_PROGRESS_GITEA="Gitea healthy"
MSG_DEPLOY_PROGRESS_GITEA_BOOT="Gitea bootstrapped"
MSG_DEPLOY_PROGRESS_DB="Database migrated"
MSG_DEPLOY_PROGRESS_API="API healthy"
MSG_DEPLOY_PROGRESS_UI="UI healthy"
MSG_DEPLOY_PROGRESS_ROUTER="Router ready"
MSG_DEPLOY_PROGRESS_TEMPLATES="Template import"
MSG_DEPLOY_PROGRESS_NGINX="Nginx"
MSG_DEPLOY_PROGRESS_CLUSTER="Cluster registration"
MSG_SUMMARY_DOMAINS="Domains"
MSG_SUMMARY_INFRA="Infrastructure"
MSG_SUMMARY_IMAGES="Images"
MSG_SUMMARY_ADMIN="Admin"
MSG_SUMMARY_MODULES="Modules"
MSG_DOMAIN_GEN_HEADER="Generated domains"
MSG_DB_AUTO_CONFIGURED="Auto-configured"
MSG_DB_AUTO_CREDS="Credentials auto-generated and stored in .secrets"
MSG_DB_CONN_URLS="Connection URLs"
MSG_DB_CONN_FORMAT="Format: postgresql://user:pass@host:port/db?schema=name"
MSG_K8S_CLUSTER_INFO="Cluster Info"
MSG_K8S_AUTO_INSTALL="k3s will be installed automatically during deploy"
```

- [ ] **Step 2: Add corresponding zh.sh keys**

Append after `MSG_ERR_UNINSTALL` line (line 181) in zh.sh:

```bash
# Style C UI
MSG_BRAND_SUBTITLE="生产级一键部署"
MSG_TAB_BASIC="1.基础"
MSG_TAB_SSL="2.SSL"
MSG_TAB_DB="3.数据库"
MSG_TAB_K8S="4.K8s"
MSG_TAB_ADV="5.高级"
MSG_TAB_DEPLOY="6.部署"
MSG_NAV_ENTER_NEXT="Enter: 下一步"
MSG_NAV_BACK="b: 返回"
MSG_NAV_QUIT="q: 退出"
MSG_NAV_ENTER_DEPLOY="Enter: 开始部署"
MSG_WIZARD_CANCELLED="安装已取消。"
MSG_DEPLOY_DEPLOYING="部署中..."
MSG_DEPLOY_PROGRESS_INFRA="基础设施已启动"
MSG_DEPLOY_PROGRESS_PG="PostgreSQL 就绪"
MSG_DEPLOY_PROGRESS_REDIS="Redis 就绪"
MSG_DEPLOY_PROGRESS_GITEA="Gitea 就绪"
MSG_DEPLOY_PROGRESS_GITEA_BOOT="Gitea 初始化完成"
MSG_DEPLOY_PROGRESS_DB="数据库迁移完成"
MSG_DEPLOY_PROGRESS_API="API 就绪"
MSG_DEPLOY_PROGRESS_UI="UI 就绪"
MSG_DEPLOY_PROGRESS_ROUTER="Router 就绪"
MSG_DEPLOY_PROGRESS_TEMPLATES="模版导入"
MSG_DEPLOY_PROGRESS_NGINX="Nginx"
MSG_DEPLOY_PROGRESS_CLUSTER="集群注册"
MSG_SUMMARY_DOMAINS="域名"
MSG_SUMMARY_INFRA="基础设施"
MSG_SUMMARY_IMAGES="镜像"
MSG_SUMMARY_ADMIN="管理员"
MSG_SUMMARY_MODULES="模块"
MSG_DOMAIN_GEN_HEADER="生成的域名"
MSG_DB_AUTO_CONFIGURED="自动配置"
MSG_DB_AUTO_CREDS="凭证已自动生成，存储在 .secrets"
MSG_DB_CONN_URLS="连接地址"
MSG_DB_CONN_FORMAT="格式: postgresql://user:pass@host:port/db?schema=name"
MSG_K8S_CLUSTER_INFO="集群信息"
MSG_K8S_AUTO_INSTALL="部署时将自动安装 k3s"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lang/en.sh deploy/scripts/lang/zh.sh
git commit -m "feat(i18n): add Style C wizard UI message keys"
```

---

### Task 2: Add rendering helper functions to common.sh

**Files:**
- Modify: `deploy/scripts/lib/common.sh` (append after `print_summary` function, line ~277)

These helpers produce the brand header, tab bar, accent blocks, hotkey bar, and deploy status display. All output goes to `/dev/tty`.

- [ ] **Step 1: Add `print_centered` helper**

Append to end of `common.sh`:

```bash
# ─── Style C rendering helpers ───────────────────────────────────────

# Terminal width (cached)
_term_width() {
    tput cols 2>/dev/null || echo 80
}

# Print text centered to terminal width
print_centered() {
    local text="$1"
    # Strip ANSI codes to calculate visible length
    local stripped
    stripped=$(echo -e "$text" | sed 's/\x1b\[[0-9;]*m//g')
    local w
    w=$(_term_width)
    local pad=$(( (w - ${#stripped}) / 2 ))
    [[ $pad -lt 0 ]] && pad=0
    printf "%*s" "$pad" "" >/dev/tty
    printf "%b\n" "$text" >/dev/tty
}
```

- [ ] **Step 2: Add `print_brand_header` helper**

```bash
# Print the 4-colored-squares brand header
print_brand_header() {
    local subtitle="${1:-$MSG_BRAND_SUBTITLE}"
    echo "" >/dev/tty
    print_centered "\033[0;31m■\033[0m \033[1;33m■\033[0m \033[0;32m■\033[0m \033[0;34m■\033[0m"
    print_centered "${BOLD}AniLaunchpad${NC}"
    print_centered "${DIM}${subtitle}${NC}"
    echo "" >/dev/tty
}
```

- [ ] **Step 3: Add `print_tab_bar` helper**

```bash
# Print tab navigation bar. Active tab (1-based index) is green+bold.
print_tab_bar() {
    local active="$1"
    local tabs=("$MSG_TAB_BASIC" "$MSG_TAB_SSL" "$MSG_TAB_DB" "$MSG_TAB_K8S" "$MSG_TAB_ADV" "$MSG_TAB_DEPLOY")
    local line=""
    local i
    for i in "${!tabs[@]}"; do
        local idx=$((i + 1))
        if [[ $idx -eq $active ]]; then
            line+="${GREEN}${BOLD}${tabs[$i]}${NC}  "
        else
            line+="${DIM}${tabs[$i]}${NC}  "
        fi
    done
    printf "  %b\n" "$line" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    echo "" >/dev/tty
}
```

- [ ] **Step 4: Add `print_accent_block` helper**

```bash
# Print a left-border accent block.
# Usage: print_accent_block <color> <title> <lines...>
# color: green, blue, yellow, red
print_accent_block() {
    local color_name="$1" title="$2"
    shift 2
    local cc
    case "$color_name" in
        green)  cc="$GREEN" ;;
        blue)   cc="$BLUE" ;;
        yellow) cc="$YELLOW" ;;
        red)    cc="$RED" ;;
        *)      cc="$NC" ;;
    esac
    echo "" >/dev/tty
    printf "  ${cc}┃${NC} ${cc}${BOLD}%s${NC}\n" "$title" >/dev/tty
    local line
    for line in "$@"; do
        printf "  ${cc}┃${NC} %b\n" "$line" >/dev/tty
    done
}
```

- [ ] **Step 5: Add `print_hotkey_bar` helper**

```bash
# Print the bottom navigation hotkey bar
# Usage: print_hotkey_bar [deploy]
print_hotkey_bar() {
    local mode="${1:-nav}"
    echo "" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    if [[ "$mode" == "deploy" ]]; then
        printf "       ${DIM}[ %s ]  [ %s ]  [ %s ]${NC}\n" "$MSG_NAV_ENTER_DEPLOY" "$MSG_NAV_BACK" "$MSG_NAV_QUIT" >/dev/tty
    else
        printf "       ${DIM}[ %s ]  [ %s ]  [ %s ]${NC}\n" "$MSG_NAV_ENTER_NEXT" "$MSG_NAV_BACK" "$MSG_NAV_QUIT" >/dev/tty
    fi
}
```

- [ ] **Step 6: Add `print_field` helper for answered fields**

```bash
# Print an answered field line: ▸ Label    value
print_field() {
    local label="$1" value="$2"
    printf "  ${GREEN}▸${NC} %-16s %b\n" "$label" "${DIM}${value}${NC}" >/dev/tty
}
```

- [ ] **Step 7: Add `_draw_deploy_status` function**

```bash
# Draw deployment progress with checkmarks, spinner, and pending items.
# Usage: _draw_deploy_status current total steps_array_name
# steps_array: ("done:Label" "active:Label" "pending:Label" ...)
_deploy_spinner_chars=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
_deploy_spinner_idx=0

_draw_deploy_status() {
    local current="$1" total="$2"
    local -n steps_ref="$3"
    local pct=$(( current * 100 / total ))
    local bar_width=40
    local filled=$(( pct * bar_width / 100 ))
    local empty=$(( bar_width - filled ))

    # Move cursor to saved position (top of deploy area)
    printf "\033[${_deploy_area_top};1H" >/dev/tty

    # Progress bar
    local bar="" trail=""
    local i
    for (( i=0; i<filled; i++ )); do bar+="━"; done
    for (( i=0; i<empty; i++ )); do trail+="━"; done
    printf "  ${GREEN}%s${NC}${DIM}%s${NC}  ${BOLD}%d%%${NC}\033[K\n" "$bar" "$trail" "$pct" >/dev/tty
    echo -e "\033[K" >/dev/tty

    # Step list
    for step in "${steps_ref[@]}"; do
        local status="${step%%:*}"
        local label="${step#*:}"
        case "$status" in
            done)
                printf "  ${GREEN}✓${NC} %s\033[K\n" "$label" >/dev/tty
                ;;
            active)
                local sc="${_deploy_spinner_chars[$_deploy_spinner_idx]}"
                printf "  ${YELLOW}%s${NC} %s\033[K\n" "$sc" "$label" >/dev/tty
                _deploy_spinner_idx=$(( (_deploy_spinner_idx + 1) % ${#_deploy_spinner_chars[@]} ))
                ;;
            pending)
                printf "  ${DIM}○${NC} ${DIM}%s${NC}\033[K\n" "$label" >/dev/tty
                ;;
        esac
    done
    # Clear any leftover lines
    printf "\033[K" >/dev/tty
}
```

- [ ] **Step 8: Verify no syntax errors**

```bash
bash -n deploy/scripts/lib/common.sh
```
Expected: no output (success)

- [ ] **Step 9: Commit**

```bash
git add deploy/scripts/lib/common.sh
git commit -m "feat(ui): add Style C rendering helpers to common.sh"
```

---

## Chunk 2: Wizard Page Rewrite

### Task 3: Rewrite `_wizard_header` and `_wizard_page_gate` in interact.sh

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (lines 7-31)

- [ ] **Step 1: Replace `_wizard_header`**

Replace lines 7-14 (the old `_wizard_header` function) with:

```bash
_wizard_header() {
    local active_tab="$1"
    clear >/dev/tty
    print_brand_header
    print_tab_bar "$active_tab"
}
```

- [ ] **Step 2: Replace `_wizard_page_gate`**

Replace lines 17-31 (the old `_wizard_page_gate` function) with:

```bash
# Show page header + navigation gate. Echoes "back" or "continue" to stdout.
_wizard_page_gate() {
    local active_tab="$1"
    _wizard_header "$active_tab"
    if [[ "$active_tab" -gt 1 ]]; then
        printf "  ${DIM}${MSG_NAV_ENTER_NEXT}  ${MSG_NAV_BACK}  ${MSG_NAV_QUIT}${NC} " >/dev/tty
        local key
        IFS= read -rsn1 key </dev/tty
        printf "\n" >/dev/tty
        case "$key" in
            b|B) echo "back"; return ;;
            q|Q) echo "" >/dev/tty; echo "$MSG_WIZARD_CANCELLED" >/dev/tty; exit 0 ;;
        esac
    fi
    echo "continue"
}
```

- [ ] **Step 3: Verify syntax**

```bash
bash -n deploy/scripts/lib/interact.sh
```

- [ ] **Step 4: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite wizard header and page gate for Style C"
```

---

### Task 4: Rewrite `_page_basic` (Page 1)

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_basic` function, lines 36-72)

The new version collects the same inputs but re-renders the page after collection to show Style C answered fields, accent blocks, and hotkey bar.

- [ ] **Step 1: Replace `_page_basic`**

```bash
_page_basic() {
    _wizard_header 1

    # Language already selected by i18n.sh — show as answered
    print_field "Language" "${LANG_CHOICE:-en}"

    while true; do
        DOMAIN=$(ask_default "$MSG_DOMAIN_PROMPT" "${DOMAIN:-}")
        if validate_domain "$DOMAIN"; then
            break
        fi
        log_error "$MSG_ERR_DOMAIN_INVALID: $DOMAIN"
    done

    SUBDOMAIN=$(ask_default "$MSG_SUBDOMAIN_PROMPT" "${SUBDOMAIN:-corp}")
    LAUNCHPAD_DOMAIN="launchpad.${SUBDOMAIN}.${DOMAIN}"
    GITEA_DOMAIN="launchpad-gitea.${SUBDOMAIN}.${DOMAIN}"

    # Re-display domain and subdomain as answered fields (matching mockup)
    print_field "$MSG_DOMAIN_PROMPT" "$DOMAIN"
    print_field "$MSG_SUBDOMAIN_PROMPT" "$SUBDOMAIN"

    # Show generated domains in green accent block
    print_accent_block green "$MSG_DOMAIN_GEN_HEADER" \
        "${DIM}▪ ${LAUNCHPAD_DOMAIN}${NC}       ${DIM}$MSG_DOMAIN_DASHBOARD${NC}" \
        "${DIM}▪ ${GITEA_DOMAIN}${NC} ${DIM}$MSG_DOMAIN_GITEA${NC}" \
        "${DIM}▪ *.${DOMAIN}${NC}                   ${DIM}$MSG_DOMAIN_PROJECTS${NC}"

    echo "" >/dev/tty
    if ! ask_confirm "$MSG_CONFIRM" "Y"; then
        _page_basic  # Retry
        return
    fi

    IMAGE_REGISTRY=$(ask_default "$MSG_REGISTRY_PROMPT" "${IMAGE_REGISTRY}")

    # Image versions in blue accent block (collected inline)
    echo "" >/dev/tty
    printf "  ${BLUE}┃${NC} ${BLUE}${BOLD}%s${NC}\n" "$MSG_IMG_VERSIONS" >/dev/tty
    IMAGE_VERSION_API=$(ask_default "  API" "${IMAGE_VERSION_API:-${IMAGE_VERSION}}")
    IMAGE_VERSION_UI=$(ask_default "  UI" "${IMAGE_VERSION_UI:-${IMAGE_VERSION}}")
    IMAGE_VERSION_ROUTER=$(ask_default "  Router" "${IMAGE_VERSION_ROUTER:-${IMAGE_VERSION}}")
    IMAGE_VERSION_GATEWAY=$(ask_default "  Gateway" "${IMAGE_VERSION_GATEWAY:-${IMAGE_VERSION}}")
    IMAGE_VERSION_GITEA=$(ask_default "  Gitea" "${IMAGE_VERSION_GITEA:-1.25-rootless}")

    print_hotkey_bar
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/interact.sh
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_basic with Style C design"
```

---

### Task 5: Rewrite `_page_ssl` (Page 2)

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_ssl` function, lines 74-103)

- [ ] **Step 1: Replace `_page_ssl`**

```bash
_page_ssl() {
    local _gate
    _gate=$(_wizard_page_gate 2)
    [[ "$_gate" == "back" ]] && return 1 || true

    SSL_MODE=$(ask_choice "$MSG_SSL_METHOD" "1" "$MSG_SSL_LETSENCRYPT" "$MSG_SSL_SELFSIGNED" "$MSG_SSL_CUSTOM")

    case "$SSL_MODE" in
        1)  SSL_MODE="letsencrypt"
            DNS_PROVIDER=$(ask_choice "$MSG_SSL_DNS_PROVIDER" "1" \
                "$MSG_SSL_DNS_CLOUDFLARE" "$MSG_SSL_DNS_ALIYUN" "$MSG_SSL_DNS_AZURE" "$MSG_SSL_DNS_OTHER")
            case "$DNS_PROVIDER" in
                1) DNS_PROVIDER="cloudflare" ;;
                2) DNS_PROVIDER="aliyun" ;;
                3) DNS_PROVIDER="azure" ;;
                *) DNS_PROVIDER="manual" ;;
            esac
            if [[ "$DNS_PROVIDER" != "manual" ]]; then
                DNS_API_TOKEN=$(ask_default "$MSG_SSL_API_TOKEN" "")
            fi
            ;;
        2)  SSL_MODE="selfsigned" ;;
        3)  SSL_MODE="custom"
            SSL_CERT_PATH=$(ask_filepath "$MSG_SSL_CERT_PATH" "")
            SSL_KEY_PATH=$(ask_filepath "$MSG_SSL_KEY_PATH" "")
            SSL_WILDCARD_CERT_PATH=$(ask_filepath "Wildcard $MSG_SSL_CERT_PATH" "")
            SSL_WILDCARD_KEY_PATH=$(ask_filepath "Wildcard $MSG_SSL_KEY_PATH" "")
            ;;
    esac

    print_hotkey_bar
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_ssl with Style C design"
```

---

### Task 6: Rewrite `_page_database` (Page 3)

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_database` function, lines 105-135)

- [ ] **Step 1: Replace `_page_database`**

```bash
_page_database() {
    local _gate
    _gate=$(_wizard_page_gate 3)
    [[ "$_gate" == "back" ]] && return 1 || true

    DB_MODE=$(ask_choice "$MSG_DB_MODE" "1" "$MSG_DB_BUILTIN" "$MSG_DB_EXTERNAL")

    if [[ "$DB_MODE" == "2" ]]; then
        DB_MODE="external"

        # Connection URLs in yellow accent block
        print_accent_block yellow "$MSG_DB_CONN_URLS" \
            "${DIM}$MSG_DB_CONN_FORMAT${NC}"
        echo "" >/dev/tty

        DATABASE_URL=$(ask_default "$MSG_DB_URL_MAIN" "${DATABASE_URL:-}")
        MONITORING_DATABASE_URL=$(ask_default "$MSG_DB_URL_MONITORING" "${MONITORING_DATABASE_URL:-}")
        EVENTS_DATABASE_URL=$(ask_default "$MSG_DB_URL_EVENTS" "${EVENTS_DATABASE_URL:-}")
        BILLING_DATABASE_URL=$(ask_default "$MSG_DB_URL_BILLING" "${BILLING_DATABASE_URL:-}")
        STATS_DATABASE_URL=$(ask_default "$MSG_DB_URL_STATS" "${STATS_DATABASE_URL:-}")
        GATEWAY_DATABASE_URL=$(ask_default "$MSG_DB_URL_GATEWAY" "${GATEWAY_DATABASE_URL:-}")
        GITEA_DATABASE_URL=$(ask_default "$MSG_DB_URL_GITEA" "${GITEA_DATABASE_URL:-}")

        # Redis in blue accent block
        print_accent_block blue "Redis"
        REDIS_HOST=$(ask_default "$MSG_DB_REDIS_HOST" "${REDIS_HOST:-}")
        REDIS_PORT=$(ask_default "$MSG_DB_REDIS_PORT" "${REDIS_PORT:-6379}")
        REDIS_PASSWORD=$(ask_default "$MSG_DB_REDIS_PASSWORD" "")
    else
        DB_MODE="builtin"
        DB_HOST="postgres"
        DB_PORT="5432"
        REDIS_HOST="redis"
        REDIS_PORT="6379"

        # Auto-configured info in green accent block
        print_accent_block green "$MSG_DB_AUTO_CONFIGURED" \
            "${DIM}PostgreSQL  postgres:5432${NC}" \
            "${DIM}Redis       redis:6379${NC}" \
            "${DIM}$MSG_DB_AUTO_CREDS${NC}"
    fi

    print_hotkey_bar
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_database with Style C design"
```

---

### Task 7: Rewrite `_page_k8s` (Page 4)

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_k8s` function, lines 137-157)

- [ ] **Step 1: Replace `_page_k8s`**

```bash
_page_k8s() {
    local _gate
    _gate=$(_wizard_page_gate 4)
    [[ "$_gate" == "back" ]] && return 1 || true

    K8S_MODE=$(ask_choice "$MSG_K8S_MODE" "1" "$MSG_K8S_BUILTIN" "$MSG_K8S_EXTERNAL")

    if [[ "$K8S_MODE" == "2" || "$K8S_MODE" == "external" ]]; then
        K8S_MODE="external"
        K8S_KUBECONFIG_PATH=$(ask_filepath "$MSG_K8S_KUBECONFIG" "${K8S_KUBECONFIG_PATH:-~/.kube/config}")
        K8S_CONTEXT=$(ask_default "$MSG_K8S_CONTEXT" "")
        K8S_INGRESS_DOMAIN=$(ask_default "$MSG_K8S_INGRESS" "")
        STORAGE_CLASS=$(ask_default "$MSG_K8S_STORAGE_CLASS" "${STORAGE_CLASS:-standard}")
        DEFAULT_BACKEND="$K8S_INGRESS_DOMAIN"
    else
        K8S_MODE="builtin"
        K8S_KUBECONFIG_PATH=$(ask_filepath "$MSG_K8S_KUBECONFIG" "${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}")
        STORAGE_CLASS="local-path"
        DEFAULT_BACKEND="localhost"

        # Cluster info in green accent block
        print_accent_block green "$MSG_K8S_CLUSTER_INFO" \
            "${DIM}Storage Class   local-path${NC}" \
            "${DIM}Backend         localhost${NC}" \
            "${DIM}$MSG_K8S_AUTO_INSTALL${NC}"
    fi

    print_hotkey_bar
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_k8s with Style C design"
```

---

### Task 8: Rewrite `_page_advanced` (Page 5)

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_advanced` function, lines 159-277)

The logic remains the same but module sections use colored accent blocks (red for admin/credentials, yellow for SMTP/payment/AI, blue for storage/perf).

- [ ] **Step 1: Replace `_page_advanced`**

```bash
_page_advanced() {
    local _gate
    _gate=$(_wizard_page_gate 5)
    [[ "$_gate" == "back" ]] && return 1 || true

    if ! ask_confirm "$MSG_ADV_ENTER" "N"; then
        ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"
        return
    fi

    echo "" >/dev/tty
    printf "  ${DIM}%s${NC}\n\n" "$MSG_ADV_SELECT" >/dev/tty
    echo -e "    ${DIM}1)${NC} $MSG_ADV_ADMIN      ${DIM}— $MSG_ADV_ADMIN_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}2)${NC} $MSG_ADV_SMTP        ${DIM}— $MSG_ADV_SMTP_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}3)${NC} $MSG_ADV_STRIPE      ${DIM}— $MSG_ADV_STRIPE_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}4)${NC} $MSG_ADV_SSO          ${DIM}— $MSG_ADV_SSO_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}5)${NC} $MSG_ADV_AI           ${DIM}— $MSG_ADV_AI_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}6)${NC} $MSG_ADV_NOTIFY       ${DIM}— $MSG_ADV_NOTIFY_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}7)${NC} $MSG_ADV_STORAGE      ${DIM}— $MSG_ADV_STORAGE_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}8)${NC} $MSG_ADV_PERF         ${DIM}— $MSG_ADV_PERF_DESC${NC}" >/dev/tty
    echo -e "    ${DIM}9)${NC} ${BOLD}$MSG_ADV_ALL${NC}" >/dev/tty
    echo "" >/dev/tty
    local selected
    selected=$(ask_multichoice "$MSG_ADV_SELECT")

    [[ "$selected" == "9" ]] && selected="1,2,3,4,5,6,7,8"
    local sel=",${selected//[[:space:]]/},"

    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"

    # Module 1: Admin (red accent — credentials)
    if [[ "$sel" == *",1,"* ]]; then
        print_accent_block red "$MSG_ADV_ADMIN"
        ADMIN_EMAIL=$(ask_default "$MSG_ADMIN_EMAIL" "$ADMIN_EMAIL")
        ADMIN_PASSWORD=$(ask_password "$MSG_ADMIN_PASSWORD")
        [[ -z "$ADMIN_PASSWORD" ]] && ADMIN_PASSWORD=""
        printf "  ${RED}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 2: SMTP (yellow accent)
    if [[ "$sel" == *",2,"* ]]; then
        print_accent_block yellow "$MSG_ADV_SMTP"
        SMTP_HOST=$(ask_default "$MSG_SMTP_HOST" "${SMTP_HOST:-smtp.gmail.com}")
        SMTP_PORT=$(ask_default "$MSG_SMTP_PORT" "${SMTP_PORT:-587}")
        SMTP_USER=$(ask_default "$MSG_SMTP_USER" "${SMTP_USER:-}")
        SMTP_PASS=$(ask_password "$MSG_SMTP_PASS")
        EMAIL_FROM=$(ask_default "$MSG_SMTP_FROM" "noreply@${DOMAIN}")
        printf "  ${YELLOW}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 3: Stripe (yellow accent)
    if [[ "$sel" == *",3,"* ]]; then
        print_accent_block yellow "$MSG_ADV_STRIPE"
        STRIPE_SECRET_KEY=$(ask_default "Stripe Secret Key" "${STRIPE_SECRET_KEY:-}")
        STRIPE_PUBLISHABLE_KEY=$(ask_default "Stripe Publishable Key" "${STRIPE_PUBLISHABLE_KEY:-}")
        printf "  ${YELLOW}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 4: SSO (yellow accent)
    if [[ "$sel" == *",4,"* ]]; then
        print_accent_block yellow "$MSG_ADV_SSO"
        ENTRA_ENABLED="true"
        ENTRA_CLIENT_ID=$(ask_default "Entra Client ID" "${ENTRA_CLIENT_ID:-}")
        ENTRA_CLIENT_SECRET=$(ask_password "Entra Client Secret")
        ENTRA_TENANT_ID=$(ask_default "Entra Tenant ID" "${ENTRA_TENANT_ID:-}")
        printf "  ${YELLOW}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 5: AI (yellow accent)
    if [[ "$sel" == *",5,"* ]]; then
        print_accent_block yellow "$MSG_ADV_AI"
        CRS2_API_URL=$(ask_default "CRS2 API URL" "${CRS2_API_URL:-}")
        CRS2_API_KEY=$(ask_default "CRS2 API Key" "${CRS2_API_KEY:-}")
        printf "  ${YELLOW}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 6: Notifications (yellow accent)
    if [[ "$sel" == *",6,"* ]]; then
        print_accent_block yellow "$MSG_ADV_NOTIFY"
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
        printf "  ${YELLOW}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 7: Storage (blue accent)
    if [[ "$sel" == *",7,"* ]]; then
        print_accent_block blue "$MSG_ADV_STORAGE"
        STORAGE_PROVIDER=$(ask_choice "$MSG_STORAGE_PROVIDER" "1" "local" "s3" "azure")
        case "$STORAGE_PROVIDER" in
            2) STORAGE_PROVIDER="s3"
               S3_ENDPOINT=$(ask_default "S3 Endpoint" "${S3_ENDPOINT:-}")
               S3_BUCKET=$(ask_default "S3 Bucket" "${S3_BUCKET:-}")
               S3_ACCESS_KEY=$(ask_default "S3 Access Key" "${S3_ACCESS_KEY:-}")
               S3_SECRET_KEY=$(ask_password "S3 Secret Key")
               ;;
            3) STORAGE_PROVIDER="azure"
               AZURE_CONTAINER_SAS_URL=$(ask_default "Azure Container SAS URL" "${AZURE_CONTAINER_SAS_URL:-}")
               ;;
            *) STORAGE_PROVIDER="local" ;;
        esac
        printf "  ${BLUE}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    # Module 8: Performance (blue accent)
    if [[ "$sel" == *",8,"* ]]; then
        print_accent_block blue "$MSG_ADV_PERF"
        API_REPLICAS=$(ask_default "API replicas" "${API_REPLICAS:-1}")
        ROUTER_REPLICAS=$(ask_default "Router replicas" "${ROUTER_REPLICAS:-1}")
        GATEWAY_REPLICAS=$(ask_default "Gateway replicas" "${GATEWAY_REPLICAS:-1}")
        DB_CONNECTION_LIMIT=$(ask_default "DB connection limit per schema" "${DB_CONNECTION_LIMIT:-10}")
        printf "  ${BLUE}┃${NC} ${GREEN}✓${NC} ${DIM}$MSG_ADV_CONFIGURED${NC}\n" >/dev/tty
    fi

    print_hotkey_bar
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_advanced with Style C accent blocks"
```

---

### Task 9: Rewrite `_page_summary` (Page 6) and `run_wizard`

**Files:**
- Modify: `deploy/scripts/lib/interact.sh` (replace `_page_summary` function, lines 279-318, and `run_wizard`, lines 324-350)

- [ ] **Step 1: Replace `_page_summary`**

```bash
_page_summary() {
    local _gate
    _gate=$(_wizard_page_gate 6)
    [[ "$_gate" == "back" ]] && return 1 || true

    # Domains — green accent block
    print_accent_block green "$MSG_SUMMARY_DOMAINS" \
        "${DIM}$MSG_OUT_DASHBOARD   ${LAUNCHPAD_DOMAIN}${NC}" \
        "${DIM}$MSG_OUT_GITEA       ${GITEA_DOMAIN}${NC}" \
        "${DIM}$MSG_DOMAIN_PROJECTS    *.${DOMAIN}${NC}"

    # Infrastructure — blue accent block
    print_accent_block blue "$MSG_SUMMARY_INFRA" \
        "${DIM}SSL         ${SSL_MODE}${NC}" \
        "${DIM}Database    ${DB_MODE}${NC}" \
        "${DIM}Kubernetes  ${K8S_MODE}${NC}"

    # Images — yellow accent block
    local img_line1="API ${IMAGE_VERSION_API}  UI ${IMAGE_VERSION_UI}  Router ${IMAGE_VERSION_ROUTER}"
    local img_line2="Gateway ${IMAGE_VERSION_GATEWAY}  Gitea ${IMAGE_VERSION_GITEA}"
    print_accent_block yellow "$MSG_SUMMARY_IMAGES" \
        "${DIM}Registry    ${IMAGE_REGISTRY}${NC}" \
        "${DIM}${img_line1}${NC}" \
        "${DIM}${img_line2}${NC}"

    # Admin — red accent block
    local admin_lines=("${DIM}$MSG_OUT_EMAIL       ${ADMIN_EMAIL}${NC}")
    local modules=""
    [[ -n "${SMTP_HOST:-}" ]]             && modules+="SMTP "
    [[ -n "${STRIPE_SECRET_KEY:-}" ]]     && modules+="Stripe "
    [[ "${ENTRA_ENABLED:-}" == "true" ]]  && modules+="SSO "
    [[ -n "${CRS2_API_URL:-}" ]]          && modules+="AI "
    [[ -n "${TELEGRAM_BOT_TOKEN:-}" ]]    && modules+="Telegram "
    [[ "${STORAGE_PROVIDER:-local}" != "local" ]] && modules+="Storage(${STORAGE_PROVIDER}) "
    [[ -n "$modules" ]] && admin_lines+=("${DIM}$MSG_SUMMARY_MODULES     ${modules}${NC}")
    print_accent_block red "$MSG_SUMMARY_ADMIN" "${admin_lines[@]}"
}
```

- [ ] **Step 2: Replace `run_wizard` with q-key support**

Replace the `run_wizard` function and `WIZARD_TOTAL` variable:

```bash
WIZARD_TOTAL=6

run_wizard() {
    local page=1
    local pages=(_page_basic _page_ssl _page_database _page_k8s _page_advanced _page_summary)

    while [[ $page -le ${#pages[@]} ]]; do
        if ! ${pages[$((page - 1))]}; then
            [[ $page -gt 1 ]] && page=$((page - 1))
            continue
        fi

        if [[ $page -eq ${#pages[@]} ]]; then
            # Last page (summary): confirm deploy
            echo "" >/dev/tty
            printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
            echo "" >/dev/tty
            printf "  ${GREEN}◆${NC} %s ${DIM}(Y/n)${NC} " "$MSG_DEPLOY_CONFIRM" >/dev/tty
            local key
            IFS= read -rsn1 key </dev/tty
            printf "\n" >/dev/tty
            case "$key" in
                q|Q) echo "$MSG_WIZARD_CANCELLED" >/dev/tty; exit 0 ;;
                b|B) page=$((page - 1)); continue ;;
                n|N) page=$((page - 1)); continue ;;
                *)   return 0 ;;  # Enter or Y = deploy
            esac
        fi

        page=$((page + 1))
    done
}
```

- [ ] **Step 3: Verify full file syntax**

```bash
bash -n deploy/scripts/lib/interact.sh
```

- [ ] **Step 4: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat(ui): rewrite _page_summary and run_wizard with Style C"
```

---

## Chunk 3: Deploy Progress & Result Display

### Task 10: Rewrite `deploy_services()` progress display

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh` (replace `deploy_services` function, lines 53-120)

The new version uses the `_draw_deploy_status()` function to show checkmarks, spinners, and pending items instead of a single-line progress bar.

- [ ] **Step 1: Replace `deploy_services`**

```bash
deploy_services() {
    # Print brand header with "Deploying..." subtitle
    clear >/dev/tty 2>/dev/null || true
    print_brand_header "$MSG_DEPLOY_DEPLOYING"

    # Define all deploy steps (global — nested functions need access)
    _ds_steps=(
        "pending:$MSG_DEPLOY_PROGRESS_INFRA"
        "pending:$MSG_DEPLOY_PROGRESS_PG"
        "pending:$MSG_DEPLOY_PROGRESS_REDIS"
        "pending:$MSG_DEPLOY_PROGRESS_DB"
        "pending:$MSG_DEPLOY_PROGRESS_GITEA"
        "pending:$MSG_DEPLOY_PROGRESS_GITEA_BOOT"
        "pending:$MSG_DEPLOY_PROGRESS_API"
        "pending:$MSG_DEPLOY_PROGRESS_UI"
        "pending:$MSG_DEPLOY_PROGRESS_ROUTER"
        "pending:$MSG_DEPLOY_PROGRESS_TEMPLATES"
        "pending:$MSG_DEPLOY_PROGRESS_NGINX"
        "pending:$MSG_DEPLOY_PROGRESS_CLUSTER"
    )
    _ds_total=${#_ds_steps[@]}
    _ds_current=0

    # Save cursor position for redraw area
    # Brand header takes ~5 lines, then our content starts
    _deploy_area_top=6
    printf "\033[?25l" >/dev/tty  # Hide cursor

    _update_step() {
        local idx="$1" status="$2"
        local label="${_ds_steps[$idx]#*:}"
        _ds_steps[$idx]="${status}:${label}"
    }

    _redraw() {
        _draw_deploy_status "$_ds_current" "$_ds_total" _ds_steps
    }

    _complete_step() {
        local idx="$1"
        _update_step "$idx" "done"
        _ds_current=$((_ds_current + 1))
        # Mark next step as active if exists
        local next=$((idx + 1))
        if [[ $next -lt $_ds_total ]]; then
            _update_step "$next" "active"
        fi
        _redraw
    }

    _fail_with_cursor() {
        printf "\033[?25h" >/dev/tty
        deploy_fail "$1"
    }

    # Mark first step as active
    _update_step 0 "active"
    _redraw

    # Phase 1: Infrastructure (PostgreSQL & Redis)
    if [[ "$DB_MODE" == "builtin" ]]; then
        $COMPOSE_CMD up -d postgres redis 2>&1 | verbose_filter
        _complete_step 0  # Infrastructure started
        wait_for_healthy postgres 30 || _fail_with_cursor "infrastructure (postgres)"
        _complete_step 1  # PostgreSQL ready
        wait_for_healthy redis 15 || _fail_with_cursor "infrastructure (redis)"
        _complete_step 2  # Redis ready
    else
        _complete_step 0  # Infrastructure (external)
        _complete_step 1  # External DB
        _complete_step 2  # External Redis
    fi

    # Phase 2: Database init + migration
    init_database || _fail_with_cursor "database initialization"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (main)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/monitoring.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (monitoring)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-billing.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (billing)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-events.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (events)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-stats.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (stats)"
    _complete_step 3  # Database migrated

    # Phase 3: Gitea
    $COMPOSE_CMD up -d gitea 2>&1 | verbose_filter
    wait_for_healthy gitea 90 || _fail_with_cursor "gitea"
    _complete_step 4  # Gitea healthy

    bootstrap_gitea || _fail_with_cursor "gitea bootstrap"
    _complete_step 5  # Gitea bootstrapped

    # Phase 4: Application services
    $COMPOSE_CMD up -d api ui router cron backup-worker gateway 2>&1 | verbose_filter
    wait_for_healthy api 90 || _fail_with_cursor "application services (api)"
    _complete_step 6  # API ready
    wait_for_healthy ui 60 || _fail_with_cursor "application services (ui)"
    _complete_step 7  # UI ready
    wait_for_healthy router 60 || _fail_with_cursor "application services (router)"
    _complete_step 8  # Router ready

    # Phase 4b: Templates
    import_templates || log_warn "Template import failed — you can retry with: ./setup.sh --import-templates"
    _complete_step 9  # Templates

    # Phase 5: Nginx
    $COMPOSE_CMD up -d nginx 2>&1 | verbose_filter
    wait_for_healthy nginx 30 || _fail_with_cursor "nginx"
    setup_hosts
    _complete_step 10  # Nginx

    # Note: Step 11 (Cluster registration) remains pending in the progress display.
    # It is handled by register_cluster() in setup.sh after deploy_services() returns.
    # The screen is then replaced by show_result() which draws its own complete view.

    printf "\033[?25h" >/dev/tty  # Show cursor
    echo "" >/dev/tty
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/deploy.sh
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat(ui): rewrite deploy_services with Style C progress display"
```

---

### Task 11: Rewrite `show_result()`

**Files:**
- Modify: `deploy/scripts/lib/deploy.sh` (replace `show_result` function, lines 349-387)

- [ ] **Step 1: Replace `show_result`**

```bash
show_result() {
    local ip
    ip=$(detect_internal_ip 2>/dev/null || echo "x.x.x.x")

    clear >/dev/tty 2>/dev/null || true
    print_brand_header "${GREEN}${BOLD}✓ $MSG_DEPLOY_COMPLETE${NC}"

    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty

    # Access URLs — green accent block
    print_accent_block green "$MSG_OUT_URLS" \
        "${DIM}$MSG_OUT_DASHBOARD  https://${LAUNCHPAD_DOMAIN}${NC}" \
        "${DIM}$MSG_OUT_GITEA      https://${GITEA_DOMAIN}${NC}" \
        "${DIM}$MSG_OUT_ADMIN      https://${LAUNCHPAD_DOMAIN}/admin${NC}"

    # Credentials — red accent block
    print_accent_block red "$MSG_OUT_CREDENTIALS" \
        "${DIM}$MSG_OUT_EMAIL      ${ADMIN_EMAIL}${NC}" \
        "${DIM}$MSG_OUT_PASSWORD   ${ADMIN_PASSWORD}${NC}"

    # K8s — blue accent block
    local k8s_status
    if [[ "${CLUSTER_REGISTERED:-false}" == "true" ]]; then
        k8s_status="${GREEN}✓${NC} ${DIM}Registered${NC}"
    else
        k8s_status="${YELLOW}⚠ Not registered${NC}"
    fi
    print_accent_block blue "$MSG_OUT_K8S" \
        "${DIM}$MSG_OUT_MODE       ${K8S_MODE}${NC}" \
        "$k8s_status"

    # DNS — yellow accent block
    print_accent_block yellow "$MSG_OUT_DNS" \
        "${DIM}$MSG_OUT_DNS_MSG ($ip):${NC}" \
        "${DIM}├─ ${LAUNCHPAD_DOMAIN}${NC}" \
        "${DIM}├─ ${GITEA_DOMAIN}${NC}" \
        "${DIM}└─ *.${DOMAIN}${NC}"

    echo "" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    printf "  ${DIM}$MSG_OUT_CONFIG:  generated/${NC}\n" >/dev/tty
    printf "  ${DIM}$MSG_OUT_LOGS:    docker compose -f generated/docker-compose.yml logs -f${NC}\n" >/dev/tty
    printf "  ${DIM}$MSG_OUT_RECONFIG:  ./setup.sh --reconfigure${NC}\n" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/deploy.sh
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat(ui): rewrite show_result with Style C accent blocks"
```

---

### Task 12: Restyle `select_language` with brand header

**Files:**
- Modify: `deploy/scripts/lib/i18n.sh`

- [ ] **Step 1: Add brand header to language selection**

Replace the first lines of `select_language()` (before the `printf "  ${CYAN}◆${NC}" line`) to clear the screen and show the brand header first. Since MSG_BRAND_SUBTITLE isn't loaded yet (no lang file sourced), use hardcoded text:

```bash
select_language() {
    local selected=0 count=2
    local options=("English" "中文")

    clear >/dev/tty 2>/dev/null || true
    # Brand header before language is loaded — use hardcoded subtitle
    echo "" >/dev/tty
    print_centered "\033[0;31m■\033[0m \033[1;33m■\033[0m \033[0;32m■\033[0m \033[0;34m■\033[0m"
    print_centered "${BOLD}AniLaunchpad${NC}"
    echo "" >/dev/tty

    printf "  ${CYAN}◆${NC} Language / 语言选择\n"
```

The rest of the function stays the same.

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/i18n.sh
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/i18n.sh
git commit -m "feat(ui): add brand header to language selection screen"
```

---

## Chunk 4: Integration & E2E Testing

### Task 13: Update `setup.sh` banner to use brand header

**Files:**
- Modify: `deploy/setup.sh` (line 67)

- [ ] **Step 1: Replace `print_banner` call**

In setup.sh line 67, replace:
```bash
        print_banner "AniLaunchpad Setup" "$SETUP_VERSION"
```
with:
```bash
        # Brand header shown by wizard pages; skip old banner for interactive mode
        if [[ -n "$CONFIG_FILE" ]]; then
            print_banner "AniLaunchpad Setup" "$SETUP_VERSION"
        fi
```

This avoids showing the old-style banner before the wizard appears (the wizard pages each render their own brand header). Non-interactive mode still shows the old banner since there's no wizard UI.

**Source order note:** `common.sh` is sourced at line 12 of `setup.sh` (startup), so `print_banner` is available at line 67. The new Style C helpers (`print_centered`, `print_brand_header`, etc.) are appended to `common.sh`, so they're also available from line 12 onward — including when `select_language()` runs at line 88.

- [ ] **Step 2: Commit**

```bash
git add deploy/setup.sh
git commit -m "feat(ui): skip old banner in interactive wizard mode"
```

---

### Task 14: Sync to remote and run E2E tests

**Files:**
- Uses: `tests/run.sh` or manual SSH sync + test

- [ ] **Step 1: Sync all modified files to remote server**

```bash
REMOTE="root@10.233.201.133"
REMOTE_DIR="~/workspace/launchpad-deploy"

# Sync lib scripts
rsync -avz deploy/scripts/lib/common.sh deploy/scripts/lib/interact.sh \
    deploy/scripts/lib/deploy.sh deploy/scripts/lib/i18n.sh \
    ${REMOTE}:${REMOTE_DIR}/deploy/scripts/lib/

# Sync lang files
rsync -avz deploy/scripts/lang/en.sh deploy/scripts/lang/zh.sh \
    ${REMOTE}:${REMOTE_DIR}/deploy/scripts/lang/

# Sync setup.sh
rsync -avz deploy/setup.sh ${REMOTE}:${REMOTE_DIR}/deploy/setup.sh

# Also sync to the root scripts/lib/ path (dual path issue, see pitfalls #9)
rsync -avz deploy/scripts/lib/common.sh deploy/scripts/lib/interact.sh \
    deploy/scripts/lib/deploy.sh deploy/scripts/lib/i18n.sh \
    ${REMOTE}:${REMOTE_DIR}/scripts/lib/
rsync -avz deploy/scripts/lang/en.sh deploy/scripts/lang/zh.sh \
    ${REMOTE}:${REMOTE_DIR}/scripts/lang/
```

- [ ] **Step 2: Run the automated E2E test suite**

```bash
./tests/run.sh
```

Expected: All tests pass. The tests use `--config` (non-interactive) so the visual wizard changes don't affect them. But verify no syntax errors break sourcing.

- [ ] **Step 3: Manual visual verification**

SSH to the remote server and run the interactive wizard:
```bash
ssh root@10.233.201.133
cd ~/workspace/launchpad-deploy
./deploy/setup.sh
```

Verify:
- Brand header with 4 colored squares appears
- Tab bar shows 6 tabs with active tab highlighted green
- Each page shows correct accent blocks with colored left borders
- Navigation: Enter advances, b goes back, q exits
- Deploy progress shows checkmarks, spinner, pending items
- Final result screen uses accent blocks

- [ ] **Step 4: Commit any fixes**

If any visual or syntax issues found during testing, fix and commit.
