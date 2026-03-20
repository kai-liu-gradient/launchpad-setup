# AniLaunchpad Setup

[English](README.md)

AniLaunchpad 全栈平台的一键部署工具 — 通过交互式向导完成基础设施配置、服务部署，一次运行全部上线。

## 功能特性

- **交互式向导** — 6 页终端 UI，支持方向键导航和前进/后退
- **非交互模式** — 传入配置文件实现全自动部署（`--config FILE`）
- **中英双语** — 启动时选择语言
- **灵活的基础设施** — 内置 K3s 或对接外部 Kubernetes 集群
- **3 种 SSL 模式** — Let's Encrypt（DNS 验证通配符证书）、自签名 CA、自定义证书
- **内置或外部数据库** — 自动部署 PostgreSQL + Redis，或连接已有实例
- **统一进度显示** — 带逐步 spinner 的实时进度条（TTY 模式），或简单进度条（非 TTY 模式）
- **断点续跑** — `--resume` 从上次失败的位置继续
- **模板自动导入** — 自动推送 Git 仓库并导入 YAML 模板到 Gitea + API
- **完整卸载** — `--uninstall-all` 移除服务、K3s、定时任务、配置和 DNS 条目

## 前置条件

| 依赖 | 最低版本 | 说明 |
|---|---|---|
| **Linux** | 任意现代发行版 | 不支持 macOS / Windows |
| **Docker** | 20.10+ | 必须运行中（`docker info`） |
| **Docker Compose** | v2 | 插件格式（`docker compose`） |
| **jq** | 1.6+ | 集群注册使用 |
| **envsubst** | 任意 | `gettext-base` 包的一部分 |
| **openssl** | 1.1+ | 密钥/证书生成 |
| **helm** | 3.x | 仅 `K8S_MODE=builtin` 时需要 |
| **curl** | 任意 | Let's Encrypt + 集群注册 |

## 快速开始

```bash
# 交互模式（首次安装推荐）
sudo ./deploy/setup.sh

# 非交互模式，使用配置文件
sudo ./deploy/setup.sh --config deploy/presets/test-server.conf
```

向导依次引导：域名配置、SSL 模式、数据库模式、Kubernetes 模式、高级选项（SMTP、SSO、计费、AI、存储、性能调优）。

---

## 系统架构

### 请求路由全景

系统中有两种不同的流量路径：

```mermaid
graph TD
    subgraph "外部流量"
        Browser["浏览器"]
    end

    subgraph "Nginx（TLS 终止 — 端口 80/443）"
        NG["nginx:alpine"]
    end

    subgraph "Docker Compose 服务"
        UI["UI :6801"]
        API["API :6802 / :6804 管理端口"]
        Router["Router :6580"]
        GW["Gateway :6555"]
        Gitea["Gitea :3000 / :2222 SSH"]
        Cron["Cron 定时任务"]
        Backup["Backup Worker"]
    end

    subgraph "数据层"
        PG["PostgreSQL :5432<br/>6 个 schema + gitea"]
        Redis["Redis :6379"]
    end

    subgraph "Kubernetes 集群"
        Ingress["Ingress-Nginx :30080"]
        CoreDNS["CoreDNS"]
        Kyverno["Kyverno<br/>（仅自签名模式）"]
        Pods["用户项目 Pod"]
    end

    Browser -->|"https://launchpad.sub.domain"| NG
    Browser -->|"https://project.domain"| NG

    NG -->|"/dashboard 路由"| UI
    NG -->|"/api/* 路由"| API
    NG -->|"*.domain（通配符）"| Router
    NG -->|"/gatewayproxy/*"| GW
    NG -->|"gitea 子域名"| Gitea

    UI --> API
    API --> PG
    API --> Redis
    GW --> PG
    Router -->|"查询路由规则"| API
    Router -->|"代理到 Pod"| Ingress
    Ingress --> Pods
    CoreDNS -.->|"解析 *.domain"| Ingress
    Kyverno -.->|"注入 CA 证书"| Pods
```

