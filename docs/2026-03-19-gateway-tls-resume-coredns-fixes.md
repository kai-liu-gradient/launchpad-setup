# Gateway TLS + --resume 重设计 + CoreDNS envsubst 修复

日期: 2026-03-19

## 1. Gateway 缺少 NODE_TLS_REJECT_UNAUTHORIZED

### 问题

Gateway 容器访问 daemon HTTPS 端点时报 `unable to verify the first certificate`，health check 连续失败后移除 daemon 实例。

之前的修复（testserver-fixes #6）只在 `generated/gateway/.env` 手动加了 `NODE_TLS_REJECT_UNAUTHORIZED=0`，但 render 模板里没加，重新 render 后丢失。

### 根因

API 和 heartbeat 的 compose 模板中有 `NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__`（render 时根据 SSL_MODE 替换为 0 或 1），但 gateway 模板**缺少这个环境变量**。

### 修复

`deploy/scripts/lib/render.sh` — gateway service 模板添加 `environment:` section：

```yaml
  gateway:
    image: __IMAGE_REGISTRY__/ani-code-gateway:__IMAGE_VERSION_GATEWAY__
    ...
    env_file: ./gateway/.env
    environment:                                         # ← 新增
      - NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__ # ← 新增
    volumes:
      ...
```

render 时 `__NODE_TLS_REJECT__` 替换为 `0`（selfsigned）或 `1`（其他模式），与 API service 行为一致。

> **与 testserver-fixes-2026-03-17.md #6 的关系**: #6 的手动 .env 修改不再需要，compose 模板已自动处理。

### E2E 测试

`tests/cases/02-service-health.sh` 新增断言：检查 `deploy/generated/docker-compose.yml` 中 api 和 gateway service 都有 `NODE_TLS_REJECT_UNAUTHORIZED`。

---

## 2. --resume 重设计：调用完整 deploy_services()

### 问题

`--resume` 被设计为只重启 docker-compose 服务，跳过了 k3s/SSL/k8s-components/prisma 迁移。用户在早期阶段（如 SSL、ingress-nginx）失败后执行 `--resume`，不会重新尝试失败的步骤。

### 修复

删除 `resume_deploy()` 函数（78 行），`--resume` 改为加载所有模块后直接调用 `deploy_services()`。

`deploy/setup.sh` resume case 改为：

```bash
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
```

**关键点**：不重新 render — 使用已有的 `generated/` 配置。`deploy_services()` 每一步自带幂等检测（k3s 检查 `kubectl get nodes`、DB init 幂等、gitea bootstrap 检测 org 是否存在等），已成功的步骤会快速通过。

Help 文本中 `--resume` 从 "Teardown" 区移到 "Setup Phases" 区，描述改为 `Resume full deploy using existing config`。

> **与 2026-03-18-unified-deploy-progress.md 的关系**: 该文档提到 `_DS_K3S_STEP` 模式支持独立调用时（`--setup-k3s`, `--resume`）使用自己的进度条。现在 `--resume` 直接调用 `deploy_services()`，会使用 Style C 统一进度显示，`_DS_K3S_STEP` 模式仅在 `--setup-k3s` 独立调用时生效。

---

## 3. deploy_services() 缺少 KUBECONFIG export

### 问题

`--resume` 调用 `deploy_services()` 后，ingress-nginx helm install 失败（静默），报 `Kubernetes cluster unreachable: Get "http://localhost:8080/version"`。

### 根因

`export KUBECONFIG` 在 `install_k3s()` 内部设置。`deploy_services()` 检测到 k3s 已在运行（`kubectl get nodes` 成功）时跳过 `install_k3s()`，导致 KUBECONFIG 未 export。后续 helm 命令默认连 `localhost:8080`。

首次全量安装不受影响：wizard 完成后走 `deploy_services()`，此时 `install_k3s()` 一定会被调用（k3s 还未安装）。只有 `--resume`（k3s 已在运行）和未来任何跳过 k3s 步骤的路径会触发此 bug。

