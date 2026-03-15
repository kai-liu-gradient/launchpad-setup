#!/bin/bash
# Interactive configuration collection

collect_basic_config() {
    # Step 1: Domain
    log_step "1/6" "$MSG_STEP_BASIC"

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

    echo ""
    echo "  $MSG_DOMAIN_CONFIRM:"
    log_ok "${LAUNCHPAD_DOMAIN}  ($MSG_DOMAIN_DASHBOARD)"
    log_ok "${GITEA_DOMAIN}  ($MSG_DOMAIN_GITEA)"
    log_ok "*.${DOMAIN}  ($MSG_DOMAIN_PROJECTS)"
    echo ""

    if ! ask_confirm "$MSG_CONFIRM" "Y"; then
        collect_basic_config  # Retry
        return
    fi

    # Image registry and version
    IMAGE_REGISTRY=$(ask_default "$MSG_REGISTRY_PROMPT" "${IMAGE_REGISTRY}")
    IMAGE_VERSION=$(ask_default "$MSG_VERSION_PROMPT" "${IMAGE_VERSION}")

    # Step 2: SSL
    log_step "2/6" "$MSG_STEP_SSL"
    local ssl_default="1"
    [[ "$PLATFORM" == "Darwin" ]] && ssl_default="2"
    SSL_MODE=$(ask_choice "$MSG_SSL_METHOD" "$ssl_default" "$MSG_SSL_LETSENCRYPT" "$MSG_SSL_SELFSIGNED" "$MSG_SSL_CUSTOM")

    case "$SSL_MODE" in
        1)  # Let's Encrypt
            SSL_MODE="letsencrypt"
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
        2)  # Self-signed
            SSL_MODE="selfsigned"
            ;;
        3)  # Custom cert files
            SSL_MODE="custom"
            SSL_CERT_PATH=$(ask_filepath "$MSG_SSL_CERT_PATH" "")
            SSL_KEY_PATH=$(ask_filepath "$MSG_SSL_KEY_PATH" "")
            SSL_WILDCARD_CERT_PATH=$(ask_filepath "Wildcard $MSG_SSL_CERT_PATH" "")
            SSL_WILDCARD_KEY_PATH=$(ask_filepath "Wildcard $MSG_SSL_KEY_PATH" "")
            ;;
    esac

    # Step 3: Database
    log_step "3/6" "$MSG_STEP_DB"
    DB_MODE=$(ask_choice "$MSG_DB_MODE" "1" "$MSG_DB_BUILTIN" "$MSG_DB_EXTERNAL")

    if [[ "$DB_MODE" == "2" ]]; then
        DB_MODE="external"
        echo ""
        echo "  $MSG_DB_EXT_HINT"
        echo ""
        DATABASE_URL=$(ask_default "$MSG_DB_URL_MAIN" "${DATABASE_URL:-}")
        MONITORING_DATABASE_URL=$(ask_default "$MSG_DB_URL_MONITORING" "${MONITORING_DATABASE_URL:-}")
        EVENTS_DATABASE_URL=$(ask_default "$MSG_DB_URL_EVENTS" "${EVENTS_DATABASE_URL:-}")
        BILLING_DATABASE_URL=$(ask_default "$MSG_DB_URL_BILLING" "${BILLING_DATABASE_URL:-}")
        STATS_DATABASE_URL=$(ask_default "$MSG_DB_URL_STATS" "${STATS_DATABASE_URL:-}")
        GATEWAY_DATABASE_URL=$(ask_default "$MSG_DB_URL_GATEWAY" "${GATEWAY_DATABASE_URL:-}")
        GITEA_DATABASE_URL=$(ask_default "$MSG_DB_URL_GITEA" "${GITEA_DATABASE_URL:-}")
        echo ""
        REDIS_HOST=$(ask_default "$MSG_DB_REDIS_HOST" "${REDIS_HOST:-}")
        REDIS_PORT=$(ask_default "$MSG_DB_REDIS_PORT" "${REDIS_PORT:-6379}")
        REDIS_PASSWORD=$(ask_default "$MSG_DB_REDIS_PASSWORD" "")
    else
        DB_MODE="builtin"
        DB_HOST="postgres"
        DB_PORT="5432"
        REDIS_HOST="redis"
        REDIS_PORT="6379"
    fi

    # Step 4: K8s
    log_step "4/6" "$MSG_STEP_K8S"

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
        STORAGE_CLASS="local-path"
        DEFAULT_BACKEND="localhost"
    fi
}

