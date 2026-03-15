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
MSG_SSL_SELFSIGNED="自签名证书（本地测试）"
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
MSG_DB_EXT_HINT="请提供各 schema 的完整 PostgreSQL 连接 URL（格式：postgresql://user:pass@host:port/db?schema=name）"
MSG_DB_URL_MAIN="主库 URL（launchpad_main schema）"
MSG_DB_URL_MONITORING="监控库 URL（launchpad_monitoring schema）"
MSG_DB_URL_EVENTS="事件库 URL（launchpad_events schema）"
MSG_DB_URL_BILLING="计费库 URL（launchpad_billing schema）"
MSG_DB_URL_STATS="统计库 URL（launchpad_stats schema）"
MSG_DB_URL_GATEWAY="网关库 URL（launchpad_gateway schema）"
MSG_DB_URL_GITEA="Gitea 库 URL（例：postgresql://gitea:pass@host:5432/gitea）"
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
MSG_K8S_MACOS_WARN="检测到 macOS：k3s 不可用，已切换为外部 K8s 模式（Colima/Docker Desktop/kind）"

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
MSG_ADV_STORAGE="文件存储"
MSG_ADV_STORAGE_DESC="本地、S3 或 Azure Blob 存储"
MSG_STORAGE_PROVIDER="存储方式"
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