**路径 A — Dashboard 访问**（`https://launchpad.sub.domain`）：
浏览器 → Nginx → UI（前端）→ API（后端）→ 管理项目、部署、用户

**路径 B — 用户服务域名访问**（`https://myproject.domain`）：
浏览器 → Nginx → Router → 查询 API 获取路由规则 → 代理到 K8s Ingress-Nginx（:30080）→ 用户 Pod

### 部署模式

系统支持两种根本不同的部署拓扑：

#### 模式 1：自签名证书 + 内置 K3s（单机部署）

适用于：开发、测试、内网/离线环境。

```mermaid
graph LR
    subgraph "单台服务器"
        subgraph "Docker Compose"
            Nginx2["Nginx<br/>自签名通配符证书"]
            Services2["API / UI / Router<br/>Gateway / Gitea"]
            DB2["PostgreSQL + Redis"]
        end
        subgraph "K3s（内置）"
            Ingress2["Ingress-Nginx<br/>NodePort 30080"]
            CoreDNS2["CoreDNS<br/>自定义 *.domain → 宿主机 IP"]
            Kyverno2["Kyverno<br/>CA 证书注入"]
            dnsmasq2["dnsmasq<br/>Docker DNS → 宿主机 IP"]
        end
    end
```

关键特征：
- K3s 自动安装（禁用 Traefik，替换为 Ingress-Nginx）
- 生成自签名 CA → `*.domain` 通配符证书
- **CoreDNS 自定义配置**：Pod 解析 `*.domain` → Ingress ClusterIP
- **dnsmasq**：Docker 容器解析 `*.domain` → 宿主机 IP（Gateway → Nginx → Pod 链路）
- **Kyverno ClusterPolicy**：通过 initContainer 注入 CA 证书到所有 Pod（与系统 CA bundle 合并）
- Gateway 容器设置 `NODE_EXTRA_CA_CERTS` 以信任自签名 CA
- API + Gateway 设置 `NODE_TLS_REJECT_UNAUTHORIZED=0`

#### 模式 2：公有证书 + 外部集群

适用于：生产环境、多节点、公网部署。

```mermaid
graph LR
    subgraph "部署服务器"
        subgraph "Docker Compose"
            Nginx3["Nginx<br/>Let's Encrypt / 自定义证书"]
            Services3["API / UI / Router<br/>Gateway / Gitea"]
            DB3["PostgreSQL + Redis<br/>（或外部实例）"]
        end
    end
    subgraph "外部 K8s 集群"
        ExtIngress["已有 Ingress Controller"]
        ExtPods["用户项目 Pod"]
    end
    Services3 -->|"kubeconfig"| ExtIngress
    ExtIngress --> ExtPods
```

关键特征：
- Let's Encrypt 通配符证书，通过 `acme.sh` + DNS API（Cloudflare、阿里云、Azure）
- 或使用自定义证书（`SSL_MODE=custom`）
- 通过 kubeconfig 文件连接外部 K8s 集群
- 无需 Kyverno、CoreDNS 自定义配置、dnsmasq
- 集群自带 Ingress Controller；Router 代理到其域名
- 数据库也可使用外部实例（`DB_MODE=external`）

### 项目结构

