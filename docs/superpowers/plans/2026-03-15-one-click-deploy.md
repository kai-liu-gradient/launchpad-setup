# One-Click Deployment Script Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a modular, interactive Bash deployment script that automates full AniLaunchpad platform deployment on a single Linux server.

**Architecture:** Modular Bash scripts under `deploy/` — a main entry point (`setup.sh`) orchestrates library modules for environment detection, interactive input, secrets generation, template rendering, SSL certificates, k3s/K8s setup, database initialization, and service deployment. Templates produce all configuration files; `generated/` holds runtime output.

**Tech Stack:** Bash, Docker Compose, Nginx, PostgreSQL, Redis, k3s, acme.sh, envsubst, openssl, curl, jq

**Spec:** `docs/superpowers/specs/2026-03-15-one-click-deploy-design.md`

---

## Chunk 1: Foundation (common.sh, i18n.sh, lang packs, versions.conf, setup.sh skeleton)

### Task 1: Create directory structure and versions.conf

**Files:**
- Create: `deploy/versions.conf`
- Create: `deploy/.gitignore`

- [ ] **Step 1: Create deploy directory structure**

```bash
mkdir -p deploy/scripts/lib deploy/scripts/lang deploy/templates deploy/generated
```

- [ ] **Step 2: Create versions.conf**

Create `deploy/versions.conf`:
```bash
# Default image versions — edit this file to update, no script changes needed
IMAGE_REGISTRY=ghcr.io/anilaunchpad
IMAGE_VERSION=1.18.7

# Per-service overrides (leave empty to use IMAGE_VERSION)
IMAGE_VERSION_API=
IMAGE_VERSION_UI=1.18.6
IMAGE_VERSION_ROUTER=1.16.2
IMAGE_VERSION_GATEWAY=
IMAGE_VERSION_GITEA=1.21
```

- [ ] **Step 3: Create deploy/.gitignore**

```
generated/
```

- [ ] **Step 4: Commit**

```bash
git add deploy/
git commit -m "feat: scaffold deploy directory structure and versions.conf"
```

---

### Task 2: Create common.sh — shared utility functions

**Files:**
- Create: `deploy/scripts/lib/common.sh`

This module provides: color output, logging (log_info, log_warn, log_error, log_ok, log_step), input helpers (ask, ask_default, ask_password, ask_choice, ask_multichoice), validation helpers (validate_domain, validate_port, validate_ip, validate_file_exists).

- [ ] **Step 1: Write common.sh**

Create `deploy/scripts/lib/common.sh` with all utility functions. Key functions:

```bash
#!/bin/bash
# Common utility functions for AniLaunchpad setup

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
```

- [ ] **Step 2: Verify common.sh is sourceable**

```bash
bash -n deploy/scripts/lib/common.sh && echo "Syntax OK"
```

Expected: `Syntax OK`

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/common.sh
git commit -m "feat: add common.sh utility functions"
```

---

### Task 3: Create i18n.sh and language packs

**Files:**
- Create: `deploy/scripts/lib/i18n.sh`
- Create: `deploy/scripts/lang/en.sh`
- Create: `deploy/scripts/lang/zh.sh`

- [ ] **Step 1: Write i18n.sh**

Create `deploy/scripts/lib/i18n.sh`:
```bash
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
```

- [ ] **Step 2: Write en.sh language pack**

Create `deploy/scripts/lang/en.sh` with all MSG_ variables. Include every user-facing string organized by section:

```bash
#!/bin/bash
# English language pack

# Banner
MSG_WELCOME="AniLaunchpad Setup Script"
MSG_VERSION="1.0"

# Step headers
MSG_STEP_BASIC="Basic Configuration"
MSG_STEP_SSL="SSL Certificate"
MSG_STEP_DB="Database"
MSG_STEP_K8S="Kubernetes Cluster"
MSG_STEP_ADVANCED="Advanced Configuration"
MSG_STEP_SUMMARY="Deployment Summary"

# Domain
MSG_DOMAIN_PROMPT="Main domain (e.g. company.com)"
MSG_SUBDOMAIN_PROMPT="Subdomain prefix"
MSG_DOMAIN_CONFIRM="The following domains will be generated"
MSG_DOMAIN_DASHBOARD="Dashboard"
MSG_DOMAIN_GITEA="Gitea"
MSG_DOMAIN_PROJECTS="Project access"
MSG_CONFIRM="Confirm?"

# Image
MSG_REGISTRY_PROMPT="Image registry"
MSG_VERSION_PROMPT="Image version"

# SSL
MSG_SSL_METHOD="Certificate method"
MSG_SSL_LETSENCRYPT="Let's Encrypt auto-apply (recommended)"
MSG_SSL_CUSTOM="Provide certificate files"
MSG_SSL_DNS_PROVIDER="DNS provider"
MSG_SSL_DNS_CLOUDFLARE="Cloudflare"
MSG_SSL_DNS_ALIYUN="Alibaba Cloud DNS"
MSG_SSL_DNS_AZURE="Azure DNS"
MSG_SSL_DNS_OTHER="Other (manual DNS verification)"
MSG_SSL_API_TOKEN="API Token"
MSG_SSL_CERT_PATH="Certificate file path (.crt/.pem)"
MSG_SSL_KEY_PATH="Private key file path (.key)"

# Database
MSG_DB_MODE="Database mode"
MSG_DB_BUILTIN="Built-in PostgreSQL + Redis (recommended)"
MSG_DB_EXTERNAL="Use external database"
MSG_DB_HOST="PostgreSQL host"
MSG_DB_PORT="PostgreSQL port"
MSG_DB_REDIS_HOST="Redis host"
MSG_DB_REDIS_PORT="Redis port"
MSG_DB_REDIS_PASSWORD="Redis password"

# K8s
MSG_K8S_MODE="K8s mode"
MSG_K8S_BUILTIN="Built-in k3s (recommended, auto-install on this machine)"
MSG_K8S_EXTERNAL="Use existing K8s cluster"
MSG_K8S_KUBECONFIG="kubeconfig file path"
MSG_K8S_CONTEXT="K8s Context name (leave empty for current)"
MSG_K8S_INGRESS="Ingress backend domain"
MSG_K8S_STORAGE_CLASS="Storage Class"
MSG_K8S_VERIFIED="Cluster connection verified"

# Advanced
MSG_ADV_ENTER="Enter advanced configuration?"
MSG_ADV_SELECT="Select modules to configure (comma-separated, Enter to skip all)"
MSG_ADV_ADMIN="Admin account"
MSG_ADV_ADMIN_DESC="Initial admin email and password"
MSG_ADV_SMTP="Email (SMTP)"
MSG_ADV_SMTP_DESC="Email sending configuration"
MSG_ADV_STRIPE="Payment integration"
MSG_ADV_STRIPE_DESC="Stripe payment keys"
MSG_ADV_SSO="Enterprise SSO"
MSG_ADV_SSO_DESC="Microsoft Entra ID"
MSG_ADV_AI="AI services"
MSG_ADV_AI_DESC="CRS2 / PayGO credential config"
MSG_ADV_NOTIFY="Notifications"
MSG_ADV_NOTIFY_DESC="Telegram Bot"
MSG_ADV_PERF="Performance tuning"
MSG_ADV_PERF_DESC="API replicas, connection pool, timeouts"
MSG_ADV_ALL="Configure all"
MSG_ADV_SKIPPED="Skipped"
MSG_ADV_RECONFIGURE="Reconfigure later with"
MSG_ADV_CONFIGURED="Configured"

# Admin
MSG_ADMIN_EMAIL="Admin email"
MSG_ADMIN_PASSWORD="Admin password (leave empty to auto-generate)"

# SMTP
MSG_SMTP_HOST="SMTP server"
MSG_SMTP_PORT="SMTP port"
MSG_SMTP_USER="SMTP username"
MSG_SMTP_PASS="SMTP password"
MSG_SMTP_FROM="Sender address"

# Image versions (advanced)
MSG_IMG_UNIFIED="Unified version"
MSG_IMG_OVERRIDE="Override per-service (leave empty to keep unified version)"

# Deploy
MSG_DEPLOY_CONFIRM="Start deployment?"
MSG_DEPLOY_STARTING="Starting deployment..."
MSG_DEPLOY_INFRA="Starting infrastructure..."
MSG_DEPLOY_DB_INIT="Initializing databases..."
MSG_DEPLOY_SERVICES="Starting application services..."
MSG_DEPLOY_NGINX="Starting Nginx..."
MSG_DEPLOY_GITEA="Bootstrapping Gitea..."
MSG_DEPLOY_REGISTER="Registering K8s cluster..."
MSG_DEPLOY_COMPLETE="AniLaunchpad deployment complete!"
MSG_DEPLOY_FAILED="Deployment failed"

# Output
MSG_OUT_URLS="Access URLs"
MSG_OUT_DASHBOARD="Dashboard"
MSG_OUT_GITEA="Gitea"
MSG_OUT_ADMIN="Admin"
MSG_OUT_CREDENTIALS="Admin credentials"
MSG_OUT_EMAIL="Email"
MSG_OUT_PASSWORD="Password"
MSG_OUT_K8S="K8s cluster"
MSG_OUT_MODE="Mode"
MSG_OUT_STATUS="Status"
MSG_OUT_DNS="DNS reminder"
MSG_OUT_DNS_MSG="Ensure these DNS records point to this server"
MSG_OUT_CONFIG="Config files"
MSG_OUT_LOGS="View logs"
MSG_OUT_RECONFIG="Reconfigure"

# Detect
MSG_DETECT_CHECKING="Checking environment..."
MSG_DETECT_OS="Operating system"
MSG_DETECT_DOCKER="Docker"
MSG_DETECT_DOCKER_COMPOSE="Docker Compose"
MSG_DETECT_DISK="Disk space"
MSG_DETECT_MEMORY="Available memory"
MSG_DETECT_PORTS="Required ports"
MSG_DETECT_FAIL="Environment check failed"

# Health
MSG_HEALTH_WAITING="Waiting for"
MSG_HEALTH_HEALTHY="healthy"
MSG_HEALTH_FAILED="failed to become healthy"

