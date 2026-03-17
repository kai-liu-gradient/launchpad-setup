#!/bin/bash
# Environment detection and pre-flight checks

check_environment() {
    local errors=0

    # OS info — Linux only
    if [[ "$PLATFORM" == "Linux" ]]; then
        log_ok "$MSG_DETECT_OS: Linux $(uname -r)"
    else
        log_error "$MSG_DETECT_OS: $(uname -s) — not supported (Linux only)"
        ((errors++))
    fi

    # Docker
    if ! command -v docker &>/dev/null; then
        log_error "$MSG_ERR_DOCKER_MISSING"
        ((errors++))
    elif ! docker info &>/dev/null; then
        log_error "$MSG_DETECT_DOCKER: not running"
        ((errors++))
    else
        log_ok "$MSG_DETECT_DOCKER: $(docker --version | awk '{print $3}' | tr -d ',')"
    fi

    # Docker Compose v2
    if ! docker compose version &>/dev/null; then
        log_error "$MSG_DETECT_DOCKER_COMPOSE: not found"
        ((errors++))
    else
        log_ok "$MSG_DETECT_DOCKER_COMPOSE: $(docker compose version --short)"
    fi

    # jq (required by register.sh)
    if ! command -v jq &>/dev/null; then
        log_error "jq is not installed (required for cluster registration)"
        ((errors++))
    else
        log_ok "jq: $(jq --version)"
    fi

    # envsubst (required for template rendering)
    if ! command -v envsubst &>/dev/null; then
        log_error "envsubst is not installed (install gettext-base)"
        ((errors++))
    else
        log_ok "envsubst: available"
    fi

    # helm (required for builtin k8s mode — ingress-nginx, kyverno)
    if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
        if ! command -v helm &>/dev/null; then
            log_error "helm is not installed (required for builtin k8s mode)"
            ((errors++))
        else
            log_ok "helm: $(helm version --short 2>/dev/null)"
        fi
    fi

    if [[ "$errors" -gt 0 ]]; then
        log_error "$MSG_DETECT_FAIL"
        exit 1
    fi
}

# Detect internal IP (for k3s tls-san and DNS reminder)
detect_internal_ip() {
    local ip=""

    for iface in eth0 ens3 ens5; do
        ip=$(ip addr show "$iface" 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | head -1)
        [[ -n "$ip" ]] && echo "$ip" && return 0
    done
    ip=$(ip -4 addr show | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | grep -v '^127\.' | head -1)
    [[ -n "$ip" ]] && echo "$ip" && return 0
    ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    [[ -n "$ip" && "$ip" != "127.0.0.1" ]] && echo "$ip" && return 0

    return 1
}