```
launchpad-setup/
├── deploy/
│   ├── setup.sh                 # 入口 — CLI 参数解析 + 流程编排
│   ├── versions.conf            # 镜像版本（修改此文件即可升级）
│   ├── presets/                  # 预设配置文件（用于非交互部署）
│   ├── scripts/
│   │   ├── lang/
│   │   │   ├── en.sh            # 英文消息
│   │   │   └── zh.sh            # 中文消息
│   │   └── lib/
│   │       ├── common.sh        # UI：日志、进度条、提示、校验
│   │       ├── i18n.sh          # 语言选择
│   │       ├── detect.sh        # 环境检测 + IP 探测
│   │       ├── interact.sh      # 6 页向导 + save_config()
│   │       ├── secrets.sh       # 密钥生成
│   │       ├── render.sh        # 模板渲染（envsubst + sed）
│   │       ├── certs.sh         # SSL：Let's Encrypt / 自签名 / 自定义
│   │       ├── k3s.sh           # K3s 安装 + 集群注册
│   │       ├── k8s-components.sh# Ingress-Nginx、CoreDNS、Kyverno
│   │       ├── database.sh      # PostgreSQL schema/用户创建
│   │       └── deploy.sh        # 部署编排 + 生命周期管理
│   ├── templates/               # 配置模板（envsubst 占位符）
│   │   ├── env.template         # → generated/launchpad/.env
│   │   ├── env.gateway.template # → generated/gateway/.env
│   │   ├── nginx.conf.template  # → generated/nginx/nginx.conf
│   │   ├── settings.yml.template# → generated/launchpad/config/settings.yml
│   │   ├── coredns-custom.yaml.template
│   │   ├── kyverno-inject-ca.yaml
│   │   ├── kyverno-sync-ca.yaml
│   │   └── values-builtin.yml   # ingress-nginx 的 Helm values
│   └── generated/               # 运行时输出（已 gitignore）
│       ├── .setup.conf          # 保存的配置
│       ├── .secrets             # 生成的密钥（chmod 600）
│       ├── docker-compose.yml   # 渲染后的 compose 文件
│       ├── launchpad/.env       # 应用环境变量
│       ├── gateway/.env         # Gateway 环境变量
│       └── nginx/               # Nginx 配置 + SSL 证书
├── files/                       # 模板仓库 + YAML 定义（用于导入）
│   ├── *.tar.gz                 # Git 仓库归档 → 推送到 Gitea
│   └── *.yaml                   # 模板定义 → 通过 admin API 导入
├── tests/
│   ├── e2e.sh                   # E2E 测试运行器
│   ├── cases/                   # 10 步测试场景
│   └── lib/                     # 测试工具库
└── docs/                        # 技术文档
```

### 模块依赖关系

| 模块 | 职责 | 依赖 |
|---|---|---|
| `common.sh` | 日志、进度条、样式化 UI、校验 | （无 — 最先加载） |
| `i18n.sh` | 语言选择提示 | `common.sh` |
| `detect.sh` | OS/Docker/工具检测、IP 探测 | `common.sh` |
| `interact.sh` | 6 页向导、`save_config()` | `common.sh`、`detect.sh` |
| `secrets.sh` | 生成/保存/加载密钥 | `common.sh` |
| `render.sh` | 模板 → 生成配置渲染 | `common.sh`、`detect.sh`、`secrets.sh` |
| `certs.sh` | SSL 证书管理（3 种模式） | `common.sh`、`detect.sh` |
| `k3s.sh` | K3s 安装、外部 K8s 验证、集群注册 | `common.sh`、`detect.sh` |
| `k8s-components.sh` | Ingress-Nginx、CoreDNS、Kyverno、CA 分发 | `common.sh`、`detect.sh` |
| `database.sh` | PostgreSQL schema/用户创建 | `common.sh`、`secrets.sh` |
| `deploy.sh` | 部署编排、健康检查、生命周期（启动/停止/升级） | 以上全部 |

---

## 部署流程

