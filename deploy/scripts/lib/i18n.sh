#!/bin/bash
# Internationalization — language selection and loading

select_language() {
    echo "  Language / 语言选择:"
    echo "    1) English"
    echo "    2) 中文"
    printf "  Please select / 请选择 [1]: "
    read -r lang_choice
    lang_choice="${lang_choice:-1}"
    case "$lang_choice" in
        2) LANG_CHOICE="zh" ;;
        *) LANG_CHOICE="en" ;;
    esac
    source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE}.sh"
}
