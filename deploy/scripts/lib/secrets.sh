#!/bin/bash
# Secret and password generation

generate_secrets() {
    log_info "Generating secrets and passwords..."

    # JWT
    JWT_SECRET=$(openssl rand -hex 32)
    JWT_REFRESH_SECRET=$(openssl rand -hex 32)
    SESSION_SECRET=$(openssl rand -hex 32)

    # Encryption keys (exactly 32 chars each)
    ENCRYPTION_KEY=$(openssl rand -hex 16)
    CLAUDE_CREDENTIALS_ENCRYPTION_KEY=$(openssl rand -hex 16)
    SSH_KEY_ENCRYPTION_SECRET=$(openssl rand -hex 16)

    # Zero Trust RSA keypair
    local zt_private_pem zt_public_pem
    zt_private_pem=$(openssl genrsa 2048 2>/dev/null)
    zt_public_pem=$(echo "$zt_private_pem" | openssl rsa -pubout 2>/dev/null)
    ZT_PRIVATE_KEY=$(echo "$zt_private_pem" | base64 -w 0 2>/dev/null || echo "$zt_private_pem" | base64)
    ZT_PUBLIC_KEY=$(echo "$zt_public_pem" | base64 -w 0 2>/dev/null || echo "$zt_public_pem" | base64)

    # Database passwords
    DB_PASSWORD_MAIN=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_MONITORING=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_EVENTS=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_BILLING=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_STATS=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_GATEWAY=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    GITEA_DB_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    POSTGRES_SUPERUSER_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)

    # Redis
    REDIS_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)

    # Gateway
    ANI_CODE_GATEWAY_API_KEY=$(openssl rand -hex 16)
    LAUNCHPAD_INTERNAL_SECRET=$(openssl rand -hex 32)

    # Admin password (if not already set by user)
    if [[ -z "${ADMIN_PASSWORD:-}" ]]; then
        ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '=/+' | head -c 16)
        ADMIN_PASSWORD_GENERATED=true
    fi

    log_ok "Secrets generated"
}

# Save generated secrets to a separate file (not in .setup.conf for security separation)
save_secrets() {
    local secrets_file="${DEPLOY_DIR}/generated/.secrets"
    chmod 600 "$secrets_file" 2>/dev/null || true
    cat > "$secrets_file" <<SECRETS
# Auto-generated secrets — DO NOT COMMIT TO VERSION CONTROL
JWT_SECRET="${JWT_SECRET}"
JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET}"
SESSION_SECRET="${SESSION_SECRET}"
ENCRYPTION_KEY="${ENCRYPTION_KEY}"
CLAUDE_CREDENTIALS_ENCRYPTION_KEY="${CLAUDE_CREDENTIALS_ENCRYPTION_KEY}"
SSH_KEY_ENCRYPTION_SECRET="${SSH_KEY_ENCRYPTION_SECRET}"
ZT_PRIVATE_KEY="${ZT_PRIVATE_KEY}"
ZT_PUBLIC_KEY="${ZT_PUBLIC_KEY}"
DB_PASSWORD_MAIN="${DB_PASSWORD_MAIN}"
DB_PASSWORD_MONITORING="${DB_PASSWORD_MONITORING}"
DB_PASSWORD_EVENTS="${DB_PASSWORD_EVENTS}"
DB_PASSWORD_BILLING="${DB_PASSWORD_BILLING}"
DB_PASSWORD_STATS="${DB_PASSWORD_STATS}"
DB_PASSWORD_GATEWAY="${DB_PASSWORD_GATEWAY}"
GITEA_DB_PASSWORD="${GITEA_DB_PASSWORD}"
POSTGRES_SUPERUSER_PASSWORD="${POSTGRES_SUPERUSER_PASSWORD}"
REDIS_PASSWORD="${REDIS_PASSWORD}"
ANI_CODE_GATEWAY_API_KEY="${ANI_CODE_GATEWAY_API_KEY}"
LAUNCHPAD_INTERNAL_SECRET="${LAUNCHPAD_INTERNAL_SECRET}"
ADMIN_PASSWORD="${ADMIN_PASSWORD}"
SECRETS
    chmod 600 "$secrets_file"
}

# Load previously generated secrets (for --resume, --upgrade, --reconfigure)
load_secrets() {
    local secrets_file="${DEPLOY_DIR}/generated/.secrets"
    if [[ -f "$secrets_file" ]]; then
        source "$secrets_file"
        return 0
    fi
    return 1
}