```mermaid
flowchart TD
    Start(["./setup.sh"]) --> ParseArgs["解析 CLI 参数"]
    ParseArgs --> Mode{"--config 文件?"}
    Mode -->|是| LoadConfig["加载配置文件<br/>（非交互）"]
    Mode -->|否| Wizard["交互式向导<br/>6 页"]

    LoadConfig --> CheckEnv["check_environment()<br/>检测 Docker、Compose、jq、envsubst、helm"]
    Wizard --> CheckEnv

    CheckEnv --> Secrets["generate_secrets()<br/>JWT、加密密钥、RSA、<br/>数据库密码、Redis 密码"]
    Secrets --> Render["render_templates()<br/>.env、docker-compose.yml、<br/>nginx.conf、settings.yml"]
    Render --> SaveConfig["save_config() + save_secrets()"]

    SaveConfig --> Deploy["deploy_services()<br/>统一进度显示"]

    Deploy --> K3S{"需要 K3s?<br/>(builtin && 未运行)"}
    K3S -->|是| InstK3S["install_k3s()"]
    K3S -->|否| SSL
    InstK3S --> SSL["setup_certificates()"]

    SSL --> Builtin{"K8S_MODE=builtin?"}
    Builtin -->|是| Ingress["install_ingress_nginx()"]
    Builtin -->|否| Infra
    Ingress --> CoreDNS["configure_coredns()"]
    CoreDNS --> SelfSigned{"SSL_MODE=selfsigned?"}
    SelfSigned -->|是| KyvernoStep["install_kyverno()<br/>setup_cert_distribution()"]
    SelfSigned -->|否| Infra
    KyvernoStep --> Infra

    Infra["启动 PostgreSQL + Redis"] --> DBInit["init_database()<br/>6 个 schema + Prisma push"]
    DBInit --> GiteaStart["启动 Gitea<br/>bootstrap_gitea()"]
    GiteaStart --> AppServices["启动 API、UI、Router、<br/>Gateway、Cron、Backup"]
    AppServices --> NginxStart["启动 Nginx<br/>setup_hosts()"]
    NginxStart --> Templates["import_templates()<br/>Git 仓库 + YAML"]
    Templates --> Register["register_cluster()"]
    Register --> Done(["show_result()"])
```

### 阶段详解

#### 阶段 1：环境检测

`check_environment()` 验证所有前置条件已安装并运行。对于 `K8S_MODE=builtin`，还会自动安装缺失的 `dnsmasq`。

#### 阶段 2：配置收集

**交互模式**呈现 6 页向导：

| 页面 | 收集内容 | 关键变量 |
|---|---|---|
| 1 - 基础 | 域名、子域名、镜像仓库、版本号 | `DOMAIN`、`SUBDOMAIN`、`IMAGE_REGISTRY`、`IMAGE_VERSION_*` |
| 2 - SSL | 证书模式、DNS 提供商、API token | `SSL_MODE`、`DNS_PROVIDER`、`DNS_API_TOKEN` |
| 3 - 数据库 | 内置或外部、连接 URL | `DB_MODE`、`DATABASE_URL`、`REDIS_HOST` |
| 4 - Kubernetes | 内置 K3s 或外部集群 | `K8S_MODE`、`K8S_KUBECONFIG_PATH`、`STORAGE_CLASS` |
| 5 - 高级 | 管理员、SMTP、Stripe、SSO、AI、通知、存储、性能 | `ADMIN_EMAIL`、`SMTP_*`、`API_REPLICAS` 等 |
| 6 - 确认 | 审查所有设置，确认部署 | （只读） |

**非交互模式**从配置文件读取所有变量（参见 `deploy/presets/test-server.conf`）。

#### 阶段 3：密钥生成与模板渲染

**生成的密钥**（存储在 `generated/.secrets`，权限 600）：
- 3 个 JWT/会话密钥（hex-64）
- 3 个加密密钥（hex-32）
- 1 对 RSA 密钥（base64 编码，用于 Zero Trust）
- 8 个数据库密码（base64-24）
- 1 个 Redis 密码
- 2 个网关密钥
- 1 个管理员密码（如用户未提供）

**渲染的模板**（通过 `envsubst` + `sed` 占位符替换）：
- `docker-compose.yml` — 根据条件包含 PostgreSQL/Redis，设置镜像版本，内置模式注入 extra_hosts
- `.env` 文件 — API、Gateway 的所有环境变量
- `nginx.conf` — 反向代理规则、TLS 证书路径
- `settings.yml` — 应用设置

#### 阶段 4：统一部署

`deploy_services()` 在运行时根据部署模式构建动态步骤列表，条件性地包含各步骤：

