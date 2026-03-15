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
    # Check if k3s is already running (works on both Linux and macOS)
    if command -v k3s &>/dev/null; then
        if k3s kubectl get nodes &>/dev/null 2>&1; then
            log_ok "k3s already installed and running"
            # Use existing kubeconfig — k3s default or user-specified
            if [[ -f /etc/rancher/k3s/k3s.yaml ]]; then
                export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
            fi
            return 0
        fi
    fi

    # k3s not running — attempt installation
    if [[ "$PLATFORM" == "Darwin" ]]; then
        # macOS: try k3d (k3s-in-Docker) if available
        if command -v k3d &>/dev/null; then
            log_info "Creating k3s cluster via k3d..."
            k3d cluster create launchpad \
                --port "80:80@loadbalancer" \
                --port "443:443@loadbalancer" \
                --k3s-arg "--disable=traefik@server:0" \
                --wait || {
                log_error "k3d cluster creation failed"
                return 1
            }
            log_ok "k3d cluster 'launchpad' created"
            return 0
        fi

        # No k3s, no k3d — guide user
        log_error "k3s is not running and cannot be auto-installed on macOS."
        echo ""
        echo "  Install via one of these methods, then re-run setup:"
        echo ""
        echo "    # Option 1: k3d (k3s in Docker, recommended)"
        echo "    brew install k3d"
        echo "    k3d cluster create launchpad --port '80:80@loadbalancer' --port '443:443@loadbalancer'"
        echo ""
        echo "    # Option 2: Lima"
        echo "    brew install lima"
        echo "    limactl start --name=k3s template://k3s"
        echo ""
        echo "    # Option 3: Switch to external K8s mode (--reconfigure)"
        echo ""
        exit 1
    fi

    # Linux: standard k3s installation
    log_info "Installing k3s..."

    local internal_ip
    internal_ip=$(detect_internal_ip) || internal_ip="127.0.0.1"

    curl -sfL https://get.k3s.io | sh -s - \
        --disable traefik \
        --tls-san "$internal_ip" \
        --write-kubeconfig-mode 644

    # Wait for k3s to be ready
    log_info "Waiting for k3s to be ready..."
    local timeout=60 elapsed=0
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    while [[ $elapsed -lt $timeout ]]; do
        if k3s kubectl get nodes &>/dev/null; then
            log_ok "k3s is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
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

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local register_script="/tmp/launchpad-register.sh"

    # Try downloading from public URL first, fallback to Docker network
    if curl -fsSL "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh" \
        -o "$register_script" 2>/dev/null; then
        log_ok "Downloaded register.sh from public URL"
    else
        log_warn "Public URL not accessible, using Docker network..."
        docker compose -f "$compose_file" exec -T api \
            curl -fsSL "http://localhost:6802/admin/adminapi/k3s/register.sh" \
            > "$register_script"
    fi
    chmod +x "$register_script"

    # register.sh runs on the host. Try public HTTPS URL first; if DNS not yet configured,
    # fall back to localhost via Nginx (which is listening on 443 on the host).
    local internal_ip
    internal_ip=$(detect_internal_ip 2>/dev/null || echo "127.0.0.1")
    local reg_url="https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register"

    # Test if the public URL is reachable; if not, use IP with Host header via --resolve
    if ! curl -sf --max-time 5 "https://${LAUNCHPAD_DOMAIN}" -o /dev/null 2>/dev/null; then
        log_warn "Public URL not reachable (DNS may not be configured). Using IP with SNI."
        reg_url="https://${internal_ip}/admin/adminapi/k3s/register"
        export CURL_EXTRA_ARGS="--resolve ${LAUNCHPAD_DOMAIN}:443:${internal_ip} -k"
    fi

    if [[ "$K8S_MODE" == "builtin" ]]; then
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        CLUSTER_NAME="local-k3s" \
            "$register_script" || log_warn "Cluster registration returned non-zero (may need manual completion)"
    else
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        KUBECONFIG="$K8S_KUBECONFIG_PATH" \
        CLUSTER_NAME="${K8S_CLUSTER_NAME:-external-k8s}" \
            "$register_script" --skip-k3s-check || log_warn "Cluster registration returned non-zero"
    fi

    rm -f "$register_script"
}
