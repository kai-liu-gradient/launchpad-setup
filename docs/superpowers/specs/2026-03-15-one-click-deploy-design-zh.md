# AniLaunchpad 一键部署脚本设计文档

## 概述

一个模块化的交互式 Bash 部署脚本，用于在单台 Linux 服务器上自动化部署完整的 AniLaunchpad 平台。支持中英双语，提供合理的默认值以实现快速部署，同时支持可选的高级配置满足进阶用户需求。

## 目标用户

主要用户：具备基本 Linux 和 Docker 经验的运维工程师，不需要了解 Launchpad 内部细节。
延伸目标：能按提示操作的非技术人员。

## 架构

### 单机全栈架构

所有服务运行在同一台机器上。Nginx 负责三个域名的 TLS 终止和反向代理：

```
用户浏览器
    │
    ▼
┌─────────────────────────────────────────────────┐
│  Nginx (Docker 容器, :443/:80)                   │
│  ├─ launchpad.{sub}.{domain}  → API/UI/Admin    │
│  ├─ launchpad-gitea.{sub}.{domain} → Gitea      │
│  └─ *.{domain} → Router → k3s/K8s Pods         │
└────────────────┬────────────────────────────────┘
                 │ Docker 网络
    ┌────────────┼────────────────────┐
    ▼            ▼                    ▼
┌────────┐ ┌─────────┐ ┌──────────────────┐
│Launchpad│ │ Gateway │ │   基础设施        │
│ API/UI  │ │  :6555  │ │ PostgreSQL Redis │
│ Router  │ │         │ │ Gitea   k3s      │
│ Cron    │ │         │ │                  │
└────────┘ └─────────┘ └──────────────────┘
```

### 关键设计决策

1. **单层 Nginx** — 在一个 Docker 容器中完成 TLS 终止 + 反向代理，取代之前的两层架构（外部 VM Nginx + 内部 Docker Nginx）。
2. **三个 server block**：
   - `launchpad.{sub}.{domain}:443` — 合并内外层配置（Terminal WS、SSE、Socket.io、Gateway 代理、Admin、UI）
   - `launchpad-gitea.{sub}.{domain}:443` — 代理到 Gitea 容器 `:3000`
   - `*.{domain}:443` — 代理到 Router `:6580`（支持 WebSocket）
3. **HTTP→HTTPS 跳转** — 80 端口统一 301 到 443。
4. **k3s** — 禁用内置 Traefik 避免端口冲突；仅用作项目容器运行时。
5. **简化域名模型** — 现有生产环境使用独立的 `playground.astratech.ae` 作为项目子域名。本脚本简化为单一域名：`*.{domain}` 用于项目，`launchpad.{sub}.{domain}` 用于面板。减少 DNS 和证书复杂度。`BASE_DOMAIN` 和 `INGRESS_DOMAIN` 环境变量均设为 `{domain}`。
6. **Admin API 端口 6804** — API 容器同时暴露 `:6802`（公共 API）和 `:6804`（管理后台）。Nginx 将 `/admin/` 路由到同一 `api` 容器的 `:6804`。

## 交互流程

### 语言选择

```
═══════════════════════════════════════════
  AniLaunchpad 一键部署脚本 v1.0
═══════════════════════════════════════════

  Language / 语言选择:
    1) English
    2) 中文
  Please select / 请选择 [1]:
```

### 第一步：域名配置

```
[1/6] 基本配置
  请输入主域名 (例: company.com):
  子域名前缀 [corp]:

  将生成以下域名:
    ✓ launchpad.corp.company.com      (主站面板)
    ✓ launchpad-gitea.corp.company.com (Gitea 仓库)
    ✓ *.company.com                    (项目访问)
  确认? [Y/n]:
```

### 第二步：SSL 证书

```
[2/6] SSL 证书
  证书方式:
    1) Let's Encrypt 自动申请 (推荐)
    2) 提供证书文件
  请选择 [1]:

  # 如果选 1:
  DNS 提供商:
    1) Cloudflare
    2) 阿里云 DNS
    3) Azure DNS
    4) 其他 (手动 DNS 验证)
  请选择 [1]:
  Cloudflare API Token:
```

- 主站 + Gitea：标准证书或复用泛域名证书
- `*.{domain}` 泛域名：**必须使用 DNS-01 验证**
- 工具：`acme.sh`（零依赖，内置 DNS 提供商插件）

### 第三步：数据库