# Errors
MSG_ERR_DOMAIN_INVALID="Invalid domain format"
MSG_ERR_FILE_NOT_FOUND="File not found"
MSG_ERR_PORT_IN_USE="Port already in use"
MSG_ERR_DOCKER_MISSING="Docker is not installed"
MSG_ERR_RESUME="To retry from this phase"
MSG_ERR_LOGS="To view logs"
MSG_ERR_UNINSTALL="To tear down everything"
```

- [ ] **Step 3: Write zh.sh language pack**

Create `deploy/scripts/lang/zh.sh` — same MSG_ variable names, Chinese values:

```bash
#!/bin/bash
# 中文语言包

# Banner
MSG_WELCOME="AniLaunchpad 一键部署脚本"
MSG_VERSION="1.0"

# Step headers
MSG_STEP_BASIC="基础配置"
MSG_STEP_SSL="SSL 证书"
MSG_STEP_DB="数据库"
MSG_STEP_K8S="Kubernetes 集群"
MSG_STEP_ADVANCED="高级配置"
MSG_STEP_SUMMARY="部署摘要"

# Domain
MSG_DOMAIN_PROMPT="主域名 (例: company.com)"
MSG_SUBDOMAIN_PROMPT="子域名前缀"
MSG_DOMAIN_CONFIRM="将生成以下域名"
MSG_DOMAIN_DASHBOARD="管理面板"
MSG_DOMAIN_GITEA="代码托管"
MSG_DOMAIN_PROJECTS="项目访问"
MSG_CONFIRM="确认？"

# Image
MSG_REGISTRY_PROMPT="镜像仓库地址"
MSG_VERSION_PROMPT="镜像版本"

# SSL
MSG_SSL_METHOD="证书获取方式"
MSG_SSL_LETSENCRYPT="Let's Encrypt 自动申请（推荐）"
MSG_SSL_CUSTOM="提供证书文件"
MSG_SSL_DNS_PROVIDER="DNS 服务商"
MSG_SSL_DNS_CLOUDFLARE="Cloudflare"
MSG_SSL_DNS_ALIYUN="阿里云 DNS"
MSG_SSL_DNS_AZURE="Azure DNS"
MSG_SSL_DNS_OTHER="其他（手动 DNS 验证）"
MSG_SSL_API_TOKEN="API Token"
MSG_SSL_CERT_PATH="证书文件路径 (.crt/.pem)"
MSG_SSL_KEY_PATH="私钥文件路径 (.key)"

# Database
MSG_DB_MODE="数据库模式"
MSG_DB_BUILTIN="内置 PostgreSQL + Redis（推荐）"
MSG_DB_EXTERNAL="使用外部数据库"
MSG_DB_HOST="PostgreSQL 主机"
MSG_DB_PORT="PostgreSQL 端口"
MSG_DB_REDIS_HOST="Redis 主机"
MSG_DB_REDIS_PORT="Redis 端口"
MSG_DB_REDIS_PASSWORD="Redis 密码"

# K8s
MSG_K8S_MODE="K8s 模式"
MSG_K8S_BUILTIN="内置 k3s（推荐，自动安装到本机）"
MSG_K8S_EXTERNAL="使用已有 K8s 集群"
MSG_K8S_KUBECONFIG="kubeconfig 文件路径"
MSG_K8S_CONTEXT="K8s Context 名称（留空使用当前）"
MSG_K8S_INGRESS="Ingress 后端域名"
MSG_K8S_STORAGE_CLASS="Storage Class"
MSG_K8S_VERIFIED="集群连接已验证"

# Advanced
MSG_ADV_ENTER="是否进入高级配置？"
MSG_ADV_SELECT="选择要配置的模块（逗号分隔，回车跳过）"
MSG_ADV_ADMIN="管理员账户"
MSG_ADV_ADMIN_DESC="初始管理员邮箱和密码"
MSG_ADV_SMTP="邮件 (SMTP)"
MSG_ADV_SMTP_DESC="邮件发送配置"
MSG_ADV_STRIPE="支付集成"
MSG_ADV_STRIPE_DESC="Stripe 支付密钥"
MSG_ADV_SSO="企业 SSO"
MSG_ADV_SSO_DESC="Microsoft Entra ID"
MSG_ADV_AI="AI 服务"
MSG_ADV_AI_DESC="CRS2 / PayGO 凭证配置"
MSG_ADV_NOTIFY="通知"
MSG_ADV_NOTIFY_DESC="Telegram Bot"
MSG_ADV_PERF="性能调优"
MSG_ADV_PERF_DESC="API 副本数、连接池、超时时间"
MSG_ADV_ALL="配置全部"
MSG_ADV_SKIPPED="已跳过"
MSG_ADV_RECONFIGURE="稍后重新配置"
MSG_ADV_CONFIGURED="已配置"

# Admin
MSG_ADMIN_EMAIL="管理员邮箱"
MSG_ADMIN_PASSWORD="管理员密码（留空自动生成）"

# SMTP
MSG_SMTP_HOST="SMTP 服务器"
MSG_SMTP_PORT="SMTP 端口"
MSG_SMTP_USER="SMTP 用户名"
MSG_SMTP_PASS="SMTP 密码"
MSG_SMTP_FROM="发件人地址"

# Image versions (advanced)
MSG_IMG_UNIFIED="统一版本"
MSG_IMG_OVERRIDE="按服务覆盖版本（留空保持统一版本）"

# Deploy
MSG_DEPLOY_CONFIRM="开始部署？"
MSG_DEPLOY_STARTING="开始部署..."
MSG_DEPLOY_INFRA="启动基础设施..."
MSG_DEPLOY_DB_INIT="初始化数据库..."
MSG_DEPLOY_SERVICES="启动应用服务..."
MSG_DEPLOY_NGINX="启动 Nginx..."
MSG_DEPLOY_GITEA="初始化 Gitea..."
MSG_DEPLOY_REGISTER="注册 K8s 集群..."
MSG_DEPLOY_COMPLETE="AniLaunchpad 部署完成！"
MSG_DEPLOY_FAILED="部署失败"

# Output
MSG_OUT_URLS="访问地址"
MSG_OUT_DASHBOARD="管理面板"
MSG_OUT_GITEA="代码托管"
MSG_OUT_ADMIN="后台管理"
MSG_OUT_CREDENTIALS="管理员凭证"
MSG_OUT_EMAIL="邮箱"
MSG_OUT_PASSWORD="密码"
MSG_OUT_K8S="K8s 集群"
MSG_OUT_MODE="模式"
MSG_OUT_STATUS="状态"
MSG_OUT_DNS="DNS 提醒"
MSG_OUT_DNS_MSG="请确保以下 DNS 记录指向本服务器"
MSG_OUT_CONFIG="配置文件"
MSG_OUT_LOGS="查看日志"
MSG_OUT_RECONFIG="重新配置"

# Detect
MSG_DETECT_CHECKING="检查环境..."
MSG_DETECT_OS="操作系统"
MSG_DETECT_DOCKER="Docker"
MSG_DETECT_DOCKER_COMPOSE="Docker Compose"
MSG_DETECT_DISK="磁盘空间"
MSG_DETECT_MEMORY="可用内存"
MSG_DETECT_PORTS="所需端口"
MSG_DETECT_FAIL="环境检查未通过"

# Health
MSG_HEALTH_WAITING="等待"
MSG_HEALTH_HEALTHY="已就绪"
MSG_HEALTH_FAILED="启动超时"

# Errors
MSG_ERR_DOMAIN_INVALID="域名格式无效"
MSG_ERR_FILE_NOT_FOUND="文件未找到"
MSG_ERR_PORT_IN_USE="端口已被占用"
MSG_ERR_DOCKER_MISSING="未安装 Docker"
MSG_ERR_RESUME="从当前阶段重试"
MSG_ERR_LOGS="查看日志"
MSG_ERR_UNINSTALL="卸载全部"
```

- [ ] **Step 4: Verify both language files are sourceable**

```bash
bash -n deploy/scripts/lang/en.sh && bash -n deploy/scripts/lang/zh.sh && echo "Both OK"
```

Expected: `Both OK`

- [ ] **Step 5: Commit**

```bash
git add deploy/scripts/lib/i18n.sh deploy/scripts/lang/
git commit -m "feat: add i18n support with English and Chinese language packs"
```

---

### Task 4: Create setup.sh skeleton

**Files:**
- Create: `deploy/setup.sh`

- [ ] **Step 1: Write setup.sh main entry point**

Create `deploy/setup.sh`:
```bash
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
        generate_secrets
        render_templates
        save_config

        # Phase 5: Deploy
        setup_kubernetes
        setup_certificates
        init_database
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
```

- [ ] **Step 2: Make setup.sh executable**

```bash
chmod +x deploy/setup.sh
```

- [ ] **Step 3: Verify syntax**

```bash
bash -n deploy/setup.sh && echo "Syntax OK"
```

Expected: `Syntax OK` (will fail at runtime since modules don't exist yet, but syntax is valid)

- [ ] **Step 4: Commit**

```bash
git add deploy/setup.sh
git commit -m "feat: add setup.sh main entry point with CLI argument parsing"
```

---

## Chunk 2: detect.sh, secrets.sh, interact.sh

### Task 5: Create detect.sh — environment pre-checks

**Files:**
- Create: `deploy/scripts/lib/detect.sh`

- [ ] **Step 1: Write detect.sh**

Create `deploy/scripts/lib/detect.sh`. Checks: OS (Linux required), Docker installed + running, Docker Compose v2, available disk (min 10GB), available memory (min 2GB), required ports (80, 443, 5432, 6379, 6555, 6801, 6802, 6804, 3000, 2222).

```bash
#!/bin/bash
# Environment detection and pre-flight checks