| 步骤 | 条件 | 超时 | 执行内容 |
|---|---|---|---|
| K3s 安装 | builtin 且未运行 | 60s | `curl get.k3s.io`，等待节点就绪 |
| SSL 证书 | 始终 | — | 按 `SSL_MODE` 生成/获取证书 |
| Ingress-Nginx | builtin | 120s | Helm 安装，等待 controller Pod |
| CoreDNS 配置 | builtin | — | 自定义 DNS + dnsmasq 配置 |
| Kyverno | builtin 且 selfsigned | 120s | Helm 安装，等待 admission controller |
| CA 分发 | builtin 且 selfsigned | — | ConfigMap + ClusterPolicy 注入 CA |
| 基础设施 | 内置 DB | 30s | 启动 PostgreSQL + Redis，健康检查 |
| 数据库初始化 | 始终 | — | 创建 6 个 schema + Gitea DB，Prisma push（6 个 schema） |
| Gitea | 始终 | 90s | 启动，健康检查，创建管理员 + token + 组织 |
| API | 始终 | 90s | 启动，健康检查 |
| UI | 始终 | 60s | 启动，健康检查 |
| Router | 始终 | 60s | 启动，健康检查 |
| Nginx | 始终 | 30s | 启动，更新 `/etc/hosts`，等待 5s 确保 HTTPS 就绪 |
| 模板导入 | 始终 | — | 推送 Git 仓库到 Gitea，通过 admin API 导入 YAML |
| 集群注册 | 始终 | — | 下载并执行 `register.sh` 注册到 API |

#### 阶段 5：完成

`show_result()` 显示：
- 访问 URL（Dashboard、Gitea、管理后台）
- 管理员凭据
- K8s 集群状态
- DNS 配置提醒（需要将哪些域名指向服务器 IP）

---

## CLI 命令参考

### 安装与配置

| 命令 | 说明 |
|---|---|
| `./deploy/setup.sh` | 交互式向导全新安装 |
| `./deploy/setup.sh --config FILE` | 使用配置文件非交互安装 |
| `./deploy/setup.sh --reconfigure` | 重新进入配置向导（保留现有配置作为默认值） |

### 管理

| 命令 | 说明 |
|---|---|
| `./deploy/setup.sh --status` | 显示 Docker Compose 服务状态 |
| `./deploy/setup.sh --restart <service>` | 重启服务（`api\|ui\|router\|gateway\|nginx\|gitea\|all`） |
| `./deploy/setup.sh --upgrade` | 拉取最新镜像（按 `versions.conf`）并重启 |

### 细粒度操作

| 命令 | 说明 |
|---|---|
| `./deploy/setup.sh --resume` | 从已有配置恢复完整部署（失败后使用） |
| `./deploy/setup.sh --setup-k3s` | 仅安装/重新配置 K3s |
| `./deploy/setup.sh --setup-certs` | 仅重新生成 SSL 证书 |
| `./deploy/setup.sh --setup-db` | 仅运行数据库初始化 |
| `./deploy/setup.sh --import-templates` | 重新导入模板到 Gitea + API |

### 卸载

| 命令 | 说明 |
|---|---|
| `./deploy/setup.sh --uninstall` | 停止服务，移除容器和**所有数据** |
| `./deploy/setup.sh --uninstall-all` | 完全移除：服务 + K3s + 定时任务 + 配置 + DNS |

---

## 重要注意事项与已知陷阱

### Shell 脚本

| 陷阱 | 影响 | 解决方案 |
|---|---|---|
| 子 shell 中使用 `local` | `local` 仅在函数内有效。在 `( ... )` 子 shell 中使用会在 `set -e` 下静默失败，退出码 128 | 子 shell 中不使用 `local` |
| EXIT trap 被子 shell 继承 | `/dev/tty` 光标恢复 trap 在子 shell 中触发。通过 SSH（无 tty）时，重定向错误变成退出码 | 在子 shell 开头加 `trap - EXIT` |
| `grep` + `pipefail` | `grep pattern \| tail \| cut` — 如果 grep 无匹配，返回 exit 1，传播并终止脚本 | grep 管道末尾加 `\|\| true` |
| `envsubst` 只能看到 export 的变量 | `source .setup.conf` 设置 shell 变量但不导出。`envsubst` 是外部进程 | 显式 `export` 模板中引用的每个变量 |

