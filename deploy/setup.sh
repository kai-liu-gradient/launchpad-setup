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
while [[ $# -gt 0 ]]; do
    case $1 in
        --reconfigure) ACTION="reconfigure"; shift ;;
        --status)      ACTION="status";      shift ;;
        --upgrade)     ACTION="upgrade";     shift ;;
        --uninstall)   ACTION="uninstall";   shift ;;
        --resume)      ACTION="resume";      shift ;;
        --help|-h)     ACTION="help";        shift ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# Load previous config if reconfiguring
if [[ "$ACTION" == "reconfigure" ]] && [[ -f "${DEPLOY_DIR}/generated/.setup.conf" ]]; then
    source "${DEPLOY_DIR}/generated/.setup.conf"
fi

case "$ACTION" in
    install|reconfigure)
        # Banner
        print_banner "AniLaunchpad Setup" "$SETUP_VERSION"

        # Language selection
        select_language

        # Load remaining modules
        source "${DEPLOY_DIR}/scripts/lib/detect.sh"
        source "${DEPLOY_DIR}/scripts/lib/interact.sh"
        source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
        source "${DEPLOY_DIR}/scripts/lib/render.sh"
        source "${DEPLOY_DIR}/scripts/lib/certs.sh"
        source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
        source "${DEPLOY_DIR}/scripts/lib/database.sh"
        source "${DEPLOY_DIR}/scripts/lib/deploy.sh"

        # Phase 1: Detect environment
        check_environment

        # Phase 2: Collect configuration
        collect_basic_config
        collect_advanced_config

        # Phase 3: Confirm
        show_summary
        if ! ask_confirm "$MSG_DEPLOY_CONFIRM" "Y"; then
            log_warn "Deployment cancelled."
            exit 0
        fi

        # Phase 4: Generate
        if ! load_secrets; then
            generate_secrets
        fi
        render_templates
        save_config
        save_secrets

        # Phase 5: Deploy
        setup_kubernetes
        setup_certificates
        deploy_services
        bootstrap_gitea
        register_cluster

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
        echo "  --reconfigure   Re-enter configuration"
        echo "  --status        Show service status"
        echo "  --upgrade       Update image versions"
        echo "  --uninstall     Uninstall (with confirmation)"
        echo "  --resume        Resume from failed phase"
        echo "  --help          Show this help"
        ;;
esac
