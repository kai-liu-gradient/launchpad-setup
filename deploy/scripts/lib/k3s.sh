#!/bin/bash
# Kubernetes (k3s) setup and cluster registration

setup_kubernetes() {
    if [[ "$K8S_MODE" == "builtin" ]]; then
        install_k3s
    else
        validate_external_k8s
    fi
}

install_k3s() {
    # Always export KUBECONFIG for k3s — needed by helm and kubectl in later phases
    export KUBECONFIG="${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}"

    # If kubectl works, k3s (or any k8s) is already running — done
    if kubectl get nodes &>/dev/null; then
        log_done "k3s already running"
        return 0
    fi

    log_info "Installing k3s..."

    local internal_ip
    internal_ip=$(detect_internal_ip) || internal_ip="127.0.0.1"

    if [[ "$VERBOSE" == "1" ]]; then
        curl -sfL https://get.k3s.io | sh -s - \
            --disable traefik \
            --tls-san "$internal_ip" \
            --write-kubeconfig-mode 644
    else
        # Run k3s installer in background, track progress via log line count
        local _k3s_log
        _k3s_log=$(mktemp)
        local _k3s_total=20  # ~20 lines expected from k3s installer
        # Only init standalone progress bar when NOT in styled deploy mode
        if [[ -z "${_DS_K3S_STEP:-}" ]]; then
            PROGRESS_TOTAL=$_k3s_total
            PROGRESS_CURRENT=0
            _draw_progress "Installing k3s..."
        fi
        curl -sfL https://get.k3s.io | sh -s - \
            --disable traefik \
            --tls-san "$internal_ip" \
            --write-kubeconfig-mode 644 >"$_k3s_log" 2>&1 &
        local _k3s_pid=$!
        while kill -0 "$_k3s_pid" 2>/dev/null; do
            local _lines
            _lines=$(wc -l < "$_k3s_log" 2>/dev/null || echo 0)
            _lines=$((_lines + 0))  # ensure numeric
            [[ $_lines -gt $_k3s_total ]] && _k3s_total=$_lines
            if [[ -n "${_DS_K3S_STEP:-}" ]] && declare -f _update_step &>/dev/null; then
                # Style C: update the unified deploy progress bar (install phase = 0-70%)
                local _pct=$(( _lines * 70 / _k3s_total ))
                [[ $_pct -gt 70 ]] && _pct=70
                _update_step "$_DS_K3S_STEP" "active" "$_pct"
                _redraw
            else
                PROGRESS_TOTAL=$_k3s_total
                PROGRESS_CURRENT=$_lines
                _draw_progress "Installing k3s..."
            fi
            sleep 1
        done
        wait "$_k3s_pid" || { rm -f "$_k3s_log"; return 1; }
        if [[ -z "${_DS_K3S_STEP:-}" ]]; then
            progress_done
        fi
        rm -f "$_k3s_log"
    fi

    log_info "Waiting for k3s to be ready..."
    local timeout=60 elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if kubectl get nodes &>/dev/null; then
            log_ok "k3s is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        if [[ -n "${_DS_K3S_STEP:-}" ]] && declare -f _update_step &>/dev/null; then
            # Style C: waiting phase = 70-95%
            local _pct=$(( 70 + elapsed * 25 / timeout ))
            [[ $_pct -gt 95 ]] && _pct=95
            _update_step "$_DS_K3S_STEP" "active" "$_pct"
            _redraw
        fi
    done
    log_error "k3s failed to start within ${timeout}s"
    return 1
}

