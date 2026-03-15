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
MSG_SSL_SELFSIGNED="Self-signed certificate (local testing)"
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
MSG_DB_EXT_HINT="Provide full PostgreSQL connection URLs for each schema (format: postgresql://user:pass@host:port/db?schema=name)"
MSG_DB_URL_MAIN="Main DB URL (launchpad_main schema)"
MSG_DB_URL_MONITORING="Monitoring DB URL (launchpad_monitoring schema)"
MSG_DB_URL_EVENTS="Events DB URL (launchpad_events schema)"
MSG_DB_URL_BILLING="Billing DB URL (launchpad_billing schema)"
MSG_DB_URL_STATS="Stats DB URL (launchpad_stats schema)"
MSG_DB_URL_GATEWAY="Gateway DB URL (launchpad_gateway schema)"
MSG_DB_URL_GITEA="Gitea DB URL (e.g. postgresql://gitea:pass@host:5432/gitea)"
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
MSG_K8S_MACOS_WARN="macOS detected: k3s not available, using external K8s mode (Colima/Docker Desktop/kind)"

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
MSG_ADV_STORAGE="File storage"
MSG_ADV_STORAGE_DESC="Local, S3, or Azure Blob storage"
MSG_STORAGE_PROVIDER="Storage provider"
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
