#!/bin/bash
set -euo pipefail

SETUP_VERSION="1.0"
DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$DEPLOY_DIR")"

# Load versions
source "${DEPLOY_DIR}/versions.conf"

# Load libraries
source "${DEPLOY_DIR}/scripts/lib/common.sh"
source "${DEPLOY_DIR}/scripts/lib/i18n.sh"

# Parse CLI arguments
ACTION="install"
CONFIG_FILE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --reconfigure) ACTION="reconfigure"; shift ;;
        --status)      ACTION="status";      shift ;;
        --upgrade)     ACTION="upgrade";     shift ;;
        --uninstall)   ACTION="uninstall";   shift ;;
        --resume)           ACTION="resume";           shift ;;
        --import-templates) ACTION="import-templates"; shift ;;
        --uninstall-all)    ACTION="uninstall-all";    shift ;;
        --setup-k3s)        ACTION="setup-k3s";        shift ;;
        --setup-certs)      ACTION="setup-certs";      shift ;;
        --setup-db)         ACTION="setup-db";         shift ;;
        --setup-telegram)   ACTION="setup-telegram";   shift ;;
        --restart)          [[ -z "${2:-}" ]] && { log_error "--restart requires a service name"; exit 1; }
                            ACTION="restart"; RESTART_TARGET="$2"; shift 2 ;;
        --verbose|-v)  VERBOSE=1;            shift ;;
        --help|-h)     ACTION="help";        shift ;;
        --config)      [[ -z "${2:-}" ]] && { log_error "--config requires a file path"; exit 1; }
                       CONFIG_FILE="$2"; shift 2 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# Validate --config is only used with install/reconfigure
if [[ -n "$CONFIG_FILE" && "$ACTION" != "install" && "$ACTION" != "reconfigure" ]]; then
    log_error "--config is only valid with install/reconfigure"
    exit 1
fi

# Load previous config if reconfiguring
if [[ "$ACTION" == "reconfigure" ]] && [[ -f "${DEPLOY_DIR}/generated/.setup.conf" ]]; then
    source "${DEPLOY_DIR}/generated/.setup.conf"
fi

# Helper: load saved config for granular commands
load_saved_config() {
    if [[ ! -d "${DEPLOY_DIR}/generated" ]]; then
        log_error "No deployment found. Run ./setup.sh first."
        exit 1
    fi
    source "${DEPLOY_DIR}/versions.conf"
    source "${DEPLOY_DIR}/generated/.setup.conf"
    source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    load_secrets || { log_error "Cannot load secrets."; exit 1; }
}

case "$ACTION" in
    install|reconfigure)
        # Brand header shown by wizard pages; skip old banner for interactive mode
        if [[ -n "$CONFIG_FILE" ]]; then
            print_banner "AniLaunchpad Setup" "$SETUP_VERSION"
        fi

        # Load remaining modules
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/interact.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/render.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        source "${DEPLOY_DIR}/scripts/lib/k8s-components.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"

        if [[ -n "$CONFIG_FILE" ]]; then
            # Non-interactive: load config from file
            [[ ! -f "$CONFIG_FILE" ]] && { log_error "Config file not found: $CONFIG_FILE"; exit 1; }
            source "$CONFIG_FILE"
            source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
            check_environment
        else
            # Interactive: page-based wizard
            select_language
            check_environment
            run_wizard
        fi

        # Phase 4: Generate (shared by both modes)
        if ! load_secrets; then
            generate_secrets
        fi
        render_templates
        save_config
        save_secrets

        # Phase 5: Deploy (unified progress display)
        deploy_services

        # Done
        show_result
        ;;
    status)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        show_status
        ;;
    upgrade)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        upgrade_services
        ;;
    uninstall)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        uninstall_services
        ;;
    resume)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/interact.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/render.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        source "${DEPLOY_DIR}/scripts/lib/k8s-components.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        deploy_services
        show_result
        ;;
    import-templates)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        # Load Gitea token from .env (not in .secrets)
        if [[ -f "${DEPLOY_DIR}/generated/launchpad/.env" ]]; then
            GITEA_ACCESS_TOKEN=$(grep '^GITEA_ACCESS_TOKEN=' "${DEPLOY_DIR}/generated/launchpad/.env" | cut -d= -f2)
            export GITEA_ACCESS_TOKEN
        fi
        import_templates
        ;;
    uninstall-all)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        # Load lang for i18n messages (may fail if no prior install, use defaults)
        if [[ -f "${DEPLOY_DIR}/generated/.setup.conf" ]]; then
            source "${DEPLOY_DIR}/generated/.setup.conf"
        fi
        source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
        uninstall_all
        ;;
    setup-k3s)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        load_saved_config
        setup_kubernetes
        ;;
    setup-certs)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        load_saved_config
        setup_certificates
        ;;
    setup-db)
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        load_saved_config
        init_database
        ;;
    setup-telegram)
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/interact.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/render.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        TELEGRAM_BOT_USERNAME=$(ask_default "Telegram Bot Username" "${TELEGRAM_BOT_USERNAME:-}")
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
        render_templates
        save_config
        restart_service "api"
        restart_service "gateway"
        ;;
    restart)
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        load_saved_config
        restart_service "$RESTART_TARGET"
        ;;
    help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Install:"
        echo "  (none)              Fresh install (interactive wizard)"
        echo "  --config FILE       Non-interactive install using config file"
        echo "  --reconfigure       Re-enter configuration wizard"
        echo ""
        echo "Manage:"
        echo "  --status            Show service status"
        echo "  --restart <svc>     Restart service (api|ui|router|gateway|nginx|gitea|all)"
        echo "  --upgrade           Update image versions"
        echo ""
        echo "Setup Phases:"
        echo "  --resume            Resume full deploy using existing config"
        echo "  --setup-k3s         Install/reconfigure k3s only"
        echo "  --setup-certs       Regenerate SSL certificates only"
        echo "  --setup-db          Run database initialization only"
        echo "  --setup-telegram    Configure Telegram bot settings"
        echo "  --import-templates  Import templates to API & Gitea"
        echo ""
        echo "Teardown:"
        echo "  --uninstall         Stop services, remove containers and data"
        echo "  --uninstall-all     Complete removal (services + k3s + config)"
        echo ""
        echo "General:"
        echo "  --verbose, -v       Show detailed output"
        echo "  --help              Show this help"
        ;;
esac
