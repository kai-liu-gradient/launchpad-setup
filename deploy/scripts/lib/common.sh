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
trap 'if [ -t 1 ] || [ -t 2 ]; then printf "\033[?25h" >/dev/tty 2>/dev/null; fi; true' EXIT INT TERM

# ─── Logging ───────────────────────────────────────────────────────────
# VERBOSE=1 shows all output; VERBOSE=0 (default) shows only steps/warnings/errors
VERBOSE="${VERBOSE:-0}"

log_info()  { [[ "$VERBOSE" == "1" ]] && echo -e "${GREEN}[INFO]${NC} $1" || true; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_ok()    { [[ "$VERBOSE" == "1" ]] && echo -e "${GREEN}  ✓${NC} $1" || true; }
log_step()  { echo -e "\n${BOLD}${CYAN}[$1]${NC} $2"; }
log_done()  { echo -e "${GREEN}  ✓${NC} $1"; }
log_debug() { [[ "$VERBOSE" == "1" ]] && echo -e "${DIM}[DEBUG]${NC} $1" || true; }

# Filter command output: show in verbose mode, suppress otherwise
verbose_filter() {
    if [[ "$VERBOSE" == "1" ]]; then
        cat
    else
        cat > /dev/null
    fi
}

# ─── Progress bar ──────────────────────────────────────────────────────
# Usage: progress_start TOTAL_STEPS
#        progress_update "message"
#        progress_done
PROGRESS_CURRENT=0
PROGRESS_TOTAL=1

progress_start() {
    PROGRESS_TOTAL="$1"
    PROGRESS_CURRENT=0
    _draw_progress ""
}

progress_update() {
    PROGRESS_CURRENT=$((PROGRESS_CURRENT + 1))
    _draw_progress "$1"
}

progress_done() {
    PROGRESS_CURRENT=$PROGRESS_TOTAL
    local pct=100
    local bar_width=30
    local filled=$bar_width
    printf "\r\033[K  ${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC} ${BOLD}100%%${NC}  ${GREEN}✓ Done${NC}\n"
}

_draw_progress() {
    local msg="$1"
    local pct=$(( PROGRESS_CURRENT * 100 / PROGRESS_TOTAL ))
    local bar_width=30
    local filled=$(( pct * bar_width / 100 ))
    local empty=$(( bar_width - filled ))
    local bar=""
    local i

    for (( i=0; i<filled; i++ )); do bar+="━"; done
    local trail=""
    for (( i=0; i<empty; i++ )); do trail+="━"; done

    printf "\r\033[K  ${GREEN}%s${DIM}%s${NC} ${BOLD}%3d%%${NC}  %s" "$bar" "$trail" "$pct" "$msg"
}

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
                '[A') ((selected > 0)) && ((selected--)) || true ;;
                '[B') ((selected < count - 1)) && ((selected++)) || true ;;
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

# Expand ~ to $HOME in paths (bash doesn't expand ~ inside quotes)
expand_path() {
    local p="$1"
    if [[ "$p" == "~/"* ]]; then
        p="${HOME}/${p#\~/}"
    elif [[ "$p" == "~" ]]; then
        p="$HOME"
    fi
    echo "$p"
}

# ask_filepath PROMPT DEFAULT — like ask_default but expands ~ in the result
ask_filepath() {
    local prompt="$1" default="$2" value
    if [[ -n "$default" ]]; then
        printf "  ${CYAN}◆${NC} %s ${DIM}(%s)${NC}: " "$prompt" "$default" >/dev/tty
    else
        printf "  ${CYAN}◆${NC} %s: " "$prompt" >/dev/tty
    fi
    read -r value </dev/tty
    value="${value:-$default}"
    value="$(expand_path "$value")"
    printf "\033[1A\r\033[K  ${GREEN}◇${NC} %s ${DIM}·${NC} %s\n" "$prompt" "$value" >/dev/tty
    echo "$value"
}

# ─── Utilities ─────────────────────────────────────────────────────────

# sed -i wrapper (GNU/Linux)
sed_i() {
    sed -i "$@"
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

# ─── Style C rendering helpers ───────────────────────────────────────

# Terminal width (cached)
_term_width() {
    tput cols 2>/dev/null || echo 80
}

# Print text centered to terminal width
print_centered() {
    local text="$1"
    # Strip ANSI codes to calculate visible length
    local stripped
    stripped=$(echo -e "$text" | sed 's/\x1b\[[0-9;]*m//g')
    local w
    w=$(_term_width)
    local pad=$(( (w - ${#stripped}) / 2 ))
    [[ $pad -lt 0 ]] && pad=0
    printf "%*s" "$pad" "" >/dev/tty
    printf "%b\n" "$text" >/dev/tty
}

# Print the 4-colored-squares brand header
print_brand_header() {
    local subtitle="${1:-$MSG_BRAND_SUBTITLE}"
    echo "" >/dev/tty
    print_centered "\033[0;31m■\033[0m \033[1;33m■\033[0m \033[0;32m■\033[0m \033[0;34m■\033[0m"
    print_centered "${BOLD}AniLaunchpad${NC}"
    print_centered "${DIM}${subtitle}${NC}"
    echo "" >/dev/tty
}

# Print tab navigation bar. Active tab (1-based index) is green+bold.
print_tab_bar() {
    local active="$1"
    local tabs=("$MSG_TAB_BASIC" "$MSG_TAB_SSL" "$MSG_TAB_DB" "$MSG_TAB_K8S" "$MSG_TAB_ADV" "$MSG_TAB_DEPLOY")
    local line=""
    local i
    for i in "${!tabs[@]}"; do
        local idx=$((i + 1))
        if [[ $idx -eq $active ]]; then
            line+="${GREEN}${BOLD}${tabs[$i]}${NC}  "
        else
            line+="${DIM}${tabs[$i]}${NC}  "
        fi
    done
    printf "  %b\n" "$line" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    echo "" >/dev/tty
}

# Print a left-border accent block.
# Usage: print_accent_block <color> <title> <lines...>
# color: green, blue, yellow, red
print_accent_block() {
    local color_name="$1" title="$2"
    shift 2
    local cc
    case "$color_name" in
        green)  cc="$GREEN" ;;
        blue)   cc="$BLUE" ;;
        yellow) cc="$YELLOW" ;;
        red)    cc="$RED" ;;
        *)      cc="$NC" ;;
    esac
    echo "" >/dev/tty
    printf "  ${cc}┃${NC} ${cc}${BOLD}%s${NC}\n" "$title" >/dev/tty
    local line
    for line in "$@"; do
        printf "  ${cc}┃${NC} %b\n" "$line" >/dev/tty
    done
}