check_environment() {
    log_step "0/6" "$MSG_DETECT_CHECKING"

    local errors=0

    # OS check
    if [[ "$(uname -s)" != "Linux" ]]; then
        log_error "$MSG_DETECT_OS: $(uname -s) — Linux required"
        ((errors++))
    else
        log_ok "$MSG_DETECT_OS: $(uname -s) $(uname -r)"
    fi

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

    # Disk space (min 10GB free)
    local free_gb
    free_gb=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    if [[ "$free_gb" -lt 10 ]]; then
        log_error "$MSG_DETECT_DISK: ${free_gb}GB free (minimum 10GB)"
        ((errors++))
    else
        log_ok "$MSG_DETECT_DISK: ${free_gb}GB free"
    fi

    # Memory (min 2GB)
    local mem_mb
    mem_mb=$(free -m | awk '/^Mem:/ {print $7}')
    if [[ "$mem_mb" -lt 2048 ]]; then
        log_warn "$MSG_DETECT_MEMORY: ${mem_mb}MB available (recommended 4GB+)"
    else
        log_ok "$MSG_DETECT_MEMORY: ${mem_mb}MB available"
    fi

    # Required ports
    local ports=(80 443 5432 6379 6555 6801 6802 6804 3000 2222)
    local port_errors=0
    for port in "${ports[@]}"; do
        if ss -tlnp 2>/dev/null | grep -q ":${port} "; then
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

# Detect internal IP (for k3s tls-san and DNS reminder)
detect_internal_ip() {
    local ip=""
    # Try common interfaces
    for iface in lan0 eth0 ens3 ens5; do
        ip=$(ip addr show "$iface" 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | head -1)
        [[ -n "$ip" ]] && echo "$ip" && return 0
    done
    # Fallback: first non-localhost
    ip=$(ip -4 addr show | grep "inet " | awk '{print $2}' | cut -d'/' -f1 | grep -v '^127\.' | head -1)
    [[ -n "$ip" ]] && echo "$ip" && return 0
    # Last resort
    ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    [[ -n "$ip" && "$ip" != "127.0.0.1" ]] && echo "$ip" && return 0
    return 1
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/detect.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/detect.sh
git commit -m "feat: add detect.sh environment pre-checks"
```

---

### Task 6: Create secrets.sh — key and password generation

**Files:**
- Create: `deploy/scripts/lib/secrets.sh`

- [ ] **Step 1: Write secrets.sh**

Create `deploy/scripts/lib/secrets.sh`. Generates all secrets per spec (JWT, ZT RSA keypair, DB passwords, encryption keys, gateway API key, internal secret).

```bash
#!/bin/bash
# Secret and password generation

generate_secrets() {
    log_info "Generating secrets and passwords..."

    # JWT
    JWT_SECRET=$(openssl rand -hex 32)
    JWT_REFRESH_SECRET=$(openssl rand -hex 32)
    SESSION_SECRET=$(openssl rand -hex 32)

    # Encryption keys (exactly 32 chars each)
    ENCRYPTION_KEY=$(openssl rand -hex 16)
    CLAUDE_CREDENTIALS_ENCRYPTION_KEY=$(openssl rand -hex 16)
    SSH_KEY_ENCRYPTION_SECRET=$(openssl rand -hex 16)

    # Zero Trust RSA keypair
    local zt_private_pem zt_public_pem
    zt_private_pem=$(openssl genrsa 2048 2>/dev/null)
    zt_public_pem=$(echo "$zt_private_pem" | openssl rsa -pubout 2>/dev/null)
    ZT_PRIVATE_KEY=$(echo "$zt_private_pem" | base64 -w 0 2>/dev/null || echo "$zt_private_pem" | base64)
    ZT_PUBLIC_KEY=$(echo "$zt_public_pem" | base64 -w 0 2>/dev/null || echo "$zt_public_pem" | base64)

    # Database passwords
    DB_PASSWORD_MAIN=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_MONITORING=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_EVENTS=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_BILLING=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_STATS=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    DB_PASSWORD_GATEWAY=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    GITEA_DB_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)
    POSTGRES_SUPERUSER_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)

    # Redis
    REDIS_PASSWORD=$(openssl rand -base64 18 | tr -d '=/+' | head -c 24)

    # Gateway
    ANI_CODE_GATEWAY_API_KEY=$(openssl rand -hex 16)
    LAUNCHPAD_INTERNAL_SECRET=$(openssl rand -hex 16)

    # Admin password (if not already set by user)
    if [[ -z "${ADMIN_PASSWORD:-}" ]]; then
        ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d '=/+' | head -c 16)
        ADMIN_PASSWORD_GENERATED=true
    fi

    log_ok "Secrets generated"
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/secrets.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/secrets.sh
git commit -m "feat: add secrets.sh key and password generation"
```

---

### Task 7: Create interact.sh — interactive configuration collection

**Files:**
- Create: `deploy/scripts/lib/interact.sh`

This is the largest module. It implements the 6-step interactive flow from the spec.

- [ ] **Step 1: Write interact.sh**

Create `deploy/scripts/lib/interact.sh` with `collect_basic_config()`, `collect_advanced_config()`, `show_summary()`, `save_config()`. The basic flow:

1. Domain + subdomain → derive LAUNCHPAD_DOMAIN, GITEA_DOMAIN
2. Image registry + version
3. SSL method (letsencrypt/custom)
4. Database mode (builtin/external)
5. K8s mode (builtin/external)
6. Advanced config (multi-select menu → per-module Q&A)
7. Summary display

Each section uses MSG_ variables from the language pack and the ask_* helpers from common.sh.

Key functions:

```bash
#!/bin/bash
# Interactive configuration collection

collect_basic_config() {
    # Step 1: Domain
    log_step "1/6" "$MSG_STEP_BASIC"

    while true; do
        DOMAIN=$(ask_default "$MSG_DOMAIN_PROMPT" "${DOMAIN:-}")
        if validate_domain "$DOMAIN"; then
            break
        fi
        log_error "$MSG_ERR_DOMAIN_INVALID: $DOMAIN"
    done

    SUBDOMAIN=$(ask_default "$MSG_SUBDOMAIN_PROMPT" "${SUBDOMAIN:-corp}")
    LAUNCHPAD_DOMAIN="launchpad.${SUBDOMAIN}.${DOMAIN}"
    GITEA_DOMAIN="launchpad-gitea.${SUBDOMAIN}.${DOMAIN}"

    echo ""
    echo "  $MSG_DOMAIN_CONFIRM:"
    log_ok "${LAUNCHPAD_DOMAIN}  ($MSG_DOMAIN_DASHBOARD)"
    log_ok "${GITEA_DOMAIN}  ($MSG_DOMAIN_GITEA)"
    log_ok "*.${DOMAIN}  ($MSG_DOMAIN_PROJECTS)"
    echo ""

    if ! ask_confirm "$MSG_CONFIRM" "Y"; then
        collect_basic_config  # Retry
        return
    fi

    # Image registry and version
    IMAGE_REGISTRY=$(ask_default "$MSG_REGISTRY_PROMPT" "${IMAGE_REGISTRY}")
    IMAGE_VERSION=$(ask_default "$MSG_VERSION_PROMPT" "${IMAGE_VERSION}")

    # Step 2: SSL
    log_step "2/6" "$MSG_STEP_SSL"
    SSL_MODE=$(ask_choice "$MSG_SSL_METHOD" "1" "$MSG_SSL_LETSENCRYPT" "$MSG_SSL_CUSTOM")

    if [[ "$SSL_MODE" == "1" ]]; then
        SSL_MODE="letsencrypt"
        DNS_PROVIDER=$(ask_choice "$MSG_SSL_DNS_PROVIDER" "1" \
            "$MSG_SSL_DNS_CLOUDFLARE" "$MSG_SSL_DNS_ALIYUN" "$MSG_SSL_DNS_AZURE" "$MSG_SSL_DNS_OTHER")
        case "$DNS_PROVIDER" in
            1) DNS_PROVIDER="cloudflare" ;;
            2) DNS_PROVIDER="aliyun" ;;
            3) DNS_PROVIDER="azure" ;;
            *) DNS_PROVIDER="manual" ;;
        esac
        if [[ "$DNS_PROVIDER" != "manual" ]]; then
            DNS_API_TOKEN=$(ask_default "$MSG_SSL_API_TOKEN" "")
        fi
    else
        SSL_MODE="custom"
        SSL_CERT_PATH=$(ask_default "$MSG_SSL_CERT_PATH" "")
        SSL_KEY_PATH=$(ask_default "$MSG_SSL_KEY_PATH" "")
        SSL_WILDCARD_CERT_PATH=$(ask_default "Wildcard $MSG_SSL_CERT_PATH" "")
        SSL_WILDCARD_KEY_PATH=$(ask_default "Wildcard $MSG_SSL_KEY_PATH" "")
    fi

    # Step 3: Database
    log_step "3/6" "$MSG_STEP_DB"
    DB_MODE=$(ask_choice "$MSG_DB_MODE" "1" "$MSG_DB_BUILTIN" "$MSG_DB_EXTERNAL")

    if [[ "$DB_MODE" == "2" ]]; then
        DB_MODE="external"
        DB_HOST=$(ask_default "$MSG_DB_HOST" "${DB_HOST:-}")
        DB_PORT=$(ask_default "$MSG_DB_PORT" "${DB_PORT:-5432}")
        REDIS_HOST=$(ask_default "$MSG_DB_REDIS_HOST" "${REDIS_HOST:-}")
        REDIS_PORT=$(ask_default "$MSG_DB_REDIS_PORT" "${REDIS_PORT:-6379}")
        REDIS_PASSWORD=$(ask_default "$MSG_DB_REDIS_PASSWORD" "")
    else
        DB_MODE="builtin"
        DB_HOST="postgres"
        DB_PORT="5432"
        REDIS_HOST="redis"
        REDIS_PORT="6379"
    fi

    # Step 4: K8s
    log_step "4/6" "$MSG_STEP_K8S"
    K8S_MODE=$(ask_choice "$MSG_K8S_MODE" "1" "$MSG_K8S_BUILTIN" "$MSG_K8S_EXTERNAL")

    if [[ "$K8S_MODE" == "2" ]]; then
        K8S_MODE="external"
        K8S_KUBECONFIG_PATH=$(ask_default "$MSG_K8S_KUBECONFIG" "${K8S_KUBECONFIG_PATH:-~/.kube/config}")
        K8S_CONTEXT=$(ask_default "$MSG_K8S_CONTEXT" "")
        K8S_INGRESS_DOMAIN=$(ask_default "$MSG_K8S_INGRESS" "")
        STORAGE_CLASS=$(ask_default "$MSG_K8S_STORAGE_CLASS" "${STORAGE_CLASS:-standard}")
        DEFAULT_BACKEND="$K8S_INGRESS_DOMAIN"
    else
        K8S_MODE="builtin"
        STORAGE_CLASS="local-path"
        DEFAULT_BACKEND="localhost"
    fi
}