### 修复

`deploy/scripts/lib/deploy.sh` — `deploy_services()` 开头、k3s 检测之前，无条件 export KUBECONFIG：

```bash
deploy_services() {
    ...
    # Ensure KUBECONFIG is set for helm/kubectl (install_k3s exports this,
    # but if k3s is already running we skip that function)
    export KUBECONFIG="${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}"

    # k3s installation (only if builtin AND not already running)
    ...
```

---

## 4. configure_coredns() envsubst 变量未 export — CoreDNS CrashLoopBackOff

### 问题

`--resume` 执行到 CoreDNS 配置步骤后，CoreDNS 进入 CrashLoopBackOff，所有 pod DNS 解析失败。日志：
```
error inspecting server blocks: zone is not a valid domain name:
```

### 根因

`configure_coredns()` 使用 `envsubst` 渲染 `coredns-custom.yaml.template`，但只 export 了 3 个变量（`HOST_IP`、`INGRESS_CLUSTER_IP`、`DOMAIN_ESCAPED`），没有 export `DOMAIN`、`LAUNCHPAD_DOMAIN`、`GITEA_DOMAIN`。

这 3 个变量虽然被 `source .setup.conf` 设为 shell 变量，但 `envsubst` 是外部进程，只能看到 **export** 过的环境变量。未 export 的变量被替换为空字符串，生成的 Corefile 包含无效 zone：

```
# 渲染结果（错误）
:53 :53 {        # ← LAUNCHPAD_DOMAIN 和 GITEA_DOMAIN 为空
    hosts {
        10.233.201.133   # ← 域名为空
        ...
```

首次安装可能不受影响：install 路径中 `DOMAIN` 等变量在 wizard 或 `--config` 处理时被 export（shell 实现细节依赖），但 `--resume` 通过 `load_saved_config` → `source .setup.conf` 设置变量时不会自动 export。

### 修复

`deploy/scripts/lib/k8s-components.sh` — `configure_coredns()` 补上 3 个 export：

```bash
export HOST_IP="$host_ip"
export INGRESS_CLUSTER_IP="$cluster_ip"
export DOMAIN_ESCAPED="$domain_escaped"
export DOMAIN="${DOMAIN}"                    # ← 新增
export LAUNCHPAD_DOMAIN="${LAUNCHPAD_DOMAIN}" # ← 新增
export GITEA_DOMAIN="${GITEA_DOMAIN}"         # ← 新增
```

---

## 5. Gateway compose `environment:` 重复导致 YAML 解析失败

### 问题

重新 render 后 `docker compose up` 报错：
```
yaml: unmarshal errors:
  line 180: mapping key "environment" already defined at line 172
```

### 根因

Gateway service 在 compose 文件中出现了两个 `environment:` 块：

1. **模板内联**：`NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__`（本次 #1 添加）
2. **selfsigned CA 注入逻辑**（render.sh 第 434 行）：在 `gateway:` 行后插入独立的 `environment:` + `NODE_EXTRA_CA_CERTS`

YAML 不允许同一 mapping 下有重复 key。

### 修复

将 selfsigned CA 注入逻辑从"在 `gateway:` 后新建 `environment:` 块"改为"在已有 `environment:` 块内追加"。用 `env_file: ./gateway/.env` 作为 gateway 独有的锚点定位，在其后第二行（`NODE_TLS_REJECT_UNAUTHORIZED` 行前）插入 `NODE_EXTRA_CA_CERTS`：

```bash
# 之前（创建重复的 environment 块）
if [[ "$line" == "  gateway:" ]]; then
    echo "    environment:"
    echo "      - NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem"
fi

# 之后（在已有 environment 块内追加）
sed_i "/env_file: .\/gateway\/.env/{n;n;s|      - NODE_TLS_REJECT_UNAUTHORIZED=|      - NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem\n      - NODE_TLS_REJECT_UNAUTHORIZED=|;}" "$compose_file"
```