### 网络与 DNS

| 陷阱 | 影响 | 解决方案 |
|---|---|---|
| Docker 容器无法解析 `*.domain` | Gateway 需要通过 Nginx:443 访问通配符 TLS，但 Docker DNS 不认识自定义域名 | 安装 dnsmasq 配置 `address=/DOMAIN/HOST_IP` |
| CoreDNS 环路检测 | 默认 CoreDNS 使用 `/etc/resolv.conf`，可能指向自身导致环路 | 注释掉 `loop` 插件，使用公共 DNS 转发器 |
| K3s 步骤跳过时 `KUBECONFIG` 未设置 | K3s 已运行则跳过 `install_k3s()`，KUBECONFIG 未设置。Helm/kubectl 默认连 `localhost:8080` | 在 `deploy_services()` 顶部无条件 export KUBECONFIG |

### SSL / TLS

| 陷阱 | 影响 | 解决方案 |
|---|---|---|
| 自签名 CA 不被 K8s Pod 中的 curl/wget 信任 | Kyverno 注入的 `NODE_EXTRA_CA_CERTS` 仅对 Node.js 有效。系统工具使用 `/etc/ssl/certs/ca-certificates.crt` | 用 initContainer 合并 CA 到系统 bundle，挂载为 volume |
| `DEFAULT_BACKEND` 不能包含协议前缀 | Router 代码内部会加 `http://`。如果值已含 `http://`，结果是 `http://http://host:port` | 在 `render.sh` 中去除 `http://`/`https://` 前缀 |

### API 与服务

| 陷阱 | 影响 | 解决方案 |
|---|---|---|
| 模板导入使用 admin 端口（6804），非 API 端口（6802） | 6802 需要 JWT 认证（此时还没有注册用户）。6804 是无需认证的管理 API | 使用 `localhost:6804/api/templates/yaml/create` |
| API 容器只有 `wget`，没有 `curl` | API Docker 镜像未安装 curl | 用 `docker compose exec -T api wget -q -O-` 调用 |
| 健康检查端点是 `/health`，不是 `/api/health` | `/api/` 前缀仅用于 API 路由 | 检查 `http://localhost:6802/health` |
| 空的 wget 响应被当作成功 | `wget` 错误输出到 stderr；`2>/dev/null \|\| echo ""` 后变量为空，`grep "error"` 返回 false | 用 `2>&1 \|\| echo "WGET_FAILED"`，显式检查 `"success":true` |
| Git commit 需要 committer 身份 | `git commit --author=` 设置作者，但 committer 身份也是必需的。远程服务器无全局 git 配置 | 设置 `GIT_AUTHOR_NAME/EMAIL` 和 `GIT_COMMITTER_NAME/EMAIL` 环境变量 |

### 配置

| 陷阱 | 影响 | 解决方案 |
|---|---|---|
| 各服务镜像版本不同 | Router（1.16.2）和 Gateway（1.18.7）与 API/UI（2.0.3）版本不同 | 使用按服务区分的版本变量：`IMAGE_VERSION_ROUTER`、`IMAGE_VERSION_GATEWAY` 等 |
| `GITEA_ACCESS_TOKEN` 在 `.env` 中，不在 `.secrets` 中 | Token 由 `bootstrap_gitea()` 运行时生成。`--import-templates` 需从 `.env` 读取 | 从 `generated/launchpad/.env` 读取，而非 `.secrets` |
| `kubectl wait` 在资源不存在时立即失败 | 报 "no matching resources found" 而不是等待 | 先轮询等待资源出现，再调用 `kubectl wait` |
| 多阶段 sed 注入产生重复 YAML key | 如果多个渲染阶段各自在同一服务下插入 `environment:`，YAML 拒绝重复 key | 追加到现有块中，使用锚点行定位，不创建新映射 key |

