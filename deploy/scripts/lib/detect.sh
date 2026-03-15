#!/bin/bash
# Environment detection and pre-flight checks

check_environment() {
    log_step "0/6" "$MSG_DETECT_CHECKING"

    local errors=0

    # OS check — support Linux and macOS
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

    # jq (required by register.sh)
    if ! command -v jq &>/dev/null; then
        log_error "jq is not installed (required for cluster registration)"
        ((errors++))
    else
        log_ok "jq: $(jq --version)"
    fi

    # Docker Compose v2
    if ! docker compose version &>/dev/null; then
        log_error "$MSG_DETECT_DOCKER_COMPOSE: not found"
        ((errors++))
    else
        log_ok "$MSG_DETECT_DOCKER_COMPOSE: $(docker compose version --short)"
    fi

    # Disk space (min 10GB free) — platform-specific
    local free_gb
    if [[ "$PLATFORM" == "Darwin" ]]; then
        free_gb=$(df -g / | awk 'NR==2 {print $4}')
    else
        free_gb=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    fi
    if [[ "$free_gb" -lt 10 ]]; then
        log_error "$MSG_DETECT_DISK: ${free_gb}GB free (minimum 10GB)"
        ((errors++))
    else
        log_ok "$MSG_DETECT_DISK: ${free_gb}GB free"
    fi

    # Memory (min 2GB) — platform-specific
    local mem_mb
    if [[ "$PLATFORM" == "Darwin" ]]; then
        mem_mb=$(( $(sysctl -n hw.memsize) / 1024 / 1024 ))
    else
        mem_mb=$(free -m | awk '/^Mem:/ {print $7}')
    fi
    if [[ "$mem_mb" -lt 2048 ]]; then
        log_warn "$MSG_DETECT_MEMORY: ${mem_mb}MB available (recommended 4GB+)"
    else
        log_ok "$MSG_DETECT_MEMORY: ${mem_mb}MB available"
    fi

    # Required ports — platform-specific
    local ports=(80 443 5432 6379 6555 6801 6802 6804 3000 2222)
    local port_errors=0
    for port in "${ports[@]}"; do
        local in_use=false
        if [[ "$PLATFORM" == "Darwin" ]]; then
            lsof -iTCP:"$port" -sTCP:LISTEN -P -n &>/dev/null && in_use=true
        else
            ss -tlnp 2>/dev/null | grep -q ":${port} " && in_use=true
        fi
        if $in_use; then
            log_warn "$MSG_ERR_PORT_IN_USE: $port"
            ((port_errors++))
        fi
    done
    if [[ "$port_errors" -eq 0 ]]; then
        log_ok "$MSG_DETECT_PORTS: all available"
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
        # macOS: use ipconfig on active interface
        ip=$(ipconfig getifaddr en0 2>/dev/null)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        ip=$(ipconfig getifaddr en1 2>/dev/null)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        # Fallback
        ip=$(ifconfig | grep "inet " | grep -v '127.0.0.1' | head -1 | awk '{print $2}')
        [[ -n "$ip" ]] && echo "$ip" && return 0
    else
        # Linux: try common interfaces
        for iface in eth0 ens3 ens5; do
            ip=$(ip addr show "$iface" 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | head -1)
            [[ -n "$ip" ]] && echo "$ip" && return 0
        done
        # Fallback: first non-localhost
        ip=$(ip -4 addr show | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | grep -v '^127\.' | head -1)
        [[ -n "$ip" ]] && echo "$ip" && return 0
        ip=$(hostname -I 2>/dev/null | awk '{print $1}')
        [[ -n "$ip" && "$ip" != "127.0.0.1" ]] && echo "$ip" && return 0
    fi
    return 1
}