validate_external_k8s() {
    log_info "Validating external K8s cluster..."

    local kubeconfig="${K8S_KUBECONFIG_PATH}"
    if [[ ! -f "$kubeconfig" ]]; then
        log_error "$MSG_ERR_FILE_NOT_FOUND: $kubeconfig"
        exit 1
    fi

    local kubectl_args="--kubeconfig=$kubeconfig"
    [[ -n "${K8S_CONTEXT:-}" ]] && kubectl_args="$kubectl_args --context=$K8S_CONTEXT"

    if kubectl $kubectl_args cluster-info --request-timeout=10s &>/dev/null; then
        local nodes
        nodes=$(kubectl $kubectl_args get nodes --no-headers 2>/dev/null | wc -l)
        local version
        version=$(kubectl $kubectl_args version -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | head -1 | cut -d'"' -f4)
        log_ok "$MSG_K8S_VERIFIED: $nodes nodes, $version"
    else
        log_error "Cannot connect to K8s cluster with provided kubeconfig"
        exit 1
    fi
}

register_cluster() {
    log_info "$MSG_DEPLOY_REGISTER"
    CLUSTER_REGISTERED=false

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local register_script="/tmp/launchpad-register.sh"

    # Try downloading from public URL first, fallback to Docker network
    if curl -fsSL -k "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh" \
        -o "$register_script" 2>/dev/null; then
        log_ok "Downloaded register.sh from public URL"
    else
        log_warn "Public URL not accessible, using Docker network..."
        docker compose -f "$compose_file" exec -T api \
            curl -fsSL "http://localhost:6802/admin/adminapi/k3s/register.sh" \
            > "$register_script" 2>/dev/null
    fi

    # Validate downloaded script
    if [[ ! -s "$register_script" ]] || ! head -1 "$register_script" | grep -q '#!/bin/bash'; then
        log_warn "Failed to download valid register.sh"
        rm -f "$register_script"
        return 1
    fi

    chmod +x "$register_script"

    local internal_ip
    internal_ip=$(detect_internal_ip 2>/dev/null || echo "127.0.0.1")
    local reg_url="https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register"

    if ! curl -sf -k --max-time 5 "https://${LAUNCHPAD_DOMAIN}" -o /dev/null 2>/dev/null; then
        log_warn "Public URL not reachable (DNS may not be configured). Using IP with SNI."
        reg_url="https://${internal_ip}/admin/adminapi/k3s/register"
    fi

    # For self-signed certs, inject -k into all curl commands in register.sh
    if [[ "${SSL_MODE}" == "selfsigned" ]]; then
        sed_i 's|curl -s |curl -sk |g' "$register_script"
    fi

    local reg_ok=false
    # Always pass --skip-k3s-check: we already validated k8s connectivity in setup_kubernetes()
    if [[ "$K8S_MODE" == "builtin" ]]; then
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        CLUSTER_NAME="local-k3s" \
            "$register_script" --skip-k3s-check 2>&1 | verbose_filter && reg_ok=true
    else
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        KUBECONFIG="$K8S_KUBECONFIG_PATH" \
        CLUSTER_NAME="${K8S_CLUSTER_NAME:-external-k8s}" \
            "$register_script" --skip-k3s-check 2>&1 | verbose_filter && reg_ok=true
    fi

    rm -f "$register_script"

    if [[ "$reg_ok" == "true" ]]; then
        CLUSTER_REGISTERED=true
        # For self-signed certs, patch the installed heartbeat script
        if [[ "${SSL_MODE}" == "selfsigned" ]] && [[ -f /usr/local/bin/k3s-heartbeat.sh ]]; then
            sed_i 's|curl -s |curl -sk |g' /usr/local/bin/k3s-heartbeat.sh
        fi
        # Ensure cron environment has PATH and KUBECONFIG for kubectl/jq
        if [[ -f /etc/default/k3s-heartbeat ]]; then
            grep -q '^export PATH=' /etc/default/k3s-heartbeat || \
                echo "export PATH=/usr/local/bin:/usr/bin:/bin" >> /etc/default/k3s-heartbeat
            grep -q '^export KUBECONFIG=' /etc/default/k3s-heartbeat || \
                echo "export KUBECONFIG=${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}" >> /etc/default/k3s-heartbeat
        fi
        log_done "Cluster registered"
    else
        log_warn "Cluster registration failed — you can retry with: ./deploy/setup.sh --resume"
        return 1
    fi
}
