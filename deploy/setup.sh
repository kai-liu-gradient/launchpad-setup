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
        --resume)      ACTION="resume";      shift ;;
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

case "$ACTION" in
    install|reconfigure)
        # Banner
        print_banner "AniLaunchpad Setup" "$SETUP_VERSION"

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
            # Interactive: original flow
            select_language
            check_environment
            collect_basic_config
            collect_advanced_config
            show_summary
            if ! ask_confirm "$MSG_DEPLOY_CONFIRM" "Y"; then
                log_warn "Deployment cancelled."
                exit 0
            fi
        fi

        # Phase 4: Generate (shared by both modes)
        if ! load_secrets; then
            generate_secrets
        fi
        render_templates
        save_config
        save_secrets

        # Phase 5: Deploy
        setup_kubernetes
        setup_certificates
        setup_k8s_components
        deploy_services
        register_cluster || log_warn "Cluster registration failed — you can retry later with: ./deploy/setup.sh --resume"

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
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
        resume_deploy
        ;;
    help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  (none)          Fresh install (interactive)"
        echo "  --config FILE   Non-interactive install using config file"
        echo "  --reconfigure   Re-enter configuration"
        echo "  --status        Show service status"
        echo "  --upgrade       Update image versions"
        echo "  --uninstall     Uninstall (with confirmation)"
        echo "  --resume        Resume from failed phase"
        echo "  --verbose, -v   Show detailed output"
        echo "  --help          Show this help"
        ;;
esac
