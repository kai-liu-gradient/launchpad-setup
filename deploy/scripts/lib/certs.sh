#!/bin/bash
# SSL certificate management

setup_certificates() {
    log_info "Setting up SSL certificates..."

    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"
    mkdir -p "$cert_dir"

    case "$SSL_MODE" in
        letsencrypt) setup_letsencrypt ;;
        selfsigned)  setup_selfsigned_certs ;;
        custom)      setup_custom_certs ;;
    esac

    log_ok "SSL certificates configured"
}

setup_letsencrypt() {
    # Install acme.sh if not present
    if ! command -v acme.sh &>/dev/null && [[ ! -f ~/.acme.sh/acme.sh ]]; then
        log_info "Installing acme.sh..."
        curl -fsSL https://get.acme.sh | sh -s email="${ADMIN_EMAIL}" 2>/dev/null
    fi
    local ACME="${HOME}/.acme.sh/acme.sh"

    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local reload_cmd="docker compose -f ${compose_file} exec nginx nginx -s reload"

    # Set DNS API credentials based on provider
    case "$DNS_PROVIDER" in
        cloudflare)
            export CF_Token="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_cf"
            ;;
        aliyun)
            export Ali_Key="${DNS_API_KEY:-}"
            export Ali_Secret="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_ali"
            ;;
        azure)
            export AZUREDNS_SUBSCRIPTIONID="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_azure"
            ;;
        manual)
            local dns_flag="--dns --yes-I-know-dns-manual-mode-enough-go-ahead-please"
            ;;
    esac

    # Issue wildcard cert (covers *.domain and the domain itself)
    log_info "Issuing wildcard certificate for *.${DOMAIN}..."
    local issue_rc=0
    $ACME --issue $dns_flag \
        -d "*.${DOMAIN}" \
        -d "${DOMAIN}" \
        -d "${LAUNCHPAD_DOMAIN}" \
        -d "${GITEA_DOMAIN}" \
        --keylength ec-256 2>/dev/null || issue_rc=$?

    # Exit code 2 = cert already issued/renewed (skip), 0 = success, other = error
    if [[ "$issue_rc" -ne 0 && "$issue_rc" -ne 2 ]]; then
        log_error "Certificate issuance failed (exit code: $issue_rc). Check DNS provider credentials."
        return 1
    fi

    # Install cert — use consistent naming for both LE and custom paths
    $ACME --install-cert -d "*.${DOMAIN}" \
        --key-file "${cert_dir}/privkey.pem" \
        --fullchain-file "${cert_dir}/fullchain.pem" \
        --reloadcmd "$reload_cmd" 2>/dev/null

    # Copy as wildcard certs (separate files — symlinks break inside Docker)
    rm -f "${cert_dir}/wildcard-fullchain.pem" "${cert_dir}/wildcard-privkey.pem"
    cp "${cert_dir}/fullchain.pem" "${cert_dir}/wildcard-fullchain.pem"
    cp "${cert_dir}/privkey.pem" "${cert_dir}/wildcard-privkey.pem"
}

setup_selfsigned_certs() {
    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"

    log_info "Generating self-signed certificate for *.${DOMAIN}..."

    # Generate CA key + cert (so we can add it to trust store if desired)
    openssl genrsa -out "${cert_dir}/ca.key" 2048 2>/dev/null
    openssl req -new -x509 -days 3650 -key "${cert_dir}/ca.key" \
        -out "${cert_dir}/ca.pem" \
        -subj "/CN=AniLaunchpad Local CA" \
        -addext "basicConstraints=critical,CA:TRUE" \
        -addext "keyUsage=critical,keyCertSign,cRLSign" 2>/dev/null

    # Generate server key
    openssl genrsa -out "${cert_dir}/privkey.pem" 2048 2>/dev/null

    # Generate CSR with SAN (Subject Alternative Names)
    local internal_ip
    internal_ip=$(detect_internal_ip 2>/dev/null || echo "")
    local san="DNS:*.${DOMAIN},DNS:${DOMAIN},DNS:${LAUNCHPAD_DOMAIN},DNS:${GITEA_DOMAIN},DNS:localhost,IP:127.0.0.1"
    [[ -n "$internal_ip" ]] && san="${san},IP:${internal_ip}"
    openssl req -new -key "${cert_dir}/privkey.pem" \
        -out "${cert_dir}/server.csr" \
        -subj "/CN=*.${DOMAIN}" 2>/dev/null

    # Sign with our CA, including SAN extension
    openssl x509 -req -days 3650 \
        -in "${cert_dir}/server.csr" \
        -CA "${cert_dir}/ca.pem" -CAkey "${cert_dir}/ca.key" -CAcreateserial \
        -out "${cert_dir}/fullchain.pem" \
        -extfile <(printf "subjectAltName=%s" "$san") 2>/dev/null

    # Copy as wildcard certs (separate files — symlinks break inside Docker)
    rm -f "${cert_dir}/wildcard-fullchain.pem" "${cert_dir}/wildcard-privkey.pem"
    cp "${cert_dir}/fullchain.pem" "${cert_dir}/wildcard-fullchain.pem"
    cp "${cert_dir}/privkey.pem" "${cert_dir}/wildcard-privkey.pem"

    # Cleanup temp files
    rm -f "${cert_dir}/server.csr" "${cert_dir}/ca.srl"

    log_ok "Self-signed certificate generated"
}

setup_custom_certs() {
    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"

    # Copy main cert (for launchpad + gitea domains)
    if [[ -n "${SSL_CERT_PATH:-}" && -f "$SSL_CERT_PATH" ]]; then
        cp "$SSL_CERT_PATH" "${cert_dir}/fullchain.pem"
        cp "$SSL_KEY_PATH" "${cert_dir}/privkey.pem"
    fi

    # Copy wildcard cert (for *.domain)
    if [[ -n "${SSL_WILDCARD_CERT_PATH:-}" && -f "$SSL_WILDCARD_CERT_PATH" ]]; then
        cp "$SSL_WILDCARD_CERT_PATH" "${cert_dir}/wildcard-fullchain.pem"
        cp "$SSL_WILDCARD_KEY_PATH" "${cert_dir}/wildcard-privkey.pem"
    else
        # Use same cert for wildcard if not provided separately
        [[ -f "${cert_dir}/fullchain.pem" ]] && cp "${cert_dir}/fullchain.pem" "${cert_dir}/wildcard-fullchain.pem"
        [[ -f "${cert_dir}/privkey.pem" ]] && cp "${cert_dir}/privkey.pem" "${cert_dir}/wildcard-privkey.pem"
    fi
}