```
[3/6] 数据库
  数据库模式:
    1) 内置 PostgreSQL + Redis (推荐)
    2) 使用外部数据库
  请选择 [1]:

  # 如果选 1 → 自动生成密码，零配置
  # 如果选 2 → 提示输入连接信息
```

### 第四步：Kubernetes

```
[4/6] Kubernetes 集群
  K8s 模式:
    1) 内置 k3s (推荐，本机自动安装)
    2) 使用已有 K8s 集群
  请选择 [1]:

  # 如果选 2:
  kubeconfig 文件路径 [~/.kube/config]:
  K8s Context 名称 (留空使用当前 context):
  Ingress 后端域名:
    (*.company.com 子域名需解析到此地址)
    例: k8s-ingress.company.com
  Storage Class [standard]:
  ✓ 已验证集群连接: 3 nodes, v1.28.2
```

### 第五步：高级配置

```
[5/6] 高级配置
  是否进入高级配置? [y/N]:

  # 如果选 y，展示分类菜单:
  请选择要配置的模块 (多选用逗号分隔，回车跳过全部):
    1) 管理员账号      - 设置初始管理员邮箱和密码
    2) 邮件服务 (SMTP) - 配置邮件发送
    3) 支付集成        - Stripe 支付密钥
    4) 企业 SSO        - Microsoft Entra ID 单点登录
    5) AI 服务         - CRS2 / PayGO 凭证配置
    6) 通知集成        - Telegram Bot 通知
    7) 性能调优        - API 副本数、连接池、超时等
    8) 全部配置
  请选择 [回车跳过]: 1,2

  ── 管理员账号 ──
  管理员邮箱 [admin@company.com]:
  管理员密码 (留空自动生成):
  ✓ 已配置

  ── 邮件服务 ──
  SMTP 服务器 [smtp.gmail.com]:
  SMTP 端口 [587]:
  SMTP 用户名:
  SMTP 密码:
  发件人地址 [noreply@company.com]:
  ✓ 已配置

  已跳过: 支付集成, 企业SSO, AI服务, 通知集成, 性能调优
  (后续可通过 ./deploy/setup.sh --reconfigure 重新配置)
```

每个高级模块有独立的线性问答流程，均提供默认值。未配置的模块使用安全的默认值（功能禁用但不影响系统运行）。

### 第六步：确认部署

```
[6/6] 部署摘要
  ┌──────────────────────────────┐
  │ 域名:     company.com        │
  │ 主站:     launchpad.corp...  │
  │ SSL:      Let's Encrypt      │
  │ 数据库:   内置                │
  │ K8s:      内置 k3s           │
  └──────────────────────────────┘
  开始部署? [Y/n]:
```

## 目录结构

```
launchpad-setup/
├── launchpad/                  # 现有目录：基础配置参考（不修改）
├── gateway/                    # 现有目录：基础配置参考（不修改）
├── helm/                       # 现有目录：K8s Helm 配置（不修改）
├── register.sh                 # 现有文件：集群注册脚本参考（不修改）
├── README.md
│
└── deploy/                     # 新建：一键部署专用目录
    ├── setup.sh                # 主入口
    ├── scripts/
    │   ├── lib/
    │   │   ├── common.sh       # 公共函数：颜色输出、日志、校验
    │   │   ├── i18n.sh         # 语言加载
    │   │   ├── detect.sh       # 环境预检：OS、Docker、端口、磁盘、内存
    │   │   ├── interact.sh     # 交互式参数收集（基础 + 高级）
    │   │   ├── secrets.sh      # 密钥/密码生成（JWT、ZT RSA、数据库密码等）
    │   │   ├── render.sh       # 模板渲染：.env + nginx.conf
    │   │   ├── certs.sh        # SSL 证书：Let's Encrypt (acme.sh) / 自带证书
    │   │   ├── k3s.sh          # k3s 安装 + 禁用 Traefik / 外部 K8s 验证
    │   │   ├── database.sh     # PostgreSQL 初始化（6 个 schema + 用户）+ Gitea 数据库
    │   │   └── deploy.sh       # docker compose up + 健康检查 + 输出结果
    │   └── lang/
    │       ├── en.sh           # 英文语言包
    │       └── zh.sh           # 中文语言包
    ├── templates/
    │   ├── env.template        # Launchpad .env 模板
    │   ├── env.gateway.template # Gateway .env 模板
    │   ├── nginx.conf.template # 合并后的单层 Nginx 配置模板
    │   ├── settings.yml.template # 运行时热加载配置模板
    │   └── docker-compose.yml.template  # 完整编排模板（Nginx/Gitea/PG/Redis + 全部服务）
    └── generated/              # 运行时生成（加入 .gitignore）
        ├── launchpad/.env
        ├── launchpad/config/settings.yml
        ├── gateway/.env
        ├── nginx/
        │   ├── nginx.conf
        │   └── certs/          # SSL 证书文件
        ├── docker-compose.yml
        └── .setup.conf         # 用户配置持久化（重新运行时加载）
```