# Print the bottom navigation hotkey bar
# Usage: print_hotkey_bar [deploy]
print_hotkey_bar() {
    local mode="${1:-nav}"
    echo "" >/dev/tty
    printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    if [[ "$mode" == "deploy" ]]; then
        printf "       ${DIM}[ %s ]  [ %s ]  [ %s ]${NC}\n" "$MSG_NAV_ENTER_DEPLOY" "$MSG_NAV_BACK" "$MSG_NAV_QUIT" >/dev/tty
    else
        printf "       ${DIM}[ %s ]  [ %s ]  [ %s ]${NC}\n" "$MSG_NAV_ENTER_NEXT" "$MSG_NAV_BACK" "$MSG_NAV_QUIT" >/dev/tty
    fi
}

# Print an answered field line: ▸ Label    value
print_field() {
    local label="$1" value="$2"
    printf "  ${GREEN}▸${NC} %-16s %b\n" "$label" "${DIM}${value}${NC}" >/dev/tty
}

# Draw deployment progress with checkmarks, spinner, and pending items.
# Usage: _draw_deploy_status current total steps_array_name
# steps_array: ("done:Label" "active:Label" "pending:Label" ...)
_deploy_spinner_chars=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
_deploy_spinner_idx=0

_draw_deploy_status() {
    local current="$1" total="$2"
    local -n steps_ref="$3"
    local pct=$(( current * 100 / total ))
    local bar_width=40
    local filled=$(( pct * bar_width / 100 ))
    local empty=$(( bar_width - filled ))

    # Move cursor to saved position (top of deploy area)
    printf "\033[${_deploy_area_top};1H" >/dev/tty

    # Overall progress bar
    local bar="" trail=""
    local i
    for (( i=0; i<filled; i++ )); do bar+="━"; done
    for (( i=0; i<empty; i++ )); do trail+="━"; done
    printf "  \033[0;32m%s\033[0m\033[2m%s\033[0m  \033[1m%d%%\033[0m\033[K\n" "$bar" "$trail" "$pct" >/dev/tty
    printf "\033[K\n" >/dev/tty

    # Step list with per-item mini progress bars
    # Step format: "status:pct:label"
    # Fixed layout: [icon 2col] [label padded to 22 display cols] [minibar 10ch] [pct 5ch]
    local target_w=22 mini_w=10
    for step in "${steps_ref[@]}"; do
        local status="${step%%:*}"
        local rest="${step#*:}"
        local step_pct="${rest%%:*}"
        local label="${rest#*:}"

        # Calculate display width: CJK chars = 2 cols, ASCII = 1 col
        # ${#label} gives char count (locale-aware), wc -c gives byte count
        # UTF-8 CJK: 3 bytes per char, ASCII: 1 byte per char
        # cjk_count = (bytes - chars) / 2; display_w = chars + cjk_count
        local char_len=${#label}
        local byte_len
        byte_len=$(printf '%s' "$label" | wc -c | tr -d ' ')
        local cjk_count=$(( (byte_len - char_len) / 2 ))
        local display_w=$(( char_len + cjk_count ))

        # Pad with spaces to reach target display width
        local pad_needed=$(( target_w - display_w ))
        [[ $pad_needed -lt 0 ]] && pad_needed=0
        local padding=""
        for (( i=0; i<pad_needed; i++ )); do padding+=" "; done
        local padded="${label}${padding}"

        # Build mini bar
        local mf=$(( step_pct * mini_w / 100 ))
        local me=$(( mini_w - mf ))
        local mbar="" mtrail=""
        for (( i=0; i<mf; i++ )); do mbar+="━"; done
        for (( i=0; i<me; i++ )); do mtrail+="━"; done
        case "$status" in
            done)
                printf "  \033[0;32m✓\033[0m %s \033[0;32m%s\033[0m \033[1m100%%\033[0m\033[K\n" "$padded" "$mbar" >/dev/tty
                ;;
            active)
                local sc="${_deploy_spinner_chars[$_deploy_spinner_idx]}"
                printf "  \033[1;33m%s\033[0m %s \033[0;32m%s\033[0m\033[2m%s\033[0m \033[1m%3d%%\033[0m\033[K\n" "$sc" "$padded" "$mbar" "$mtrail" "$step_pct" >/dev/tty
                _deploy_spinner_idx=$(( (_deploy_spinner_idx + 1) % ${#_deploy_spinner_chars[@]} ))
                ;;
            pending)
                printf "  \033[2m○ %s %s   0%%\033[0m\033[K\n" "$padded" "$mtrail" >/dev/tty
                ;;
        esac
    done
    # Clear any leftover lines
    printf "\033[K" >/dev/tty
}
