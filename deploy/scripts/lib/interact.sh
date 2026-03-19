#!/bin/bash
# Interactive configuration collection — page-based wizard with back/forward navigation

# ─── Page rendering helpers ───────────────────────────────────────────

# Clear screen and draw page header with brand + tabs
_wizard_header() {
    local active_tab="$1"
    clear >/dev/tty
    print_brand_header
    print_tab_bar "$active_tab"
}

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

# ─── Page functions ───────────────────────────────────────────────────
# Each page: collects config, then redraws with a summary of answered values.

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
        TELEGRAM_BOT_USERNAME=$(ask_default "Telegram Bot Username" "${TELEGRAM_BOT_USERNAME:-}")
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

# ─── Wizard loop ──────────────────────────────────────────────────────

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

# ─── Legacy API (called by setup.sh) ─────────────────────────────────

collect_basic_config()    { :; }  # no-op, wizard handles everything
collect_advanced_config() { :; }
show_summary()            { :; }

save_config() {
    mkdir -p "${DEPLOY_DIR}/generated"
    cat > "${DEPLOY_DIR}/generated/.setup.conf" <<CONF
# AniLaunchpad Setup Configuration — auto-generated, do not edit manually
# Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

LANG_CHOICE="${LANG_CHOICE:-en}"
DOMAIN="${DOMAIN}"
SUBDOMAIN="${SUBDOMAIN}"
LAUNCHPAD_DOMAIN="${LAUNCHPAD_DOMAIN}"
GITEA_DOMAIN="${GITEA_DOMAIN}"
IMAGE_REGISTRY="${IMAGE_REGISTRY}"
IMAGE_VERSION="${IMAGE_VERSION}"
IMAGE_VERSION_API="${IMAGE_VERSION_API:-}"
IMAGE_VERSION_UI="${IMAGE_VERSION_UI:-}"
IMAGE_VERSION_ROUTER="${IMAGE_VERSION_ROUTER:-}"
IMAGE_VERSION_GATEWAY="${IMAGE_VERSION_GATEWAY:-}"
IMAGE_VERSION_GITEA="${IMAGE_VERSION_GITEA:-}"
SSL_MODE="${SSL_MODE}"
SSL_CERT_PATH="${SSL_CERT_PATH:-}"
SSL_KEY_PATH="${SSL_KEY_PATH:-}"
SSL_WILDCARD_CERT_PATH="${SSL_WILDCARD_CERT_PATH:-}"
SSL_WILDCARD_KEY_PATH="${SSL_WILDCARD_KEY_PATH:-}"
DNS_PROVIDER="${DNS_PROVIDER:-}"
DNS_API_TOKEN="${DNS_API_TOKEN:-}"
DB_MODE="${DB_MODE}"
DATABASE_URL="${DATABASE_URL:-}"
MONITORING_DATABASE_URL="${MONITORING_DATABASE_URL:-}"
EVENTS_DATABASE_URL="${EVENTS_DATABASE_URL:-}"
BILLING_DATABASE_URL="${BILLING_DATABASE_URL:-}"
STATS_DATABASE_URL="${STATS_DATABASE_URL:-}"
GATEWAY_DATABASE_URL="${GATEWAY_DATABASE_URL:-}"
GITEA_DATABASE_URL="${GITEA_DATABASE_URL:-}"
REDIS_HOST="${REDIS_HOST:-}"
REDIS_PORT="${REDIS_PORT:-}"
K8S_MODE="${K8S_MODE}"
K8S_KUBECONFIG_PATH="${K8S_KUBECONFIG_PATH:-}"
K8S_CONTEXT="${K8S_CONTEXT:-}"
K8S_INGRESS_DOMAIN="${K8S_INGRESS_DOMAIN:-}"
STORAGE_CLASS="${STORAGE_CLASS}"
DEFAULT_BACKEND="${DEFAULT_BACKEND}"
ADMIN_EMAIL="${ADMIN_EMAIL}"
SMTP_HOST="${SMTP_HOST:-}"
SMTP_PORT="${SMTP_PORT:-}"
SMTP_USER="${SMTP_USER:-}"
SMTP_PASS="${SMTP_PASS:-}"
EMAIL_FROM="${EMAIL_FROM:-}"
STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_PUBLISHABLE_KEY="${STRIPE_PUBLISHABLE_KEY:-}"
ENTRA_ENABLED="${ENTRA_ENABLED:-false}"
ENTRA_CLIENT_ID="${ENTRA_CLIENT_ID:-}"
ENTRA_CLIENT_SECRET="${ENTRA_CLIENT_SECRET:-}"
ENTRA_TENANT_ID="${ENTRA_TENANT_ID:-}"
CRS2_API_URL="${CRS2_API_URL:-}"
CRS2_API_KEY="${CRS2_API_KEY:-}"
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_BOT_USERNAME="${TELEGRAM_BOT_USERNAME:-}"
STORAGE_PROVIDER="${STORAGE_PROVIDER:-local}"
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
AZURE_CONTAINER_SAS_URL="${AZURE_CONTAINER_SAS_URL:-}"
API_REPLICAS="${API_REPLICAS:-1}"
ROUTER_REPLICAS="${ROUTER_REPLICAS:-1}"
GATEWAY_REPLICAS="${GATEWAY_REPLICAS:-1}"
DB_CONNECTION_LIMIT="${DB_CONNECTION_LIMIT:-10}"
CONF
    log_done "Configuration saved to generated/.setup.conf"
}