## 模板渲染

### 渲染方式

`.env` 和 `nginx.conf` 模板使用 `envsubst` 进行简单变量替换（零依赖）。`docker-compose.yml.template` 使用 `render.sh` 中基于 Bash 的渲染器处理条件逻辑（如根据内置/外部模式决定是否包含 `postgres` 和 `redis` 服务），避免引入 `jinja2` 或 `gomplate` 等额外依赖。

```bash
render_template() {
    local template="$1"
    local output="$2"
    envsubst < "$template" > "$output"
}

# Docker Compose：使用 Bash heredoc + 条件判断
render_compose() {
    # 根据 DB_MODE=builtin/external、K8S_MODE=builtin/external
    # 使用 shell 条件语句生成 docker-compose.yml
    # 输出到 generated/docker-compose.yml
}
```

### 模板变量

**基础变量（必填）：**

| 变量 | 示例 | 来源 |
|------|------|------|
| `DOMAIN` | `company.com` | 用户输入 |
| `SUBDOMAIN` | `corp` | 用户输入（默认：`corp`） |
| `LAUNCHPAD_DOMAIN` | `launchpad.corp.company.com` | 自动派生 |
| `GITEA_DOMAIN` | `launchpad-gitea.corp.company.com` | 自动派生 |
| `WILDCARD_DOMAIN` | `company.com` | 等于 DOMAIN |

**自动生成（secrets.sh）：**

| 变量 | 生成方式 |
|------|----------|
| `JWT_SECRET` | `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | `openssl rand -hex 32` |
| `SESSION_SECRET` | `openssl rand -hex 32` |
| `ENCRYPTION_KEY` | `openssl rand -hex 16` |
| `CLAUDE_CREDENTIALS_ENCRYPTION_KEY` | `openssl rand -hex 16`（恰好 32 字符） |
| `SSH_KEY_ENCRYPTION_SECRET` | `openssl rand -hex 16`（32+ 字符） |
| `ZT_PRIVATE_KEY` | `openssl genrsa 2048 \| base64` |
| `ZT_PUBLIC_KEY` | 从私钥派生 |
| `DB_PASSWORD_MAIN` | `openssl rand -base64 18` |
| `DB_PASSWORD_MONITORING` | `openssl rand -base64 18` |
| `DB_PASSWORD_EVENTS` | `openssl rand -base64 18` |
| `DB_PASSWORD_BILLING` | `openssl rand -base64 18` |
| `DB_PASSWORD_STATS` | `openssl rand -base64 18` |
| `DB_PASSWORD_GATEWAY` | `openssl rand -base64 18` |
| `GITEA_DB_PASSWORD` | `openssl rand -base64 18` |
| `REDIS_PASSWORD` | `openssl rand -base64 18` |
| `ANI_CODE_GATEWAY_API_KEY` | `openssl rand -hex 16` |
| `LAUNCHPAD_INTERNAL_SECRET` | `openssl rand -hex 16` |

**域名派生变量（render.sh 自动计算）：**

| 变量 | 派生规则 |
|------|----------|
| `EXTERNAL_DOMAIN` | `https://${LAUNCHPAD_DOMAIN}` |
| `PUBLIC_DOMAIN` | `${LAUNCHPAD_DOMAIN}` |
| `PUBLIC_URL` | `https://${LAUNCHPAD_DOMAIN}` |
| `EXTERNAL_API_URL` | `https://${LAUNCHPAD_DOMAIN}/api` |
| `INTERNAL_API_URL` | `https://${LAUNCHPAD_DOMAIN}/api` |
| `INTERNAL_ADMIN_URL` | `https://${LAUNCHPAD_DOMAIN}/admin/adminapi` |
| `CORS_ORIGINS` | `https://${LAUNCHPAD_DOMAIN}` |
| `INGRESS_DOMAIN` | `${DOMAIN}` |
| `BASE_DOMAIN` | `${DOMAIN}` |
| `ANI_CODE_GATEWAY_URL` | `https://${LAUNCHPAD_DOMAIN}/gatewayproxy` |
| `ANI_CODE_RELEASE_URL` | `https://${GITEA_DOMAIN}/launchpad/ani-code/archive/main.tar.gz` |
| `GITEA_ROOT_URL` | `https://${GITEA_DOMAIN}` |
| `GITEA_GIT_SSH` | `git@${GITEA_DOMAIN}:2222` |
| `ENTRA_REDIRECT_URI` | `https://${LAUNCHPAD_DOMAIN}/api/v1/auth/microsoft/callback` |
| `GATEWAY_PUBLIC_URL` | `https://${LAUNCHPAD_DOMAIN}/gatewayproxy` |