---

## 故障排除

### 查看日志

```bash
# 所有服务
docker compose -f deploy/generated/docker-compose.yml logs -f

# 特定服务
docker compose -f deploy/generated/docker-compose.yml logs -f api

# K3s / kubectl
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get pods -A
kubectl logs -n <namespace> <pod-name>
```

### 常见问题

| 症状 | 可能原因 | 修复方法 |
|---|---|---|
| 脚本静默退出（exit 128） | 子 shell 中使用 `local` 或 SSH 下 EXIT trap | 检查子 shell 使用模式，加 `trap - EXIT` |
| 端口 6802 "connection refused" | API 尚未就绪 | `./setup.sh --resume` 或查看 `docker compose logs api` |
| Gitea HTTPS 不可达 | DNS 未配置或 Nginx 未就绪 | 等待 Nginx 启动；检查 `/etc/hosts` 或 DNS 记录 |
| 模板导入时 Git push 失败 | SSL 证书不受信任 / Gitea 未就绪 | 自签名模式：`GIT_SSL_NO_VERIFY=1`；检查 Gitea 健康状态 |
| Pod 无法访问 `*.domain` | CoreDNS 或 dnsmasq 未配置 | 检查 `kubectl get cm coredns-custom -n kube-system` 和 `/etc/dnsmasq.d/launchpad.conf` |
| `helm: command not found` | Helm 未安装 | 安装 Helm 3：`curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \| bash` |
| 集群注册失败 | API 公网 URL 不可达或 DNS 未设置 | 检查 `curl -k https://LAUNCHPAD_DOMAIN/health`；用 `--resume` 重试 |
| 数据库 "role already exists" | 对已有 DB 重复运行初始化 | 可安全忽略 — `IF NOT EXISTS` 已做保护 |

### 恢复命令

```bash
# 失败后恢复（从断点继续）
./deploy/setup.sh --resume

# 仅重新导入模板
./deploy/setup.sh --import-templates

# 重启异常服务
./deploy/setup.sh --restart api

# 完全重装（警告：删除所有数据）
./deploy/setup.sh --uninstall-all
./deploy/setup.sh
```

---

## 配置文件参考

非交互部署时，创建配置文件（参见 `deploy/presets/test-server.conf`）：

```bash
# 必填
DOMAIN="example.com"
SUBDOMAIN="corp"
SSL_MODE="selfsigned"           # letsencrypt | selfsigned | custom
DB_MODE="builtin"               # builtin | external
K8S_MODE="builtin"              # builtin | external
ADMIN_EMAIL="admin@example.com"

# 镜像版本
IMAGE_REGISTRY="swr.ap-southeast-1.myhuaweicloud.com/ghisha"
IMAGE_VERSION_API="2.0.3"
IMAGE_VERSION_UI="2.0.3"
IMAGE_VERSION_ROUTER="1.16.2"
IMAGE_VERSION_GATEWAY="1.18.7"
IMAGE_VERSION_GITEA="1.25-rootless"

# 可选：K8s
K8S_KUBECONFIG_PATH="/etc/rancher/k3s/k3s.yaml"
STORAGE_CLASS="local-path"

# 可选：Let's Encrypt
DNS_PROVIDER="cloudflare"       # cloudflare | aliyun | azure | manual
DNS_API_TOKEN="your-token"

# 可选：外部数据库
DATABASE_URL="postgresql://user:pass@host:5432/db?schema=launchpad_main"
REDIS_HOST="redis.example.com"
REDIS_PORT="6379"
REDIS_PASSWORD="your-password"

# 可选：高级选项
ADMIN_PASSWORD="custom-password"
SMTP_HOST="smtp.gmail.com"
API_REPLICAS="2"
ROUTER_REPLICAS="2"
GATEWAY_REPLICAS="2"
```

---

## 许可证

Copyright (c) Gradient8. All rights reserved.
