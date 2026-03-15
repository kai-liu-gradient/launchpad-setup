#!/bin/bash
# Internationalization — language selection and loading

select_language() {
    local selected=0 count=2
    local options=("English" "中文")

    printf "  ${CYAN}◆${NC} Language / 语言选择\n"

    # Hide cursor
    printf "\033[?25l"

    _draw_lang() {
        local i
        for i in "${!options[@]}"; do
            if [[ $i -eq $selected ]]; then
                printf "    ${CYAN}❯ %s${NC}\033[K\n" "${options[$i]}"
            else
                printf "      ${DIM}%s${NC}\033[K\n" "${options[$i]}"
            fi
        done
    }

    _draw_lang

    while true; do
        IFS= read -rsn1 key
        if [[ "$key" == $'\x1b' ]]; then
            read -rsn2 key
            case "$key" in
                '[A') ((selected > 0)) && ((selected--)) ;;
                '[B') ((selected < count - 1)) && ((selected++)) ;;
            esac
        elif [[ "$key" == "" ]]; then
            break
        elif [[ "$key" == "1" ]]; then
            selected=0; break
        elif [[ "$key" == "2" ]]; then
            selected=1; break
        fi
        printf "\033[%dA" "$count"
        _draw_lang
    done

    # Collapse
    printf "\033[%dA" "$((count + 1))"
    printf "\r\033[J  ${GREEN}◇${NC} Language ${DIM}·${NC} %s\n" "${options[$selected]}"
    printf "\033[?25h"

    case "$selected" in
        1) LANG_CHOICE="zh" ;;
        *) LANG_CHOICE="en" ;;
    esac
    source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE}.sh"
}