**模式相关变量：**

| 变量 | 内置模式 | 外部模式 |
|------|----------|----------|
| `DB_HOST` | `postgres`（Docker 服务名） | 用户提供的 IP |
| `DEFAULT_BACKEND` | `localhost` | 用户提供的 ingress 域名 |
| `STORAGE_CLASS` | `local-path`（k3s 默认） | 用户提供（如 `managed-csi`） |

## K8s/k3s 集成

### 内置 k3s 模式

```
安装 k3s ──→ 部署 Launchpad ──→ 等待 API 就绪 ──→ 下载 register.sh ──→ 执行注册
    │                                                    │
    ├─ curl get.k3s.io                                   └─ curl -fsSL
    ├─ --disable traefik                                    https://{LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh
    ├─ --tls-san <本机IP>
    └─ 导出 kubeconfig
```

### 外部 K8s 模式

```
部署 Launchpad ──→ 等待 API 就绪 ──→ 下载 register.sh ──→ 使用 --skip-k3s-check 执行
```

### register.sh 下载

注册脚本始终从已部署的 Launchpad 实例下载，确保与运行中的 API 镜像版本一致：

```bash
register_cluster() {
    wait_for_api

    curl -fsSL "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh" \
        -o /tmp/register.sh
    chmod +x /tmp/register.sh

    if [ "$K8S_MODE" = "builtin" ]; then
        LAUNCHPAD_REGISTRATION_URL="http://api:6802/admin/adminapi/k3s/register" \
        CLUSTER_NAME="local-k3s" \
            /tmp/register.sh
    else
        LAUNCHPAD_REGISTRATION_URL="http://api:6802/admin/adminapi/k3s/register" \
        KUBECONFIG="$K8S_KUBECONFIG_PATH" \
        CLUSTER_NAME="$K8S_CLUSTER_NAME" \
            /tmp/register.sh --skip-k3s-check
    fi
}
```

## Nginx 配置

单层 Nginx 取代之前的双层架构（外部 VM + 内部 Docker）。模板合并了两层的配置。

### Server Block 详情

**Block 1: `launchpad.{sub}.{domain}`（面板 + API + Gateway）**

- 使用配置的证书进行 TLS 终止
- `/api/v1/terminal/*` → WebSocket 代理到 API（3600s 超时）
- `/api/v1/headless-oauth/*/output` → SSE 代理，关闭缓冲（660s 超时）
- `/api/v1/auth/oauth-signup/*/output` → SSE 代理，关闭缓冲（660s 超时）
- `/api/socket.io/` → WebSocket 代理到 API（86400s 超时）
- `/gatewayproxy/` → 代理到 Gateway `:6555`（300s 超时）
- `/admin/` → 代理到 Admin `:6804`（限速）
- `/api/` → 代理到 API `:6802`（限速，100MB 上传限制）
- `/` → 代理到 UI `:6801`

**Block 2: `launchpad-gitea.{sub}.{domain}`（Gitea）**

- TLS 终止
- `/` → 代理到 Gitea `:3000`

**Block 3: `*.{domain}`（项目子域名）**

- 使用泛域名证书进行 TLS 终止
- `/` → 代理到 Router `:6580`（支持 WebSocket，86400s 超时）

**Block 4: HTTP 跳转**

- 80 端口，所有域名 → 301 跳转到 HTTPS

## Docker Compose 服务

```yaml
services:
  # ---- 基础设施（条件性：仅内置模式）----
  postgres:          # PostgreSQL 15, 端口 5432, 带健康检查
  redis:             # Redis 7 Alpine, 端口 6379, 带健康检查

  # ---- 核心服务 ----
  api:               # Launchpad API (:6802, :6804 管理后台)
                     #   挂载: logs/api, config/settings.yml, data/api, kubeconfig
  ui:                # Launchpad UI (:6801)
  router:            # 项目路由 (:6580)
  cron:              # 定时任务（挂载 pd/deploy/crontab）
  backup-worker:     # 备份队列工作进程

  # ---- 附加服务 ----
  gateway:           # ani-code Gateway (:6555), 挂载: gateway_data, logs
  gitea:             # Gitea (:3000, SSH :2222), 挂载: gitea_data

  # ---- 入口 ----
  nginx:             # TLS + 反向代理 (:80, :443)
                     #   挂载: nginx.conf, certs/
```

