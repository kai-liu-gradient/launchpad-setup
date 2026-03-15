#!/bin/bash
# Environment detection and pre-flight checks

check_environment() {
    local errors=0

    # OS info → banner line
    case "$PLATFORM" in
        Linux)  log_ok "$MSG_DETECT_OS: Linux $(uname -r)" ;;
        Darwin) log_ok "$MSG_DETECT_OS: macOS $(sw_vers -productVersion 2>/dev/null || uname -r)" ;;
        *)      log_error "$MSG_DETECT_OS: $(uname -s) — not supported"; ((errors++)) ;;
    esac

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

    if [[ "$errors" -gt 0 ]]; then
        log_error "$MSG_DETECT_FAIL"
        exit 1
    fi
}

# Detect internal IP (for k3s tls-san and DNS reminder) — platform-specific
detect_internal_ip() {
    local ip=""

    if [[ "$PLATFORM" == "Darwin" ]]; then
        ip=$(ipconfig getifaddr en0 2>/dev/null)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        ip=$(ipconfig getifaddr en1 2>/dev/null)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        ip=$(ifconfig | grep "inet " | grep -v '127.0.0.1' | head -1 | awk '{print $2}')
        [[ -n "$ip" ]] && echo "$ip" && return 0
    else
        for iface in eth0 ens3 ens5; do
            ip=$(ip addr show "$iface" 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | head -1)
            [[ -n "$ip" ]] && echo "$ip" && return 0
        done
        ip=$(ip -4 addr show | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | grep -v '^127\.' | head -1)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        ip=$(hostname -I 2>/dev/null | awk '{print $1}')
        [[ -n "$ip" && "$ip" != "127.0.0.1" ]] && echo "$ip" && return 0
    fi
    return 1
}