渲染结果：
```yaml
  gateway:
    ...
    env_file: ./gateway/.env
    environment:
      - NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem   # selfsigned CA
      - NODE_TLS_REJECT_UNAUTHORIZED=0                         # TLS 跳过
    volumes:
      - gateway-data:/app/data
      - gateway-logs:/app/logs
      - /path/to/ca.pem:/etc/ssl/certs/launchpad-ca.pem:ro    # CA 文件挂载
```

---

## 6. `DEFAULT_BACKEND` 带 `http://` 前缀 → Router `ENOTFOUND http`

### 问题

Router 日志持续报 `getaddrinfo ENOTFOUND http`（hostname 为 `http`），所有非 system-path 请求（如 gateway 回调 `/.ani-code-callback`）返回 502。System path 正常。

### 根因

`.setup.conf` 中存储了 `DEFAULT_BACKEND="http://10.233.201.133:30080"`（带协议前缀）。render.sh builtin 分支会覆盖为纯 `host:port`，但 `.setup.conf` 的值在 `save_config()` 时已经保存了带前缀的版本（历史残留或早期脚本写入）。

Router 代码对非 system-path 请求构造 backend URL 时做 `new URL('http://' + DEFAULT_BACKEND)`，双重前缀导致 URL 变为 `http://http://10.233.201.133:30080`，Node.js URL 解析把 `http` 当成 hostname。

### 修复

`deploy/scripts/lib/render.sh` — 在 `DEFAULT_BACKEND` 设置逻辑之后，无条件 strip 协议前缀：

```bash
# Strip accidental http:// prefix — router adds the protocol itself
DEFAULT_BACKEND="${DEFAULT_BACKEND#http://}"
DEFAULT_BACKEND="${DEFAULT_BACKEND#https://}"
```

确保 `DEFAULT_BACKEND` 始终是纯 `host:port` 格式，不论来源（builtin 覆盖、.setup.conf 历史值、external 模式用户输入）。

---

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `deploy/scripts/lib/render.sh` | Gateway 模板添加 `environment: NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__` |
| `deploy/scripts/lib/render.sh` | selfsigned CA 注入改为在已有 `environment:` 块内追加，不再新建重复块 |
| `deploy/scripts/lib/render.sh` | `DEFAULT_BACKEND` strip `http://`/`https://` 前缀 |
| `deploy/setup.sh` | `resume)` case：加载全部模块 → `load_saved_config` → `deploy_services()` → `show_result` |
| `deploy/setup.sh` | help 文本：`--resume` 移到 Setup Phases 区，描述更新 |
| `deploy/scripts/lib/deploy.sh` | 删除 `resume_deploy()` 函数（78 行） |
| `deploy/scripts/lib/deploy.sh` | `deploy_services()` 开头无条件 `export KUBECONFIG` |
| `deploy/scripts/lib/k8s-components.sh` | `configure_coredns()` 补 export `DOMAIN`/`LAUNCHPAD_DOMAIN`/`GITEA_DOMAIN` |
| `tests/cases/02-service-health.sh` | 新增 NODE_TLS_REJECT_UNAUTHORIZED 断言（api + gateway） |
| `tests/lib/helpers.sh` | `assert_healthy` compose 路径修复：`generated/` → `deploy/generated/` |

## 验证

1. E2E `02-service-health.sh` — NODE_TLS_REJECT 断言 PASS（api + gateway）
2. `--resume` 在 k3s 已运行的服务器上成功执行完整 deploy 流程
3. CoreDNS 正常运行，`coredns-custom` 包含正确的域名映射
4. Gateway compose 文件：单一 `environment:` 块，含 `NODE_EXTRA_CA_CERTS` + `NODE_TLS_REJECT_UNAUTHORIZED`
5. `DEFAULT_BACKEND=10.233.201.133:30080`（无协议前缀）
6. Router 无 `ENOTFOUND` 错误，gateway 启动正常