### Docker 镜像

镜像通过可配置的仓库前缀引用。脚本在基础配置中提示输入镜像仓库地址：

```
  镜像仓库地址 [ghcr.io/anilaunchpad]:
```

基础配置中提示输入统一版本号：

```
  镜像仓库地址 [ghcr.io/anilaunchpad]:
  镜像版本 [latest]:
```

所有服务默认使用相同版本：`{REGISTRY}/launchpad-api:{VERSION}`、`{REGISTRY}/launchpad-ui:{VERSION}` 等。

高级配置（"性能调优"模块）中可单独覆盖每个服务的版本：

```
  ── 镜像版本 ──
  统一版本已设为: 1.18.7
  如需单独指定，请输入 (留空保持统一版本):
    API     版本 [1.18.7]:
    UI      版本 [1.18.7]: 1.18.6
    Router  版本 [1.18.7]: 1.16.2
    Gateway 版本 [1.18.7]:
    Gitea   版本 [1.18.7]: 1.21
```

模板变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `IMAGE_REGISTRY` | `ghcr.io/anilaunchpad` | 仓库前缀 |
| `IMAGE_VERSION` | `latest` | 统一版本 |
| `IMAGE_VERSION_API` | `${IMAGE_VERSION}` | API 单独覆盖 |
| `IMAGE_VERSION_UI` | `${IMAGE_VERSION}` | UI 单独覆盖 |
| `IMAGE_VERSION_ROUTER` | `${IMAGE_VERSION}` | Router 单独覆盖 |
| `IMAGE_VERSION_GATEWAY` | `${IMAGE_VERSION}` | Gateway 单独覆盖 |
| `IMAGE_VERSION_GITEA` | `${IMAGE_VERSION}` | Gitea 单独覆盖 |

支持任意来源：公共仓库、私有仓库（用户事先自行完成 `docker login`）、或已预加载的本地镜像。脚本在启动部署前验证镜像是否可拉取。

## 部署执行顺序

```bash
deploy_services() {
    # 阶段 1：基础设施
    docker compose up -d postgres redis
    wait_for_healthy postgres 30
    wait_for_healthy redis 15

    # 阶段 2：数据库初始化（首次部署，内置模式）
    init_postgres_schemas    # 6 个 schema: main, monitoring, events, billing, stats, gateway
    init_gitea_database      # gitea 数据库

    # 阶段 3：应用服务
    docker compose up -d api ui router cron backup-worker gateway gitea
    wait_for_healthy api 60
    wait_for_healthy ui 30
    wait_for_healthy router 30
    wait_for_healthy gitea 45

    # 阶段 4：Nginx 入口（最后启动，确保所有后端已就绪）
    docker compose up -d nginx
    wait_for_healthy nginx 15

    # 阶段 5：Gitea 引导（仅首次部署）
    bootstrap_gitea          # 见下方"Gitea 引导"章节

    # 阶段 6：集群注册
    register_cluster
}
```

### 健康检查机制

每个服务在 Docker Compose 中定义了 healthcheck。脚本轮询 `docker inspect` 检查健康状态，每个服务有独立的超时时间。如果超时失败，显示最后 20 行日志。

## Gitea 引导

Gitea 首次启动后，脚本通过 Gitea API 自动化初始设置：

1. **创建管理员用户** — `POST /api/v1/admin/users`（或通过 Gitea 首次运行安装 API）
2. **创建 `launchpad` 组织** — `POST /api/v1/orgs`
3. **创建 `ani-code` 仓库** — `POST /api/v1/orgs/launchpad/repos`
4. **生成访问令牌** — `POST /api/v1/users/{admin}/tokens` → 存储为 `GITEA_ACCESS_TOKEN`
5. **配置 SSH** — Gitea 在端口 2222 监听 git SSH

生成的 `GITEA_ACCESS_TOKEN` 写回 `generated/launchpad/.env`，然后重启 API 容器使其生效。

如果 Gitea 已初始化（重复运行），此阶段跳过。

## SSL 证书续期