collect_advanced_config() {
    log_step "5/6" "$MSG_STEP_ADVANCED"

    if ! ask_confirm "$MSG_ADV_ENTER" "N"; then
        # Set defaults for skipped modules
        ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"
        return
    fi

    echo "    1) $MSG_ADV_ADMIN - $MSG_ADV_ADMIN_DESC"
    echo "    2) $MSG_ADV_SMTP - $MSG_ADV_SMTP_DESC"
    echo "    3) $MSG_ADV_STRIPE - $MSG_ADV_STRIPE_DESC"
    echo "    4) $MSG_ADV_SSO - $MSG_ADV_SSO_DESC"
    echo "    5) $MSG_ADV_AI - $MSG_ADV_AI_DESC"
    echo "    6) $MSG_ADV_NOTIFY - $MSG_ADV_NOTIFY_DESC"
    echo "    7) $MSG_ADV_PERF - $MSG_ADV_PERF_DESC"
    echo "    8) $MSG_ADV_ALL"
    local selected
    selected=$(ask_multichoice "$MSG_ADV_SELECT")

    # Parse selection — use comma-delimited word-boundary matching
    [[ "$selected" == "8" ]] && selected="1,2,3,4,5,6,7"
    # Convert to comma-delimited with leading/trailing commas for safe matching
    local sel=",${selected//[[:space:]]/},"

    # Set defaults
    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@${DOMAIN}}"

    # Module 1: Admin
    if [[ "$sel" == *",1,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_ADMIN ──"
        ADMIN_EMAIL=$(ask_default "$MSG_ADMIN_EMAIL" "$ADMIN_EMAIL")
        ADMIN_PASSWORD=$(ask_password "$MSG_ADMIN_PASSWORD")
        [[ -z "$ADMIN_PASSWORD" ]] && ADMIN_PASSWORD=""  # Will be auto-generated in secrets.sh
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 2: SMTP
    if [[ "$sel" == *",2,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_SMTP ──"
        SMTP_HOST=$(ask_default "$MSG_SMTP_HOST" "${SMTP_HOST:-smtp.gmail.com}")
        SMTP_PORT=$(ask_default "$MSG_SMTP_PORT" "${SMTP_PORT:-587}")
        SMTP_USER=$(ask_default "$MSG_SMTP_USER" "${SMTP_USER:-}")
        SMTP_PASS=$(ask_password "$MSG_SMTP_PASS")
        EMAIL_FROM=$(ask_default "$MSG_SMTP_FROM" "noreply@${DOMAIN}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 3: Stripe
    if [[ "$sel" == *",3,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_STRIPE ──"
        STRIPE_SECRET_KEY=$(ask_default "Stripe Secret Key" "${STRIPE_SECRET_KEY:-}")
        STRIPE_PUBLISHABLE_KEY=$(ask_default "Stripe Publishable Key" "${STRIPE_PUBLISHABLE_KEY:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 4: SSO
    if [[ "$sel" == *",4,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_SSO ──"
        ENTRA_ENABLED="true"
        ENTRA_CLIENT_ID=$(ask_default "Entra Client ID" "${ENTRA_CLIENT_ID:-}")
        ENTRA_CLIENT_SECRET=$(ask_password "Entra Client Secret")
        ENTRA_TENANT_ID=$(ask_default "Entra Tenant ID" "${ENTRA_TENANT_ID:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 5: AI
    if [[ "$sel" == *",5,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_AI ──"
        CRS2_API_URL=$(ask_default "CRS2 API URL" "${CRS2_API_URL:-}")
        CRS2_API_KEY=$(ask_default "CRS2 API Key" "${CRS2_API_KEY:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 6: Notifications
    if [[ "$sel" == *",6,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_NOTIFY ──"
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi

    # Module 7: Performance
    if [[ "$sel" == *",7,"* ]]; then
        echo ""
        echo "  ── $MSG_ADV_PERF ──"
        API_REPLICAS=$(ask_default "API replicas" "${API_REPLICAS:-1}")
        DB_CONNECTION_LIMIT=$(ask_default "DB connection limit per container" "${DB_CONNECTION_LIMIT:-10}")

        echo ""
        echo "  ── $MSG_IMG_UNIFIED: $IMAGE_VERSION ──"
        echo "  $MSG_IMG_OVERRIDE:"
        IMAGE_VERSION_API=$(ask_default "  API" "${IMAGE_VERSION_API:-$IMAGE_VERSION}")
        IMAGE_VERSION_UI=$(ask_default "  UI" "${IMAGE_VERSION_UI:-$IMAGE_VERSION}")
        IMAGE_VERSION_ROUTER=$(ask_default "  Router" "${IMAGE_VERSION_ROUTER:-$IMAGE_VERSION}")
        IMAGE_VERSION_GATEWAY=$(ask_default "  Gateway" "${IMAGE_VERSION_GATEWAY:-$IMAGE_VERSION}")
        IMAGE_VERSION_GITEA=$(ask_default "  Gitea" "${IMAGE_VERSION_GITEA:-$IMAGE_VERSION}")
        log_ok "$MSG_ADV_CONFIGURED"
    fi
}

show_summary() {
    log_step "6/6" "$MSG_STEP_SUMMARY"

    echo "  ┌──────────────────────────────────────┐"
    printf "  │ %-14s %-22s │\n" "Domain:" "$DOMAIN"
    printf "  │ %-14s %-22s │\n" "Dashboard:" "$LAUNCHPAD_DOMAIN"
    printf "  │ %-14s %-22s │\n" "SSL:" "$SSL_MODE"
    printf "  │ %-14s %-22s │\n" "Database:" "$DB_MODE"
    printf "  │ %-14s %-22s │\n" "K8s:" "$K8S_MODE"
    printf "  │ %-14s %-22s │\n" "Registry:" "$IMAGE_REGISTRY"
    printf "  │ %-14s %-22s │\n" "Version:" "$IMAGE_VERSION"
    echo "  └──────────────────────────────────────┘"
}

save_config() {
    mkdir -p "${DEPLOY_DIR}/generated"
    cat > "${DEPLOY_DIR}/generated/.setup.conf" <<CONF
# AniLaunchpad Setup Configuration — auto-generated, do not edit manually
# Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

DOMAIN="${DOMAIN}"
SUBDOMAIN="${SUBDOMAIN}"
LAUNCHPAD_DOMAIN="${LAUNCHPAD_DOMAIN}"
GITEA_DOMAIN="${GITEA_DOMAIN}"
IMAGE_REGISTRY="${IMAGE_REGISTRY}"
IMAGE_VERSION="${IMAGE_VERSION}"
IMAGE_VERSION_API="${IMAGE_VERSION_API:-}"
IMAGE_VERSION_UI="${IMAGE_VERSION_UI:-}"
IMAGE_VERSION_ROUTER="${IMAGE_VERSION_ROUTER:-}"
IMAGE_VERSION_GATEWAY="${IMAGE_VERSION_GATEWAY:-}"
IMAGE_VERSION_GITEA="${IMAGE_VERSION_GITEA:-}"
SSL_MODE="${SSL_MODE}"
SSL_CERT_PATH="${SSL_CERT_PATH:-}"
SSL_KEY_PATH="${SSL_KEY_PATH:-}"
SSL_WILDCARD_CERT_PATH="${SSL_WILDCARD_CERT_PATH:-}"
SSL_WILDCARD_KEY_PATH="${SSL_WILDCARD_KEY_PATH:-}"
DNS_PROVIDER="${DNS_PROVIDER:-}"
DB_MODE="${DB_MODE}"
DB_HOST="${DB_HOST}"
DB_PORT="${DB_PORT}"
REDIS_HOST="${REDIS_HOST}"
REDIS_PORT="${REDIS_PORT}"
K8S_MODE="${K8S_MODE}"
K8S_KUBECONFIG_PATH="${K8S_KUBECONFIG_PATH:-}"
K8S_CONTEXT="${K8S_CONTEXT:-}"
K8S_INGRESS_DOMAIN="${K8S_INGRESS_DOMAIN:-}"
STORAGE_CLASS="${STORAGE_CLASS}"
DEFAULT_BACKEND="${DEFAULT_BACKEND}"
ADMIN_EMAIL="${ADMIN_EMAIL}"
SMTP_HOST="${SMTP_HOST:-}"
SMTP_PORT="${SMTP_PORT:-}"
SMTP_USER="${SMTP_USER:-}"
EMAIL_FROM="${EMAIL_FROM:-}"
STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_PUBLISHABLE_KEY="${STRIPE_PUBLISHABLE_KEY:-}"
ENTRA_ENABLED="${ENTRA_ENABLED:-false}"
ENTRA_CLIENT_ID="${ENTRA_CLIENT_ID:-}"
ENTRA_TENANT_ID="${ENTRA_TENANT_ID:-}"
CRS2_API_URL="${CRS2_API_URL:-}"
CRS2_API_KEY="${CRS2_API_KEY:-}"
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
API_REPLICAS="${API_REPLICAS:-1}"
DB_CONNECTION_LIMIT="${DB_CONNECTION_LIMIT:-10}"
CONF
    log_ok "Configuration saved to generated/.setup.conf"
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/interact.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/interact.sh
git commit -m "feat: add interact.sh interactive configuration collection"
```

---

## Chunk 3: Templates (env, nginx, settings.yml, docker-compose)

### Task 8: Create env.template — Launchpad .env

**Files:**
- Create: `deploy/templates/env.template`

- [ ] **Step 1: Write env.template**

Create `deploy/templates/env.template` based on the existing `launchpad/.env` reference file, replacing all hardcoded values with `${VARIABLE}` placeholders. The template must include ALL of the following placeholder variables:

**Core:** `NODE_ENV`, `PORT` (6802), `ADMIN_PORT` (6804)
**Database (6 URLs):** `${DATABASE_URL}`, `${MONITORING_DATABASE_URL}`, `${EVENTS_DATABASE_URL}`, `${BILLING_DATABASE_URL}`, `${STATS_DATABASE_URL}`, `${GATEWAY_DATABASE_URL}`
**Redis:** `REDIS_HOST=${REDIS_HOST}`, `REDIS_PORT=${REDIS_PORT}`, `REDIS_PASSWORD=${REDIS_PASSWORD}`
**Security:** `${JWT_SECRET}`, `${JWT_REFRESH_SECRET}`, `${SESSION_SECRET}`, `${ENCRYPTION_KEY}`, `${CLAUDE_CREDENTIALS_ENCRYPTION_KEY}`, `${SSH_KEY_ENCRYPTION_SECRET}`, `${ZT_PRIVATE_KEY}`, `${ZT_PUBLIC_KEY}`
**Domain:** `${EXTERNAL_DOMAIN}`, `${PUBLIC_DOMAIN}`, `${PUBLIC_URL}`, `${EXTERNAL_API_URL}`, `${INTERNAL_API_URL}`, `${INTERNAL_ADMIN_URL}`, `${CORS_ORIGINS}`, `${INGRESS_DOMAIN}`, `${BASE_DOMAIN}`, `${DEFAULT_BACKEND}`, `${STORAGE_CLASS}`
**Gitea:** `${GITEA_ROOT_URL}`, `${GITEA_GIT_SSH}`, `GITEA_ACCESS_TOKEN` (populated post-bootstrap)
**Gateway:** `${ANI_CODE_GATEWAY_URL}`, `${ANI_CODE_GATEWAY_API_KEY}`, `${ANI_CODE_RELEASE_URL}`
**Email:** `${SMTP_HOST}`, `${SMTP_PORT}`, `${SMTP_USER}`, `${SMTP_PASS}`, `${EMAIL_FROM}`
**SSO:** `${ENTRA_ENABLED}`, `${ENTRA_CLIENT_ID}`, `${ENTRA_CLIENT_SECRET}`, `${ENTRA_TENANT_ID}`, `${ENTRA_REDIRECT_URI}`
**Billing:** `${STRIPE_SECRET_KEY}`, `${STRIPE_PUBLISHABLE_KEY}`

Reference: `launchpad/.env` lines 1-420. Copy the file structure and replace values with placeholders.

- [ ] **Step 2: Commit**

```bash
git add deploy/templates/env.template
git commit -m "feat: add env.template for Launchpad .env generation"
```

---

### Task 9: Create env.gateway.template — Gateway .env

**Files:**
- Create: `deploy/templates/env.gateway.template`

- [ ] **Step 1: Write env.gateway.template**

Based on `gateway/.env`. Include ALL variables from that reference file. Key variables: `ALLOWED_API_KEYS=${ANI_CODE_GATEWAY_API_KEY}`, `DATABASE_URL=${GATEWAY_DATABASE_URL}`, `GATEWAY_PUBLIC_URL=${GATEWAY_PUBLIC_URL}`, `LAUNCHPAD_API_URL=http://api:6802`, `LAUNCHPAD_INTERNAL_SECRET=${LAUNCHPAD_INTERNAL_SECRET}`, `TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}`, `API_KEY_HEADER`, `PORT` (6555), and any other variables present in `gateway/.env`.

- [ ] **Step 2: Commit**

```bash
git add deploy/templates/env.gateway.template
git commit -m "feat: add env.gateway.template for Gateway .env generation"
```

---

### Task 10: Create settings.yml.template

**Files:**
- Create: `deploy/templates/settings.yml.template`

- [ ] **Step 1: Write settings.yml.template**

Based on `launchpad/config/settings.yml`. Most values stay as-is (safe defaults). Parameterize: `allowedDomains` → include `${DOMAIN}`, `operationsReport.adminEmails` → `${ADMIN_EMAIL}`, disable enforceOnly for Microsoft SSO, set notices to disabled.

- [ ] **Step 2: Commit**

```bash
git add deploy/templates/settings.yml.template
git commit -m "feat: add settings.yml.template with safe defaults"
```

---

### Task 11: Create nginx.conf.template

**Files:**
- Create: `deploy/templates/nginx.conf.template`

- [ ] **Step 1: Write nginx.conf.template**

Merge `launchpad/pd/deploy/nginx.conf` (inner layer) and `launchpad/pd/deploy/nginx.txt` (outer layer) into a single unified config with TLS. Four server blocks as per spec:

1. `${LAUNCHPAD_DOMAIN}:443` — all API/UI/Admin/Gateway/WS/SSE routes
2. `${GITEA_DOMAIN}:443` — Gitea proxy
3. `*.${DOMAIN}:443` — Router/project proxy
4. Port 80 catch-all → HTTPS redirect

SSL cert paths: `/etc/nginx/certs/`. Upstream definitions: `api:6802`, `api:6804` (admin), `ui:6801`, `router:6580`, `gateway:6555`, `gitea:3000`.

Reference: `launchpad/pd/deploy/nginx.txt` for outer layer routes, `launchpad/pd/deploy/nginx.conf` for inner layer config.

- [ ] **Step 2: Verify template structure and placeholders**

```bash
# Check that all 4 server blocks exist
grep -c 'server_name' deploy/templates/nginx.conf.template
# Check that key placeholders are present
grep -q '${LAUNCHPAD_DOMAIN}' deploy/templates/nginx.conf.template && echo "LAUNCHPAD_DOMAIN OK"
grep -q '${GITEA_DOMAIN}' deploy/templates/nginx.conf.template && echo "GITEA_DOMAIN OK"
grep -q '${DOMAIN}' deploy/templates/nginx.conf.template && echo "DOMAIN OK"
```

Expected: count of 4 server_name directives, all placeholder checks OK

- [ ] **Step 3: Commit**

```bash
git add deploy/templates/nginx.conf.template
git commit -m "feat: add nginx.conf.template merging two-layer config into single TLS proxy"
```

---

### Task 12: Create render.sh — template rendering

**Files:**
- Create: `deploy/scripts/lib/render.sh`

- [ ] **Step 1: Write render.sh**

Implements `render_templates()` and `render_compose()`. Uses `envsubst` for `.env` and nginx templates. Uses Bash heredoc with conditionals for docker-compose (to handle builtin/external DB and K8s modes).

```bash
#!/bin/bash
# Template rendering

render_templates() {
    log_info "Rendering configuration templates..."

    # Create output directories
    mkdir -p "${DEPLOY_DIR}/generated/launchpad/config"
    mkdir -p "${DEPLOY_DIR}/generated/gateway"
    mkdir -p "${DEPLOY_DIR}/generated/nginx/certs"

    # Compute derived variables
    export EXTERNAL_DOMAIN="https://${LAUNCHPAD_DOMAIN}"
    export PUBLIC_DOMAIN="${LAUNCHPAD_DOMAIN}"
    export PUBLIC_URL="https://${LAUNCHPAD_DOMAIN}"
    export EXTERNAL_API_URL="https://${LAUNCHPAD_DOMAIN}/api"
    export INTERNAL_API_URL="https://${LAUNCHPAD_DOMAIN}/api"
    export INTERNAL_ADMIN_URL="https://${LAUNCHPAD_DOMAIN}/admin/adminapi"
    export CORS_ORIGINS="https://${LAUNCHPAD_DOMAIN}"
    export INGRESS_DOMAIN="${DOMAIN}"
    export BASE_DOMAIN="${DOMAIN}"
    export ANI_CODE_GATEWAY_URL="https://${LAUNCHPAD_DOMAIN}/gatewayproxy"
    export ANI_CODE_RELEASE_URL="https://${GITEA_DOMAIN}/launchpad/ani-code/archive/main.tar.gz"
    export GITEA_ROOT_URL="https://${GITEA_DOMAIN}"
    export GITEA_GIT_SSH="git@${GITEA_DOMAIN}:2222"
    export ENTRA_REDIRECT_URI="https://${LAUNCHPAD_DOMAIN}/api/v1/auth/microsoft/callback"
    export GATEWAY_PUBLIC_URL="https://${LAUNCHPAD_DOMAIN}/gatewayproxy"

    # Resolve image versions
    export IMAGE_VERSION_API="${IMAGE_VERSION_API:-$IMAGE_VERSION}"
    export IMAGE_VERSION_UI="${IMAGE_VERSION_UI:-$IMAGE_VERSION}"
    export IMAGE_VERSION_ROUTER="${IMAGE_VERSION_ROUTER:-$IMAGE_VERSION}"
    export IMAGE_VERSION_GATEWAY="${IMAGE_VERSION_GATEWAY:-$IMAGE_VERSION}"
    export IMAGE_VERSION_GITEA="${IMAGE_VERSION_GITEA:-$IMAGE_VERSION}"

    # Build database URLs
    if [[ "$DB_MODE" == "builtin" ]]; then
        export DATABASE_URL="postgresql://launchpad_mainuser:${DB_PASSWORD_MAIN}@postgres:5432/launchpad?schema=launchpad_main"
        export MONITORING_DATABASE_URL="postgresql://launchpad_monitoringuser:${DB_PASSWORD_MONITORING}@postgres:5432/launchpad?schema=launchpad_monitoring"
        export EVENTS_DATABASE_URL="postgresql://launchpad_eventsuser:${DB_PASSWORD_EVENTS}@postgres:5432/launchpad?schema=launchpad_events"
        export BILLING_DATABASE_URL="postgresql://launchpad_billinguser:${DB_PASSWORD_BILLING}@postgres:5432/launchpad?schema=launchpad_billing"
        export STATS_DATABASE_URL="postgresql://launchpad_statsuser:${DB_PASSWORD_STATS}@postgres:5432/launchpad?schema=launchpad_stats"
        export GATEWAY_DATABASE_URL="postgresql://launchpad_gatewayuser:${DB_PASSWORD_GATEWAY}@postgres:5432/launchpad?schema=launchpad_gateway"
    fi
    # External mode: user provides DATABASE_URL etc. directly

    # Export all variables for envsubst
    export DOMAIN SUBDOMAIN LAUNCHPAD_DOMAIN GITEA_DOMAIN
    export JWT_SECRET JWT_REFRESH_SECRET SESSION_SECRET
    export ENCRYPTION_KEY CLAUDE_CREDENTIALS_ENCRYPTION_KEY SSH_KEY_ENCRYPTION_SECRET
    export ZT_PRIVATE_KEY ZT_PUBLIC_KEY
    export DB_HOST DB_PORT REDIS_HOST REDIS_PORT REDIS_PASSWORD
    export DEFAULT_BACKEND STORAGE_CLASS
    export ANI_CODE_GATEWAY_API_KEY LAUNCHPAD_INTERNAL_SECRET
    export ADMIN_EMAIL IMAGE_REGISTRY
    export IMAGE_VERSION_API IMAGE_VERSION_UI IMAGE_VERSION_ROUTER IMAGE_VERSION_GATEWAY IMAGE_VERSION_GITEA

    # Render .env files
    envsubst < "${DEPLOY_DIR}/templates/env.template" > "${DEPLOY_DIR}/generated/launchpad/.env"
    envsubst < "${DEPLOY_DIR}/templates/env.gateway.template" > "${DEPLOY_DIR}/generated/gateway/.env"
    envsubst < "${DEPLOY_DIR}/templates/nginx.conf.template" > "${DEPLOY_DIR}/generated/nginx/nginx.conf"
    envsubst < "${DEPLOY_DIR}/templates/settings.yml.template" > "${DEPLOY_DIR}/generated/launchpad/config/settings.yml"

    # Render docker-compose.yml (conditional logic)
    render_compose

    log_ok "Templates rendered to generated/"
}

render_compose() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    # Generate docker-compose.yml with Bash heredoc
    # Conditionally include postgres/redis services based on DB_MODE
    # Full implementation writes the complete YAML
    cat > "$compose_file" <<COMPOSE_EOF
# Auto-generated by AniLaunchpad setup — do not edit manually
# Re-generate with: ./deploy/setup.sh --reconfigure

services:
COMPOSE_EOF

    # Conditionally add infrastructure services
    if [[ "$DB_MODE" == "builtin" ]]; then
        cat >> "$compose_file" <<'INFRA_EOF'
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: __POSTGRES_SUPERUSER_PASSWORD__
      POSTGRES_DB: launchpad
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d launchpad"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: >
      sh -c "redis-server
      --appendonly yes
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
      --requirepass __REDIS_PASSWORD__"
    volumes:
      - redis-data:/data
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a __REDIS_PASSWORD__ ping"]
      interval: 10s
      timeout: 5s
      retries: 3

INFRA_EOF
    fi

    # Add core services (always present)
    # Use sed to replace __PLACEHOLDER__ tokens with actual variable values
    # (single-quoted heredoc prevents shell expansion; we replace tokens after)
    cat >> "$compose_file" <<'SERVICES_EOF'
  api:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    restart: unless-stopped
    env_file: ./launchpad/.env
    volumes:
      - ./launchpad/config/settings.yml:/app/config/settings.yml:ro
      - api-logs:/app/logs
      - api-data:/app/data
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD", "node", "scripts/healthcheck.js"]
      interval: 15s
      timeout: 10s
      retries: 5
      start_period: 30s

  ui:
    image: __IMAGE_REGISTRY__/launchpad-ui:__IMAGE_VERSION_UI__
    restart: unless-stopped
    env_file: ./launchpad/.env
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "wget --spider -q http://localhost:6801 || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3

  router:
    image: __IMAGE_REGISTRY__/launchpad-router:__IMAGE_VERSION_ROUTER__
    restart: unless-stopped
    env_file: ./launchpad/.env
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "wget --spider http://localhost:6580/_health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3

  cron:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    restart: unless-stopped
    env_file: ./launchpad/.env
    command: ["node", "scripts/cron.js"]
    networks:
      - launchpad-network

  backup-worker:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    restart: unless-stopped
    env_file: ./launchpad/.env
    command: ["node", "scripts/backup-worker.js"]
    networks:
      - launchpad-network

  gateway:
    image: __IMAGE_REGISTRY__/ani-code-gateway:__IMAGE_VERSION_GATEWAY__
    restart: unless-stopped
    env_file: ./gateway/.env
    volumes:
      - gateway-data:/app/data
      - gateway-logs:/app/logs
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "wget --spider -q http://localhost:6555/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3

  gitea:
    image: gitea/gitea:__IMAGE_VERSION_GITEA__
    restart: unless-stopped
    environment:
      - GITEA__database__DB_TYPE=postgres
      - GITEA__database__HOST=__DB_HOST__:__DB_PORT__
      - GITEA__database__NAME=gitea
      - GITEA__database__USER=gitea
      - GITEA__database__PASSWD=__GITEA_DB_PASSWORD__
      - GITEA__server__ROOT_URL=__GITEA_ROOT_URL__
      - GITEA__server__SSH_PORT=2222
      - GITEA__server__SSH_LISTEN_PORT=22
    volumes:
      - gitea-data:/data
    ports:
      - "2222:22"
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "wget --spider -q http://localhost:3000/api/v1/version || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 30s

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/certs:/etc/nginx/certs:ro
    networks:
      - launchpad-network
    depends_on:
      - api
      - ui
      - router
      - gateway
      - gitea
    healthcheck:
      test: ["CMD-SHELL", "nginx -t && curl -sf http://localhost:80/ || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3

SERVICES_EOF

    # Replace __PLACEHOLDER__ tokens with actual variable values
    sed -i \
        -e "s|__IMAGE_REGISTRY__|${IMAGE_REGISTRY}|g" \
        -e "s|__IMAGE_VERSION_API__|${IMAGE_VERSION_API}|g" \
        -e "s|__IMAGE_VERSION_UI__|${IMAGE_VERSION_UI}|g" \
        -e "s|__IMAGE_VERSION_ROUTER__|${IMAGE_VERSION_ROUTER}|g" \
        -e "s|__IMAGE_VERSION_GATEWAY__|${IMAGE_VERSION_GATEWAY}|g" \
        -e "s|__IMAGE_VERSION_GITEA__|${IMAGE_VERSION_GITEA}|g" \
        -e "s|__DB_HOST__|${DB_HOST}|g" \
        -e "s|__DB_PORT__|${DB_PORT}|g" \
        -e "s|__GITEA_DB_PASSWORD__|${GITEA_DB_PASSWORD}|g" \
        -e "s|__GITEA_ROOT_URL__|${GITEA_ROOT_URL}|g" \
        -e "s|__REDIS_PASSWORD__|${REDIS_PASSWORD}|g" \
        -e "s|__POSTGRES_SUPERUSER_PASSWORD__|${POSTGRES_SUPERUSER_PASSWORD}|g" \
        "$compose_file"

    # Networks and volumes
    cat >> "$compose_file" <<'NET_EOF'

networks:
  launchpad-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
  api-logs:
  api-data:
  gateway-data:
  gateway-logs:
  gitea-data:
NET_EOF

}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/render.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/render.sh
git commit -m "feat: add render.sh template rendering with conditional compose generation"
```

---

## Chunk 4: Infrastructure modules (certs.sh, k3s.sh, database.sh)

### Task 13: Create certs.sh — SSL certificate management

**Files:**
- Create: `deploy/scripts/lib/certs.sh`

- [ ] **Step 1: Write certs.sh**

Implements `setup_certificates()`. Two paths:
- **letsencrypt**: Install `acme.sh` if not present, issue certs using DNS-01 (cloudflare/aliyun/azure/manual), install with `--reloadcmd` for Nginx reload, auto-renewal cron.
- **custom**: Copy user-provided cert/key files to `generated/nginx/certs/`.

Certificates needed:
1. `${LAUNCHPAD_DOMAIN}` + `${GITEA_DOMAIN}` — can be one SAN cert or use wildcard
2. `*.${DOMAIN}` — wildcard for projects

```bash
#!/bin/bash
# SSL certificate management

setup_certificates() {
    log_info "Setting up SSL certificates..."

    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"
    mkdir -p "$cert_dir"

    if [[ "$SSL_MODE" == "letsencrypt" ]]; then
        setup_letsencrypt
    else
        setup_custom_certs
    fi

    log_ok "SSL certificates configured"
}

setup_letsencrypt() {
    # Install acme.sh if not present
    if ! command -v acme.sh &>/dev/null && [[ ! -f ~/.acme.sh/acme.sh ]]; then
        log_info "Installing acme.sh..."
        curl -fsSL https://get.acme.sh | sh -s email="${ADMIN_EMAIL}" 2>/dev/null
    fi
    local ACME="${HOME}/.acme.sh/acme.sh"

    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local reload_cmd="docker compose -f ${compose_file} exec nginx nginx -s reload"

    # Set DNS API credentials based on provider
    case "$DNS_PROVIDER" in
        cloudflare)
            export CF_Token="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_cf"
            ;;
        aliyun)
            export Ali_Key="${DNS_API_KEY:-}"
            export Ali_Secret="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_ali"
            ;;
        azure)
            export AZUREDNS_SUBSCRIPTIONID="${DNS_API_TOKEN}"
            local dns_flag="--dns dns_azure"
            ;;
        manual)
            local dns_flag="--dns --yes-I-know-dns-manual-mode-enough-go-ahead-please"
            ;;
    esac

    # Issue wildcard cert (covers *.domain and the domain itself)
    log_info "Issuing wildcard certificate for *.${DOMAIN}..."
    local issue_rc=0
    $ACME --issue $dns_flag \
        -d "*.${DOMAIN}" \
        -d "${DOMAIN}" \
        -d "${LAUNCHPAD_DOMAIN}" \
        -d "${GITEA_DOMAIN}" \
        --keylength ec-256 2>/dev/null || issue_rc=$?

    # Exit code 2 = cert already issued/renewed (skip), 0 = success, other = error
    if [[ "$issue_rc" -ne 0 && "$issue_rc" -ne 2 ]]; then
        log_error "Certificate issuance failed (exit code: $issue_rc). Check DNS provider credentials."
        return 1
    fi

    # Install cert — use consistent naming for both LE and custom paths
    $ACME --install-cert -d "*.${DOMAIN}" \
        --key-file "${cert_dir}/privkey.pem" \
        --fullchain-file "${cert_dir}/fullchain.pem" \
        --reloadcmd "$reload_cmd" 2>/dev/null

    # Create symlinks for wildcard paths (consistent with custom cert naming)
    ln -sf "${cert_dir}/fullchain.pem" "${cert_dir}/wildcard-fullchain.pem"
    ln -sf "${cert_dir}/privkey.pem" "${cert_dir}/wildcard-privkey.pem"
}

setup_custom_certs() {
    local cert_dir="${DEPLOY_DIR}/generated/nginx/certs"

    # Copy main cert (for launchpad + gitea domains)
    if [[ -n "${SSL_CERT_PATH:-}" && -f "$SSL_CERT_PATH" ]]; then
        cp "$SSL_CERT_PATH" "${cert_dir}/fullchain.pem"
        cp "$SSL_KEY_PATH" "${cert_dir}/privkey.pem"
    fi

    # Copy wildcard cert (for *.domain)
    if [[ -n "${SSL_WILDCARD_CERT_PATH:-}" && -f "$SSL_WILDCARD_CERT_PATH" ]]; then
        cp "$SSL_WILDCARD_CERT_PATH" "${cert_dir}/wildcard-fullchain.pem"
        cp "$SSL_WILDCARD_KEY_PATH" "${cert_dir}/wildcard-privkey.pem"
    else
        # Use same cert for wildcard if not provided separately
        [[ -f "${cert_dir}/fullchain.pem" ]] && cp "${cert_dir}/fullchain.pem" "${cert_dir}/wildcard-fullchain.pem"
        [[ -f "${cert_dir}/privkey.pem" ]] && cp "${cert_dir}/privkey.pem" "${cert_dir}/wildcard-privkey.pem"
    fi
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/certs.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/certs.sh
git commit -m "feat: add certs.sh SSL certificate management (Let's Encrypt + custom)"
```

---

### Task 14: Create k3s.sh — Kubernetes setup

**Files:**
- Create: `deploy/scripts/lib/k3s.sh`

- [ ] **Step 1: Write k3s.sh**

Implements `setup_kubernetes()`. Two modes:
- **builtin**: Install k3s with `--disable traefik`, `--tls-san <IP>`, wait for ready, export kubeconfig.
- **external**: Validate kubeconfig connectivity, skip k3s install.

`register_cluster()`: Download `register.sh` from deployed Launchpad API, execute with appropriate flags.

```bash
#!/bin/bash
# Kubernetes (k3s) setup and cluster registration

setup_kubernetes() {
    if [[ "$K8S_MODE" == "builtin" ]]; then
        install_k3s
    else
        validate_external_k8s
    fi
}

install_k3s() {
    # Skip if already installed
    if command -v k3s &>/dev/null && systemctl is-active --quiet k3s 2>/dev/null; then
        log_ok "k3s already installed and running"
        export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
        return 0
    fi

    log_info "Installing k3s..."

    local internal_ip
    internal_ip=$(detect_internal_ip) || internal_ip="127.0.0.1"

    curl -sfL https://get.k3s.io | sh -s - \
        --disable traefik \
        --tls-san "$internal_ip" \
        --write-kubeconfig-mode 644

    # Wait for k3s to be ready
    log_info "Waiting for k3s to be ready..."
    local timeout=60 elapsed=0
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    while [[ $elapsed -lt $timeout ]]; do
        if k3s kubectl get nodes &>/dev/null; then
            log_ok "k3s is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    log_error "k3s failed to start within ${timeout}s"
    return 1
}

validate_external_k8s() {
    log_info "Validating external K8s cluster..."

    local kubeconfig="${K8S_KUBECONFIG_PATH}"
    if [[ ! -f "$kubeconfig" ]]; then
        log_error "$MSG_ERR_FILE_NOT_FOUND: $kubeconfig"
        exit 1
    fi

    local kubectl_args="--kubeconfig=$kubeconfig"
    [[ -n "${K8S_CONTEXT:-}" ]] && kubectl_args="$kubectl_args --context=$K8S_CONTEXT"

    if kubectl $kubectl_args cluster-info --request-timeout=10s &>/dev/null; then
        local nodes
        nodes=$(kubectl $kubectl_args get nodes --no-headers 2>/dev/null | wc -l)
        local version
        version=$(kubectl $kubectl_args version -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | head -1 | cut -d'"' -f4)
        log_ok "$MSG_K8S_VERIFIED: $nodes nodes, $version"
    else
        log_error "Cannot connect to K8s cluster with provided kubeconfig"
        exit 1
    fi
}

register_cluster() {
    log_info "$MSG_DEPLOY_REGISTER"

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local register_script="/tmp/launchpad-register.sh"

    # Try downloading from public URL first, fallback to Docker network
    if curl -fsSL "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh" \
        -o "$register_script" 2>/dev/null; then
        log_ok "Downloaded register.sh from public URL"
    else
        log_warn "Public URL not accessible, using Docker network..."
        docker compose -f "$compose_file" exec -T api \
            curl -fsSL "http://localhost:6802/admin/adminapi/k3s/register.sh" \
            > "$register_script"
    fi
    chmod +x "$register_script"

    # register.sh runs on the host, so use localhost (API port 6802 is exposed via Nginx on 443,
    # but we can also use docker port mapping). Use the public HTTPS URL for registration.
    local reg_url="https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register"

    if [[ "$K8S_MODE" == "builtin" ]]; then
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        CLUSTER_NAME="local-k3s" \
            "$register_script" || log_warn "Cluster registration returned non-zero (may need manual completion)"
    else
        LAUNCHPAD_REGISTRATION_URL="$reg_url" \
        KUBECONFIG="$K8S_KUBECONFIG_PATH" \
        CLUSTER_NAME="${K8S_CLUSTER_NAME:-external-k8s}" \
            "$register_script" --skip-k3s-check || log_warn "Cluster registration returned non-zero"
    fi

    rm -f "$register_script"
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/k3s.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/k3s.sh
git commit -m "feat: add k3s.sh Kubernetes setup and cluster registration"
```

---

### Task 15: Create database.sh — PostgreSQL initialization

**Files:**
- Create: `deploy/scripts/lib/database.sh`

- [ ] **Step 1: Write database.sh**

Implements `init_database()`. Only runs in builtin mode. Creates 6 schemas (main, monitoring, events, billing, stats, gateway) + Gitea DB. Each schema gets its own user with limited permissions.

Uses `docker compose exec postgres psql` to run SQL commands against the running PostgreSQL container.

```bash
#!/bin/bash
# Database initialization

init_database() {
    if [[ "$DB_MODE" != "builtin" ]]; then
        log_info "External database mode — skipping initialization"
        return 0
    fi

    log_info "$MSG_DEPLOY_DB_INIT"

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local psql_base="docker compose -f $compose_file exec -T postgres psql -U postgres"

    # Create the launchpad database if it doesn't exist
    $psql_base -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'launchpad'" | grep -q 1 \
        || $psql_base -d postgres -c "CREATE DATABASE launchpad"

    local psql_cmd="$psql_base -d launchpad"

    # Check if already initialized
    local schema_count
    schema_count=$($psql_cmd -t -c "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'launchpad_%'" 2>/dev/null | tr -d ' ')
    if [[ "${schema_count:-0}" -ge 6 ]]; then
        log_ok "Database schemas already exist — skipping"
        return 0
    fi

    # Create schemas and users
    local schemas=("main" "monitoring" "events" "billing" "stats" "gateway")
    local passwords=(
        "$DB_PASSWORD_MAIN" "$DB_PASSWORD_MONITORING" "$DB_PASSWORD_EVENTS"
        "$DB_PASSWORD_BILLING" "$DB_PASSWORD_STATS" "$DB_PASSWORD_GATEWAY"
    )

    for i in "${!schemas[@]}"; do
        local schema="launchpad_${schemas[$i]}"
        local user="launchpad_${schemas[$i]}user"
        local pass="${passwords[$i]}"

        $psql_cmd <<SQL
CREATE SCHEMA IF NOT EXISTS ${schema};
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${user}') THEN
        CREATE ROLE ${user} LOGIN PASSWORD '${pass}';
    END IF;
END \$\$;
GRANT USAGE ON SCHEMA ${schema} TO ${user};
GRANT CREATE ON SCHEMA ${schema} TO ${user};
ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT ALL ON TABLES TO ${user};
ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT ALL ON SEQUENCES TO ${user};
SQL
        log_ok "Schema: $schema"
    done

    # Create Gitea database
    docker compose -f "$compose_file" exec -T postgres psql -U postgres <<SQL
SELECT 'CREATE DATABASE gitea' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gitea')\gexec
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'gitea') THEN
        CREATE ROLE gitea LOGIN PASSWORD '${GITEA_DB_PASSWORD}';
    END IF;
END \$\$;
GRANT ALL PRIVILEGES ON DATABASE gitea TO gitea;
SQL
    log_ok "Gitea database"

    log_ok "Database initialization complete"
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/database.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/database.sh
git commit -m "feat: add database.sh PostgreSQL schema and user initialization"
```

---

## Chunk 5: deploy.sh (deployment orchestration, health checks, Gitea bootstrap, output)

### Task 16: Create deploy.sh — deployment orchestration

**Files:**
- Create: `deploy/scripts/lib/deploy.sh`

- [ ] **Step 1: Write deploy.sh**

Implements: `deploy_services()`, `wait_for_healthy()`, `bootstrap_gitea()`, `show_result()`, `show_status()`, `upgrade_services()`, `uninstall_services()`, `resume_deploy()`.

```bash
#!/bin/bash
# Deployment orchestration, health checks, and lifecycle management

COMPOSE_CMD="docker compose -f ${DEPLOY_DIR}/generated/docker-compose.yml"

wait_for_healthy() {
    local service="$1" timeout="$2"
    local elapsed=0

    while [[ $elapsed -lt $timeout ]]; do
        local container_id
        container_id=$($COMPOSE_CMD ps -q "$service" 2>/dev/null | head -1)
        if [[ -n "$container_id" ]]; then
            local status
            status=$(docker inspect --format='{{.State.Health.Status}}' "$container_id" 2>/dev/null || echo "unknown")
            if [[ "$status" == "healthy" ]]; then
                log_ok "$service $MSG_HEALTH_HEALTHY"
                return 0
            fi
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        printf "\r  $MSG_HEALTH_WAITING $service... ${elapsed}s/${timeout}s"
    done
    echo ""
    log_error "$service $MSG_HEALTH_FAILED in ${timeout}s"
    $COMPOSE_CMD logs --tail=20 "$service"
    return 1
}

deploy_fail() {
    local phase="$1"
    log_error "$MSG_DEPLOY_FAILED at phase: $phase"
    echo ""
    echo "  $MSG_ERR_RESUME:"
    echo "    ./deploy/setup.sh --resume"
    echo ""
    echo "  $MSG_ERR_LOGS:"
    echo "    docker compose -f deploy/generated/docker-compose.yml logs <service>"
    echo ""
    echo "  $MSG_ERR_UNINSTALL:"
    echo "    ./deploy/setup.sh --uninstall"
    exit 1
}

deploy_services() {
    log_info "$MSG_DEPLOY_STARTING"

    # Phase 1: Infrastructure
    if [[ "$DB_MODE" == "builtin" ]]; then
        log_step "1/4" "$MSG_DEPLOY_INFRA"
        $COMPOSE_CMD up -d postgres redis
        wait_for_healthy postgres 30 || deploy_fail "infrastructure (postgres)"
        wait_for_healthy redis 15 || deploy_fail "infrastructure (redis)"
    fi

    # Phase 2: Database init
    init_database || deploy_fail "database initialization"

    # Phase 3: Application services
    log_step "2/4" "$MSG_DEPLOY_SERVICES"
    $COMPOSE_CMD up -d api ui router cron backup-worker gateway gitea
    wait_for_healthy api 60 || deploy_fail "application services (api)"
    wait_for_healthy ui 30 || deploy_fail "application services (ui)"
    wait_for_healthy router 30 || deploy_fail "application services (router)"
    wait_for_healthy gitea 45 || deploy_fail "application services (gitea)"

    # Phase 4: Nginx
    log_step "3/4" "$MSG_DEPLOY_NGINX"
    $COMPOSE_CMD up -d nginx
    wait_for_healthy nginx 15 || deploy_fail "nginx"

    log_ok "$MSG_DEPLOY_SERVICES complete"
}

bootstrap_gitea() {
    log_info "$MSG_DEPLOY_GITEA"

    local gitea_url="http://localhost:3000"
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"

    # Check if already bootstrapped
    local status
    status=$(docker compose -f "$compose_file" exec -T gitea \
        curl -s -o /dev/null -w "%{http_code}" "http://localhost:3000/api/v1/orgs/launchpad" 2>/dev/null || echo "000")

    if [[ "$status" == "200" ]]; then
        log_ok "Gitea already bootstrapped — skipping"
        return 0
    fi

    # Wait for Gitea to be fully ready
    sleep 5

    local gitea_admin="${ADMIN_EMAIL%%@*}"  # username from email
    local gitea_pass="${ADMIN_PASSWORD}"

    # Create admin user via Gitea CLI
    docker compose -f "$compose_file" exec -T gitea \
        gitea admin user create \
        --username "$gitea_admin" \
        --password "$gitea_pass" \
        --email "$ADMIN_EMAIL" \
        --admin --must-change-password=false 2>/dev/null || true

    # Generate API token
    local token_response
    token_response=$(docker compose -f "$compose_file" exec -T gitea \
        curl -s -X POST "http://localhost:3000/api/v1/users/${gitea_admin}/tokens" \
        -u "${gitea_admin}:${gitea_pass}" \
        -H "Content-Type: application/json" \
        -d '{"name":"launchpad-api","scopes":["all"]}')

    # Gitea returns token in "sha1" (older) or "token" (newer) field
    GITEA_ACCESS_TOKEN=$(echo "$token_response" | grep -oE '"(sha1|token)":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [[ -n "$GITEA_ACCESS_TOKEN" ]]; then
        # Create organization
        docker compose -f "$compose_file" exec -T gitea \
            curl -s -X POST "http://localhost:3000/api/v1/orgs" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            -H "Content-Type: application/json" \
            -d '{"username":"launchpad","full_name":"Launchpad","visibility":"public"}' >/dev/null

        # Create ani-code repo
        docker compose -f "$compose_file" exec -T gitea \
            curl -s -X POST "http://localhost:3000/api/v1/orgs/launchpad/repos" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            -H "Content-Type: application/json" \
            -d '{"name":"ani-code","auto_init":true,"default_branch":"main"}' >/dev/null

        # Write token back to .env
        echo "" >> "${DEPLOY_DIR}/generated/launchpad/.env"
        echo "GITEA_ACCESS_TOKEN=${GITEA_ACCESS_TOKEN}" >> "${DEPLOY_DIR}/generated/launchpad/.env"
        echo "GITEA_USER=${gitea_admin}" >> "${DEPLOY_DIR}/generated/launchpad/.env"

        # Restart API to pick up new token
        $COMPOSE_CMD restart api
        wait_for_healthy api 60

        log_ok "Gitea bootstrapped (admin: $gitea_admin, org: launchpad, repo: ani-code)"
    else
        log_warn "Gitea token generation failed — manual setup may be required"
    fi
}

show_result() {
    local ip
    ip=$(detect_internal_ip 2>/dev/null || echo "x.x.x.x")

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  ✓ $MSG_DEPLOY_COMPLETE${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
    echo ""
    echo "  $MSG_OUT_URLS:"
    echo "    $MSG_OUT_DASHBOARD:  https://${LAUNCHPAD_DOMAIN}"
    echo "    $MSG_OUT_GITEA:      https://${GITEA_DOMAIN}"
    echo "    $MSG_OUT_ADMIN:      https://${LAUNCHPAD_DOMAIN}/admin"
    echo ""
    echo "  $MSG_OUT_CREDENTIALS:"
    echo "    $MSG_OUT_EMAIL:      ${ADMIN_EMAIL}"
    echo "    $MSG_OUT_PASSWORD:   ${ADMIN_PASSWORD}"
    echo ""
    echo "  $MSG_OUT_K8S:"
    echo "    $MSG_OUT_MODE:       ${K8S_MODE}"
    echo "    $MSG_OUT_STATUS:     ✓ Registered"
    echo ""
    echo "  $MSG_OUT_DNS:"
    echo "    $MSG_OUT_DNS_MSG ($ip):"
    echo "    ├─ ${LAUNCHPAD_DOMAIN}  → $ip"
    echo "    ├─ ${GITEA_DOMAIN}      → $ip"
    echo "    └─ *.${DOMAIN}          → $ip"
    echo ""
    echo "  $MSG_OUT_CONFIG: deploy/generated/"
    echo "  $MSG_OUT_LOGS: docker compose -f deploy/generated/docker-compose.yml logs -f"
    echo "  $MSG_OUT_RECONFIG: ./deploy/setup.sh --reconfigure"
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
}

show_status() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        echo "No deployment found. Run ./deploy/setup.sh to deploy."
        exit 1
    fi
    docker compose -f "$compose_file" ps
}

upgrade_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    # Reload versions and saved config, then re-render compose to pick up new image tags
    source "${DEPLOY_DIR}/versions.conf"
    source "${DEPLOY_DIR}/generated/.setup.conf"
    source "${DEPLOY_DIR}/scripts/lib/render.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    render_compose
    log_info "Pulling latest images..."
    docker compose -f "$compose_file" pull
    docker compose -f "$compose_file" up -d
    wait_for_healthy api 60
    wait_for_healthy ui 30
    wait_for_healthy nginx 15
    log_ok "Services upgraded"
}

uninstall_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    log_warn "This will stop all services, remove containers, AND DELETE ALL DATA (databases, Gitea repos, etc.)."
    echo ""
    if ask_confirm "Are you sure you want to delete everything? This cannot be undone." "N"; then
        docker compose -f "$compose_file" down -v
        log_ok "Services and volumes removed"
    fi
}

resume_deploy() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_error "No deployment configuration found. Run ./deploy/setup.sh first."
        exit 1
    fi

    # Load saved config and all modules
    source "${DEPLOY_DIR}/generated/.setup.conf"
    source "${DEPLOY_DIR}/scripts/lib/common.sh"
    source "${DEPLOY_DIR}/scripts/lib/database.sh"
    source "${DEPLOY_DIR}/scripts/lib/k3s.sh"

    log_info "Checking deployment state and resuming from failed phase..."

    # Helper to check if a service is already healthy
    is_healthy() {
        local cid
        cid=$(docker compose -f "$compose_file" ps -q "$1" 2>/dev/null | head -1)
        [[ -n "$cid" ]] && [[ "$(docker inspect --format='{{.State.Health.Status}}' "$cid" 2>/dev/null)" == "healthy" ]]
    }

    # Phase 1: Infrastructure
    if [[ "$DB_MODE" == "builtin" ]]; then
        if ! is_healthy postgres || ! is_healthy redis; then
            log_step "1/4" "$MSG_DEPLOY_INFRA"
            docker compose -f "$compose_file" up -d postgres redis
            wait_for_healthy postgres 30 || deploy_fail "infrastructure (postgres)"
            wait_for_healthy redis 15 || deploy_fail "infrastructure (redis)"
        else
            log_ok "Infrastructure already healthy — skipping"
        fi
    fi

    # Phase 2: Database init (idempotent, safe to re-run)
    init_database

    # Phase 3: Application services
    if ! is_healthy api || ! is_healthy ui || ! is_healthy router; then
        log_step "2/4" "$MSG_DEPLOY_SERVICES"
        docker compose -f "$compose_file" up -d api ui router cron backup-worker gateway gitea
        wait_for_healthy api 60 || deploy_fail "application services (api)"
        wait_for_healthy ui 30 || deploy_fail "application services (ui)"
        wait_for_healthy router 30 || deploy_fail "application services (router)"
        wait_for_healthy gitea 45 || deploy_fail "application services (gitea)"
    else
        log_ok "Application services already healthy — skipping"
    fi

    # Phase 4: Nginx
    if ! is_healthy nginx; then
        log_step "3/4" "$MSG_DEPLOY_NGINX"
        docker compose -f "$compose_file" up -d nginx
        wait_for_healthy nginx 15 || deploy_fail "nginx"
    else
        log_ok "Nginx already healthy — skipping"
    fi

    # Phase 5: Gitea bootstrap (idempotent)
    bootstrap_gitea

    # Phase 6: Cluster registration
    register_cluster

    log_ok "All services healthy"
}
```

- [ ] **Step 2: Verify syntax**

```bash
bash -n deploy/scripts/lib/deploy.sh && echo "Syntax OK"
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/deploy.sh
git commit -m "feat: add deploy.sh orchestration, health checks, Gitea bootstrap, lifecycle"
```

---

### Task 17: Final integration and validation

- [ ] **Step 1: Verify all scripts are syntactically valid**

```bash
for f in deploy/scripts/lib/*.sh deploy/scripts/lang/*.sh deploy/setup.sh; do
    bash -n "$f" && echo "OK: $f" || echo "FAIL: $f"
done
```

Expected: all OK

- [ ] **Step 2: Verify setup.sh sources all modules without error**

```bash
bash -c 'DEPLOY_DIR=deploy; source deploy/scripts/lib/common.sh && echo "common OK"'
bash -c 'DEPLOY_DIR=deploy; source deploy/scripts/lib/common.sh && source deploy/scripts/lib/i18n.sh && echo "i18n OK"'
```

Expected: both OK

- [ ] **Step 3: Verify directory structure matches spec**

```bash
find deploy/ -type f | sort
```

Expected output should match the spec's directory structure (minus generated/).

- [ ] **Step 4: Final commit**

```bash
git add -A deploy/
git commit -m "feat: complete one-click deployment script v1.0

Modular Bash deployment script for AniLaunchpad platform:
- Interactive setup with Chinese/English support
- Built-in or external PostgreSQL/Redis/k3s
- Let's Encrypt or custom SSL certificates
- Unified Nginx TLS proxy (3 server blocks)
- Gitea auto-bootstrap (admin, org, repo, token)
- Cluster registration via Launchpad API
- Config persistence and idempotent re-runs"
```
