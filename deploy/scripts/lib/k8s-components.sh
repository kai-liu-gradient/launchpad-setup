#!/bin/bash
# Kubernetes component setup (ingress-nginx, CoreDNS, Kyverno)
# Only runs in K8S_MODE=builtin

install_ingress_nginx() {
    # Remove k3s built-in Traefik if present (leftover from prior installs)
    if helm status traefik -n kube-system &>/dev/null; then
        log_info "Removing k3s built-in Traefik..."
        helm uninstall traefik -n kube-system 2>&1 | verbose_filter
        helm uninstall traefik-crd -n kube-system 2>&1 | verbose_filter || true
        # Remove k3s manifests so Traefik doesn't come back on restart
        rm -f /var/lib/rancher/k3s/server/manifests/traefik.yaml
        rm -f /var/lib/rancher/k3s/server/manifests/traefik-config.yaml
    fi

    if helm status ingress-nginx -n ingress-nginx &>/dev/null; then
        log_done "ingress-nginx already installed"
        return 0
    fi

    log_info "Installing ingress-nginx via Helm..."
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>&1 | verbose_filter
    helm install ingress-nginx ingress-nginx/ingress-nginx \
        -n ingress-nginx --create-namespace \
        -f "${DEPLOY_DIR}/templates/values-builtin.yml" 2>&1 | verbose_filter

    # Wait for controller pod to exist
    local attempts=0
    while ! kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller --no-headers 2>/dev/null | grep -q .; do
        attempts=$((attempts + 1))
        if [[ $attempts -ge 30 ]]; then
            log_error "ingress-nginx pod not found after 60s"
            return 1
        fi
        sleep 2
    done

    kubectl wait -n ingress-nginx --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=120s 2>&1 | verbose_filter

    log_done "ingress-nginx is ready"
}

install_kyverno() {
    if helm status kyverno -n kyverno &>/dev/null; then
        log_done "Kyverno already installed"
        return 0
    fi

    log_info "Installing Kyverno via Helm..."
    helm repo add kyverno https://kyverno.github.io/kyverno/ 2>&1 | verbose_filter
    helm install kyverno kyverno/kyverno \
        -n kyverno --create-namespace 2>&1 | verbose_filter

    log_info "Waiting for Kyverno admission controller to be ready..."
    kubectl wait --namespace kyverno \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=admission-controller \
        --timeout=120s 2>&1 | verbose_filter

    log_done "Kyverno installed"
}

setup_cert_distribution() {
    local ca_cert="${DEPLOY_DIR}/generated/nginx/certs/ca.pem"
    if [[ ! -f "$ca_cert" ]]; then
        log_warn "CA certificate not found at $ca_cert — skipping cert distribution"
        return 0
    fi

    log_info "Distributing CA certificate via Kyverno..."

    # Create source ConfigMap with raw CA PEM in kube-system
    kubectl create configmap launchpad-ca-cert \
        --from-file=ca.pem="$ca_cert" \
        -n kube-system --dry-run=client -o yaml | kubectl apply -f - 2>&1 | verbose_filter

    # Pre-merge system CA bundle + custom CA, and include profile.d script
    # so that login shells (su - node) also get NODE_EXTRA_CA_CERTS
    cat /etc/ssl/certs/ca-certificates.crt "$ca_cert" > /tmp/ca-bundle.crt
    kubectl create configmap launchpad-ca-bundle \
        --from-file=ca-certificates.crt=/tmp/ca-bundle.crt \
        --from-literal=launchpad-ca.sh='export NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem' \
        -n kube-system --dry-run=client -o yaml | kubectl apply -f - 2>&1 | verbose_filter
    rm -f /tmp/ca-bundle.crt

    # Apply Kyverno policies
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-sync-ca.yaml" 2>&1 | verbose_filter
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-inject-ca.yaml" 2>&1 | verbose_filter

    log_done "CA certificate distribution configured"
}

configure_coredns() {
    local host_ip
    host_ip=$(detect_internal_ip) || { log_error "Cannot detect internal IP for CoreDNS config"; return 1; }

    local cluster_ip
    cluster_ip=$(kubectl get svc ingress-nginx-controller \
        -n ingress-nginx -o jsonpath='{.spec.clusterIP}')

    if [[ -z "$cluster_ip" ]]; then
        log_error "Cannot get ingress-nginx ClusterIP"
        return 1
    fi

    log_info "Configuring CoreDNS (host=$host_ip, ingress=$cluster_ip)..."

    local domain_escaped="${DOMAIN//./\\.}"

    # Render template — limit envsubst variables to avoid clobbering CoreDNS {{ .Name }} syntax
    export HOST_IP="$host_ip"
    export INGRESS_CLUSTER_IP="$cluster_ip"
    export DOMAIN_ESCAPED="$domain_escaped"
    export DOMAIN="${DOMAIN}"
    export LAUNCHPAD_DOMAIN="${LAUNCHPAD_DOMAIN}"
    export GITEA_DOMAIN="${GITEA_DOMAIN}"

    envsubst '${HOST_IP} ${INGRESS_CLUSTER_IP} ${DOMAIN} ${DOMAIN_ESCAPED} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN}' \
        < "${DEPLOY_DIR}/templates/coredns-custom.yaml.template" \
        > /tmp/coredns-custom.yaml

    kubectl apply -f /tmp/coredns-custom.yaml 2>&1 | verbose_filter

    # Patch default CoreDNS ConfigMap: disable loop plugin, use public DNS forwarders
    # Idempotent sed: only comments out uncommented 'loop' lines
    kubectl get cm coredns -n kube-system -o yaml \
        | sed '/^[^#]*loop$/s/loop/# loop/' \
        | sed 's|forward \. /etc/resolv\.conf|forward . 223.5.5.5 8.8.8.8|' \
        | kubectl apply -f - 2>&1 | verbose_filter

    # k3s natively supports coredns-custom ConfigMap convention (auto-imports)
    kubectl rollout restart deploy/coredns -n kube-system 2>&1 | verbose_filter
    rm -f /tmp/coredns-custom.yaml

    # Configure dnsmasq so Docker containers resolve *.DOMAIN to host IP.
    # Gateway needs to reach nginx:443 (for correct TLS cert on wildcard domains).
    # Router falls back to DEFAULT_BACKEND (host:30080) for pod traffic.
    if command -v dnsmasq &>/dev/null; then
        mkdir -p /etc/dnsmasq.d
        cat > /etc/dnsmasq.d/launchpad.conf << EOF
# Auto-generated by launchpad deploy — do not edit
# Route *.${DOMAIN} to host IP so Docker containers reach nginx:443
address=/${DOMAIN}/${host_ip}
server=223.5.5.5
server=8.8.8.8
EOF
        systemctl restart dnsmasq 2>&1 | verbose_filter || true
        log_ok "dnsmasq configured (*.${DOMAIN} → ${host_ip})"
    fi

    log_done "CoreDNS configured"
}

setup_k8s_components() {
    [[ "$K8S_MODE" != "builtin" ]] && return 0

    log_info "Setting up k8s components..."

    install_ingress_nginx
    configure_coredns

    if [[ "$SSL_MODE" == "selfsigned" ]]; then
        install_kyverno
        setup_cert_distribution
    fi

    log_done "k8s components ready"
}