使用 Let's Encrypt 通过 `acme.sh` 时：

- `acme.sh` 自动安装续期 cron 任务（每天运行，60 天时续期）
- `--reloadcmd` 设置为续期后重载 Nginx 容器：
  ```bash
  acme.sh --install-cert -d "${DOMAIN}" \
      --key-file "generated/nginx/certs/..." \
      --fullchain-file "generated/nginx/certs/..." \
      --reloadcmd "docker compose -f generated/docker-compose.yml exec nginx nginx -s reload"
  ```
- 证书文件通过 volume 挂载到 Nginx 容器

## 错误处理与回滚

### 部署失败策略

部分失败时，脚本**不会**自动拆除已运行的服务。替代方案：

1. **清晰记录失败阶段**，显示错误信息和相关容器日志
2. **保留基础设施运行** — PostgreSQL、Redis 和已成功启动的服务保持运行
3. **提供恢复命令**：
   ```
   ✗ 阶段 3 失败: API 服务未通过健康检查

   从此阶段重试:
     ./deploy/setup.sh --resume

   查看日志:
     docker compose -f deploy/generated/docker-compose.yml logs api

   卸载全部:
     ./deploy/setup.sh --uninstall
   ```
4. **`--resume` 参数** — 跳过已完成的阶段（检查容器健康状态），从失败阶段重试

### register.sh 访问方式

`register_cluster()` 函数通过 Docker 网络访问 API，而非 `localhost`：

```bash
# 使用 Docker 网络确保可靠访问（API 端口可能未暴露到宿主机）
LAUNCHPAD_REGISTRATION_URL="http://api:6802/admin/adminapi/k3s/register"

# 下载 register.sh 使用公共 HTTPS URL
curl -fsSL "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh"
```

如果公共 URL 尚不可访问（DNS 未配置），回退到在 Docker 网络内部执行 curl：

```bash
docker compose exec api curl -fsSL "http://localhost:6802/admin/adminapi/k3s/register.sh"
```

## 最终输出

```
═══════════════════════════════════════════════════
  ✓ AniLaunchpad 部署完成!
═══════════════════════════════════════════════════

  访问地址:
    主站面板:  https://launchpad.corp.company.com
    Gitea:     https://launchpad-gitea.corp.company.com
    管理后台:  https://launchpad.corp.company.com/admin

  管理员账号:
    邮箱:      admin@company.com
    密码:      <自动生成>

  K8s 集群:
    模式:      内置 k3s
    状态:      ✓ 已注册

  DNS 配置提醒:
    请确保以下 DNS 记录指向本机 IP (x.x.x.x):
    ├─ launchpad.corp.company.com       → x.x.x.x
    ├─ launchpad-gitea.corp.company.com → x.x.x.x
    └─ *.company.com                     → x.x.x.x

  配置文件: deploy/generated/
  查看日志: docker compose -f deploy/generated/docker-compose.yml logs -f
  重新配置: ./deploy/setup.sh --reconfigure

═══════════════════════════════════════════════════
```

## 命令行参数

```bash
./deploy/setup.sh                  # 全新安装（交互式）
./deploy/setup.sh --reconfigure    # 重新配置（加载上次 .setup.conf）
./deploy/setup.sh --status         # 查看服务状态
./deploy/setup.sh --upgrade        # 更新镜像版本
./deploy/setup.sh --uninstall      # 卸载（需确认）
./deploy/setup.sh --resume         # 从上次失败处继续
```

## 双语支持

语言包位于 `scripts/lang/{en,zh}.sh`。所有面向用户的文案通过变量引用：

```bash
# lang/en.sh
MSG_WELCOME="AniLaunchpad Setup Script"
MSG_DOMAIN_PROMPT="Enter your main domain (e.g. company.com)"

# lang/zh.sh
MSG_WELCOME="AniLaunchpad 一键部署脚本"
MSG_DOMAIN_PROMPT="请输入您的主域名 (例: company.com)"
```

启动时选择语言，通过 `source "lang/${LANG_CHOICE}.sh"` 加载。

## 配置持久化

用户输入的参数在部署成功后保存到 `generated/.setup.conf`。重新运行时（`--reconfigure`），脚本加载上次配置作为默认值，用户只需修改变更的部分。

## 幂等性

重复运行脚本时检测已有状态：
- 已运行的容器 → 提示升级或跳过
- 已有证书 → 除非即将过期否则跳过续期
- 已有数据库 schema → 跳过初始化
- 已有 k3s → 跳过安装