collect_advanced_config() {
    log_step "5/6" "$MSG_STEP_ADVANCED"

    if ! ask_confirm "$MSG_ADV_ENTER" "N"; then
        # Set defaults for skipped modules
        ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"
        return
    fi

    echo ""
    echo -e "    ${DIM}1)${NC} $MSG_ADV_ADMIN ${DIM}- $MSG_ADV_ADMIN_DESC${NC}"
    echo -e "    ${DIM}2)${NC} $MSG_ADV_SMTP ${DIM}- $MSG_ADV_SMTP_DESC${NC}"
    echo -e "    ${DIM}3)${NC} $MSG_ADV_STRIPE ${DIM}- $MSG_ADV_STRIPE_DESC${NC}"
    echo -e "    ${DIM}4)${NC} $MSG_ADV_SSO ${DIM}- $MSG_ADV_SSO_DESC${NC}"
    echo -e "    ${DIM}5)${NC} $MSG_ADV_AI ${DIM}- $MSG_ADV_AI_DESC${NC}"
    echo -e "    ${DIM}6)${NC} $MSG_ADV_NOTIFY ${DIM}- $MSG_ADV_NOTIFY_DESC${NC}"
    echo -e "    ${DIM}7)${NC} $MSG_ADV_STORAGE ${DIM}- $MSG_ADV_STORAGE_DESC${NC}"
    echo -e "    ${DIM}8)${NC} $MSG_ADV_PERF ${DIM}- $MSG_ADV_PERF_DESC${NC}"
    echo -e "    ${DIM}9)${NC} ${BOLD}$MSG_ADV_ALL${NC}"
    echo ""
    local selected
    selected=$(ask_multichoice "$MSG_ADV_SELECT")

    # Parse selection — use comma-delimited word-boundary matching
    [[ "$selected" == "9" ]] && selected="1,2,3,4,5,6,7,8"
    # Convert to comma-delimited with leading/trailing commas for safe matching
    local sel=",${selected//[[:space:]]/},"

    # Set defaults
    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"

    # Module 1: Admin
    if [[ "$sel" == *",1,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_ADMIN ──"
        ADMIN_EMAIL=$(ask_default "$MSG_ADMIN_EMAIL" "$ADMIN_EMAIL")
        ADMIN_PASSWORD=$(ask_password "$MSG_ADMIN_PASSWORD")
        [[ -z "$ADMIN_PASSWORD" ]] && ADMIN_PASSWORD=""  # Will be auto-generated in secrets.sh
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 2: SMTP
    if [[ "$sel" == *",2,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_SMTP ──"
        SMTP_HOST=$(ask_default "$MSG_SMTP_HOST" "${SMTP_HOST:-smtp.gmail.com}")
        SMTP_PORT=$(ask_default "$MSG_SMTP_PORT" "${SMTP_PORT:-587}")
        SMTP_USER=$(ask_default "$MSG_SMTP_USER" "${SMTP_USER:-}")
        SMTP_PASS=$(ask_password "$MSG_SMTP_PASS")
        EMAIL_FROM=$(ask_default "$MSG_SMTP_FROM" "noreply@${DOMAIN}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 3: Stripe
    if [[ "$sel" == *",3,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_STRIPE ──"
        STRIPE_SECRET_KEY=$(ask_default "Stripe Secret Key" "${STRIPE_SECRET_KEY:-}")
        STRIPE_PUBLISHABLE_KEY=$(ask_default "Stripe Publishable Key" "${STRIPE_PUBLISHABLE_KEY:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 4: SSO
    if [[ "$sel" == *",4,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_SSO ──"
        ENTRA_ENABLED="true"
        ENTRA_CLIENT_ID=$(ask_default "Entra Client ID" "${ENTRA_CLIENT_ID:-}")
        ENTRA_CLIENT_SECRET=$(ask_password "Entra Client Secret")
        ENTRA_TENANT_ID=$(ask_default "Entra Tenant ID" "${ENTRA_TENANT_ID:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 5: AI
    if [[ "$sel" == *",5,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_AI ──"
        CRS2_API_URL=$(ask_default "CRS2 API URL" "${CRS2_API_URL:-}")
        CRS2_API_KEY=$(ask_default "CRS2 API Key" "${CRS2_API_KEY:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 6: Notifications
    if [[ "$sel" == *",6,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_NOTIFY ──"
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 7: Storage
    if [[ "$sel" == *",7,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_STORAGE ──"
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
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 8: Performance
    if [[ "$sel" == *",8,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_PERF ──"
        API_REPLICAS=$(ask_default "API replicas" "${API_REPLICAS:-1}")
        DB_CONNECTION_LIMIT=$(ask_default "DB connection limit per container" "${DB_CONNECTION_LIMIT:-10}")

        echo ""
        echo "  ── $MSG_IMG_UNIFIED: $IMAGE_VERSION ──"
        echo "  $MSG_IMG_OVERRIDE:"
        IMAGE_VERSION_API=$(ask_default "  API" "${IMAGE_VERSION_API:-$IMAGE_VERSION}")
        IMAGE_VERSION_UI=$(ask_default "  UI" "${IMAGE_VERSION_UI:-$IMAGE_VERSION}")
        IMAGE_VERSION_ROUTER=$(ask_default "  Router" "${IMAGE_VERSION_ROUTER:-$IMAGE_VERSION}")
        IMAGE_VERSION_GATEWAY=$(ask_default "  Gateway" "${IMAGE_VERSION_GATEWAY:-$IMAGE_VERSION}")
        # Gitea version managed in versions.conf (Docker Hub image, not IMAGE_REGISTRY)
        log_ok "$MSG_ADV_CONFIGURED"
    fi
}

show_summary() {
    log_step "6/6" "$MSG_STEP_SUMMARY"

    echo "  ┌──────────────────────────────────────┐"
    printf "  │ %-14s %-22s │\n" "Domain:" "$DOMAIN"
    printf "  │ %-14s %-22s │\n" "Dashboard:" "$LAUNCHPAD_DOMAIN"
    printf "  │ %-14s %-22s │\n" "SSL:" "$SSL_MODE"
    printf "  │ %-14s %-22s │\n" "Database:" "$DB_MODE"
    printf "  │ %-14s %-22s │\n" "K8s:" "$K8S_MODE"
    printf "  │ %-14s %-22s │\n" "Registry:" "$IMAGE_REGISTRY"
    printf "  │ %-14s %-22s │\n" "Version:" "$IMAGE_VERSION"
    echo "  └──────────────────────────────────────┘"
}

save_config() {
    mkdir -p "${DEPLOY_DIR}/generated"
    cat > "${DEPLOY_DIR}/generated/.setup.conf" <<CONF
# AniLaunchpad Setup Configuration — auto-generated, do not edit manually
# Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

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
STORAGE_PROVIDER="${STORAGE_PROVIDER:-local}"
S3_ENDPOINT="${S3_ENDPOINT:-}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
AZURE_CONTAINER_SAS_URL="${AZURE_CONTAINER_SAS_URL:-}"
API_REPLICAS="${API_REPLICAS:-1}"
DB_CONNECTION_LIMIT="${DB_CONNECTION_LIMIT:-10}"
CONF
    log_ok "Configuration saved to generated/.setup.conf"
}
