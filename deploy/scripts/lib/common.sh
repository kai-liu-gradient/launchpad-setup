#!/bin/bash
# Common utility functions for AniLaunchpad setup
# Prompt style inspired by @clack/prompts — all display goes to /dev/tty
# so $() captures only the return value.

# Platform detection
PLATFORM="$(uname -s)"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'
DIM='\033[2m'; NC='\033[0m'

# Ensure cursor is always visible (cleanup on exit/interrupt)
trap 'printf "\033[?25h" >/dev/tty 2>/dev/null' EXIT INT TERM

# ─── Logging ───────────────────────────────────────────────────────────

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_ok()    { echo -e "${GREEN}  ✓${NC} $1"; }
log_step()  { echo -e "\n${BOLD}${CYAN}[$1]${NC} $2"; }

# ─── Interactive prompts ───────────────────────────────────────────────

# ask_default PROMPT DEFAULT — text input with optional default
ask_default() {
    local prompt="$1" default="$2" value
    if [[ -n "$default" ]]; then
        printf "  ${CYAN}◆${NC} %s ${DIM}(%s)${NC}: " "$prompt" "$default" >/dev/tty
    else
        printf "  ${CYAN}◆${NC} %s: " "$prompt" >/dev/tty
    fi
    read -r value </dev/tty
    value="${value:-$default}"
    # Collapse to answered state
    printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} %s\n" "$prompt" "$value" >/dev/tty
    echo "$value"
}

# ask_password PROMPT — masked input, shows dots when answered
ask_password() {
    local prompt="$1" value
    printf "  ${CYAN}◆${NC} %s: " "$prompt" >/dev/tty
    read -rs value </dev/tty
    printf "\n" >/dev/tty
    if [[ -n "$value" ]]; then
        printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} ••••••\n" "$prompt" >/dev/tty
    else
        printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}· (empty)${NC}\n" "$prompt" >/dev/tty
    fi
    echo "$value"
}

# ask_choice PROMPT DEFAULT_INDEX OPTIONS... — arrow-key single select
#   Returns 1-based index. Supports arrow keys and number shortcuts.
ask_choice() {
    local prompt="$1" default="$2"
    shift 2
    local options=("$@")
    local selected=$((default - 1))
    local count=${#options[@]}

    printf "  ${CYAN}◆${NC} %s\n" "$prompt" >/dev/tty

    # Hide cursor
    printf "\033[?25l" >/dev/tty

    # Draw function
    _ask_choice_draw() {
        local i
        for i in "${!options[@]}"; do
            if [[ $i -eq $selected ]]; then
                printf "    ${CYAN}❯ %s${NC}\033[K\n" "${options[$i]}" >/dev/tty
            else
                printf "      ${DIM}%s${NC}\033[K\n" "${options[$i]}" >/dev/tty
            fi
        done
    }

    _ask_choice_draw

    while true; do
        IFS= read -rsn1 key </dev/tty
        if [[ "$key" == $'\x1b' ]]; then
            read -rsn2 key </dev/tty
            case "$key" in
                '[A') ((selected > 0)) && ((selected--)) ;;
                '[B') ((selected < count - 1)) && ((selected++)) ;;
            esac
        elif [[ "$key" == "" ]]; then
            break  # Enter
        elif [[ "$key" =~ ^[1-9]$ ]]; then
            local num=$((key - 1))
            if ((num >= 0 && num < count)); then
                selected=$num
                break
            fi
        fi
        # Redraw
        printf "\033[%dA" "$count" >/dev/tty
        _ask_choice_draw
    done

    # Collapse: erase prompt + options, show single answered line
    printf "\033[%dA" "$((count + 1))" >/dev/tty
    printf "\r\033[J  ${GREEN}◇${NC} %s ${DIM}·${NC} %s\n" "$prompt" "${options[$selected]}" >/dev/tty

    # Show cursor
    printf "\033[?25h" >/dev/tty

    echo $((selected + 1))
}

# ask_multichoice PROMPT — comma-separated indices, returns string
ask_multichoice() {
    local prompt="$1" choices
    printf "  ${CYAN}◆${NC} %s: " "$prompt" >/dev/tty
    read -r choices </dev/tty
    printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} %s\n" "$prompt" "${choices:-(skipped)}" >/dev/tty
    echo "$choices"
}

# ask_confirm PROMPT DEFAULT(Y/N) — returns 0=yes, 1=no
ask_confirm() {
    local prompt="$1" default="${2:-Y}" answer hint
    if [[ "$default" =~ ^[Yy] ]]; then
        hint="Y/n"
    else
        hint="y/N"
    fi
    printf "  ${CYAN}◆${NC} %s ${DIM}(%s)${NC}: " "$prompt" "$hint" >/dev/tty
    read -r answer </dev/tty
    answer="${answer:-$default}"
    if [[ "$answer" =~ ^[Yy] ]]; then
        printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} ${GREEN}Yes${NC}\n" "$prompt" >/dev/tty
        return 0
    else
        printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} ${YELLOW}No${NC}\n" "$prompt" >/dev/tty
        return 1
    fi
}

# ask PROMPT — simple text input, no default
ask() {
    local prompt="$1" value
    printf "  ${CYAN}◆${NC} %s: " "$prompt" >/dev/tty
    read -r value </dev/tty
    printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} %s\n" "$prompt" "$value" >/dev/tty
    echo "$value"
}

# ─── Validation ────────────────────────────────────────────────────────

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

# ─── Utilities ─────────────────────────────────────────────────────────

# Cross-platform sed -i (GNU vs BSD)
sed_i() {
    if [[ "$PLATFORM" == "Darwin" ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
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
