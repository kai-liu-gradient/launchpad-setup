#!/bin/bash
# Kubernetes component setup (ingress-nginx, CoreDNS, Kyverno)
# Only runs in K8S_MODE=builtin

install_ingress_nginx() {
    if helm status ingress-nginx -n ingress-nginx &>/dev/null; then
        log_ok "ingress-nginx already installed"
        return 0
    fi

    log_info "Installing ingress-nginx via Helm..."
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
    helm repo update

    helm install ingress-nginx ingress-nginx/ingress-nginx \
        -n ingress-nginx --create-namespace \
        -f "${DEPLOY_DIR}/templates/values-builtin.yml"

    log_info "Waiting for ingress-nginx controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=120s

    log_ok "ingress-nginx installed"
}

install_kyverno() {
    if helm status kyverno -n kyverno &>/dev/null; then
        log_ok "Kyverno already installed"
        return 0
    fi

    log_info "Installing Kyverno via Helm..."
    helm repo add kyverno https://kyverno.github.io/kyverno/
    helm install kyverno kyverno/kyverno \
        -n kyverno --create-namespace

    log_info "Waiting for Kyverno admission controller to be ready..."
    kubectl wait --namespace kyverno \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=admission-controller \
        --timeout=120s

    log_ok "Kyverno installed"
}

setup_cert_distribution() {
    local ca_cert="${DEPLOY_DIR}/generated/nginx/certs/ca.pem"
    if [[ ! -f "$ca_cert" ]]; then
        log_warn "CA certificate not found at $ca_cert — skipping cert distribution"
        return 0
    fi

    log_info "Distributing CA certificate via Kyverno..."

    # Create source ConfigMap in kube-system
    kubectl create configmap launchpad-ca-cert \
        --from-file=ca.pem="$ca_cert" \
        -n kube-system --dry-run=client -o yaml | kubectl apply -f -

    # Apply Kyverno policies
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-sync-ca.yaml"
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-inject-ca.yaml"

    log_ok "CA certificate distribution configured"
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
    export INGRESS_NGINX_CLUSTER_IP="$cluster_ip"
    export DOMAIN_ESCAPED="$domain_escaped"

    envsubst '${HOST_IP} ${INGRESS_NGINX_CLUSTER_IP} ${DOMAIN} ${DOMAIN_ESCAPED} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN}' \
        < "${DEPLOY_DIR}/templates/coredns-custom.yaml.template" \
        > /tmp/coredns-custom.yaml

    kubectl apply -f /tmp/coredns-custom.yaml

    # Patch default CoreDNS ConfigMap: disable loop plugin, use public DNS forwarders
    # Idempotent sed: only comments out uncommented 'loop' lines
    kubectl get cm coredns -n kube-system -o yaml \
        | sed '/^[^#]*loop$/s/loop/# loop/' \
        | sed 's|forward \. /etc/resolv\.conf|forward . 223.5.5.5 8.8.8.8|' \
        | kubectl apply -f -

    # k3s natively supports coredns-custom ConfigMap convention (auto-imports)
    kubectl rollout restart deploy/coredns -n kube-system
    rm -f /tmp/coredns-custom.yaml

    log_ok "CoreDNS configured"
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

    log_ok "k8s components ready"
}
