#!/bin/bash
# Common utility functions for AniLaunchpad setup

# Platform detection
PLATFORM="$(uname -s)"  # Linux or Darwin

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_ok()    { echo -e "${GREEN}  ✓${NC} $1"; }
log_step()  { echo -e "\n${BOLD}${CYAN}[$1]${NC} $2"; }

# ask_default PROMPT DEFAULT_VALUE — prints prompt with default, reads input, returns value
ask_default() {
    local prompt="$1" default="$2" value
    printf "  %s [%s]: " "$prompt" "$default"
    read -r value
    echo "${value:-$default}"
}

# ask_password PROMPT — reads password without echo
ask_password() {
    local prompt="$1" value
    printf "  %s: " "$prompt"
    read -rs value
    echo ""
    echo "$value"
}

# ask_choice PROMPT OPTIONS_ARRAY DEFAULT_INDEX — numbered selection
ask_choice() {
    local prompt="$1" default="$2"
    shift 2
    local options=("$@")
    local i=1
    for opt in "${options[@]}"; do
        echo "    $i) $opt"
        ((i++))
    done
    printf "  %s [%s]: " "$prompt" "$default"
    read -r choice
    echo "${choice:-$default}"
}

# ask_multichoice PROMPT — comma-separated multi-select, returns selected indices
ask_multichoice() {
    local prompt="$1"
    printf "  %s: " "$prompt"
    read -r choices
    echo "$choices"
}

# ask_confirm PROMPT DEFAULT(Y/n) — returns 0 for yes, 1 for no
ask_confirm() {
    local prompt="$1" default="${2:-Y}"
    printf "  %s [%s]: " "$prompt" "$default"
    read -r answer
    answer="${answer:-$default}"
    [[ "$answer" =~ ^[Yy] ]]
}

# Validation
validate_domain() {
    [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]
}

validate_ip() {
    [[ "$1" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]
}

validate_port() {
    [[ "$1" =~ ^[0-9]+$ ]] && [[ "$1" -ge 1 && "$1" -le 65535 ]]
}

validate_file_exists() {
    [[ -f "$1" ]]
}

# Cross-platform sed -i (GNU vs BSD)
sed_i() {
    if [[ "$PLATFORM" == "Darwin" ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# ask PROMPT — simple prompt, returns input (no default)
ask() {
    local prompt="$1" value
    printf "  %s: " "$prompt"
    read -r value
    echo "$value"
}

# Print banner
print_banner() {
    local title="$1" version="$2"
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
    echo -e "${BOLD}  $title v$version${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
    echo ""
}

# Print summary box
print_summary() {
    echo -e "  ┌──────────────────────────────────────┐"
    while IFS='|' read -r label value; do
        printf "  │ %-14s %-22s │\n" "$label" "$value"
    done
    echo -e "  └──────────────────────────────────────┘"
}
