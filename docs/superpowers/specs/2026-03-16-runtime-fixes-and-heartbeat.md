# 部署脚本运行时修复与心跳服务设计补丁

> 本文档记录 2026-03-16 实际部署测试中发现的问题及修复方案，作为原设计文档 `2026-03-15-one-click-deploy-design-zh.md` 的补丁。

## 1. Nginx 健康检查修复

**问题：** Nginx 容器内 `localhost` 解析为 IPv6 `::1`，但 nginx 仅监听 IPv4 `0.0.0.0`，导致健康检查超时。同时 Docker healthcheck interval=15s 与 wait timeout=15s 产生竞态。

**修复：**
- 健康检查命令使用 `127.0.0.1` 替代 `localhost`
- 使用 `wget` 替代 `curl`（alpine 镜像自带）
- 增加 `start_period: 5s`，wait timeout 增至 30s

```yaml
healthcheck:
  test: ["CMD-SHELL", "nginx -t && wget -q --spider http://127.0.0.1:80/nginx-health || exit 1"]
  start_period: 5s
```

## 2. Docker 内部服务连通性

**问题：** API 容器的环境变量使用外部域名 URL（如 `https://launchpad.corp.testclaw.com/...`），容器内解析后指向自身 `127.0.0.1`，导致 Router、Gateway、Gitea 连接失败。

**修复：**
- 内部服务 URL 改为 Docker DNS 地址，支持外部覆盖：
  ```bash
  export ROUTER_LOCAL_URL="${ROUTER_LOCAL_URL:-http://router:6580}"
  export ANI_CODE_GATEWAY_URL="${ANI_CODE_GATEWAY_URL:-http://gateway:6555}"
  export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@gitea:2222}"
  ```
- Nginx 容器添加网络别名，使域名在 Docker 网络内可解析：
  ```yaml
  networks:
    launchpad-network:
      aliases:
        - launchpad.corp.testclaw.com
        - launchpad-gitea.corp.testclaw.com
  ```

## 3. 自签名证书 TLS 处理

**问题：** Node.js 默认拒绝自签名证书，容器间通过 HTTPS 互访失败。

**修复：**
- `SSL_MODE=selfsigned` 时，API 容器注入 `NODE_TLS_REJECT_UNAUTHORIZED=0`
- 自签名证书直接复制为泛域名证书（不用 symlink，Docker 跨设备挂载时 symlink 失效）

## 4. Cron 容器修复

**问题：** Cron 容器命令 `cp /app/pd/deploy/crontab /etc/crontabs/root` 失败——容器内不存在该路径。

**修复：**
- 参照 `old/launchpad/` 的配置，将 crontab 文件作为 volume 只读挂载：
  ```yaml
  volumes:
    - ./launchpad/cron/crontab:/etc/crontabs/root:ro
    - cron-logs:/var/log/cron
  ```
- 新增文件：`deploy/templates/crontab`（从 `old/launchpad/pd/deploy/crontab` 复制）
- `render_templates()` 自动将 crontab 复制到 `generated/launchpad/cron/`

## 5. 静默模式与进度条

**问题：** 部署过程输出大量信息（数据库初始化、docker pull 等），用户看不到进度。

**修复：**
- 新增 `--verbose|-v` 标志，默认静默模式
- `log_info()`/`log_ok()` 仅在 verbose 模式可见
- 新增 `log_done()` 始终可见，`verbose_filter()` 管道函数
- 新增进度条：`progress_start N` / `progress_update "msg"` / `progress_done`
  ```
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 100%  ✓ Done
  ```

## 6. `--resume` 命令修复

**问题：** `--resume` 路径未加载语言包，导致 `MSG_DEPLOY_DB_INIT: unbound variable` 崩溃。集群注册失败时 `set -euo pipefail` 导致整个脚本退出。

**修复：**
- `resume_deploy()` 增加 `source lang/${LANG_CHOICE}.sh`
- `save_config()` 保存 `LANG_CHOICE` 到 `.setup.conf`
- `register_cluster` 调用包装为 `|| log_warn`（非致命）

## 7. 副本数配置

**问题：** 高级配置只有 API 可选副本数，缺少 Router 和 Gateway。

**修复：**
- 高级配置增加 `ROUTER_REPLICAS`、`GATEWAY_REPLICAS` 选项
- Docker Compose 模板中 API、Router、Gateway 均支持 `deploy.replicas` + 滚动更新配置

## 8. K3s 集群注册修复

**问题：** `register.sh` 在 macOS 上多处不兼容：
- 检查 k3s 二进制文件/服务进程（colima 环境不存在）
- `detect_internal_ip` 使用 `ip` 命令（macOS 无此命令）
- curl POST 未使用 `-k`（自签名证书被拒）
- `head -n -1` 为 GNU 扩展（macOS BSD 不支持）

**修复：**
- 始终传递 `--skip-k3s-check`（k8s 连通性已在 `setup_kubernetes()` 中验证）
- `SSL_MODE=selfsigned` 时 sed 注入 `-k` 到 curl 命令
- macOS 上注入已检测的内部 IP 替换 `detect_internal_ip` 调用
- macOS 上 `head -n -1` 替换为 `sed '$ d'`

## 9. Gitea 环境变量修复

**问题：** 内置 DB 模式下 `GITEA_DB_HOST`、`GITEA_DB_PORT`、`GITEA_ROOT_URL` 为空（`DB_HOST`/`DB_PORT` 未设置），导致 YAML 解析失败。

**修复：**
- sed 替换增加嵌套默认值：
  ```bash
  -e "s|__GITEA_DB_HOST__|${GITEA_DB_HOST:-${DB_HOST:-postgres}}|g"
  -e "s|__GITEA_DB_PORT__|${GITEA_DB_PORT:-${DB_PORT:-5432}}|g"
  -e "s|__GITEA_ROOT_URL__|${GITEA_ROOT_URL:-https://${GITEA_DOMAIN}}|g"
  ```

## 10. Kubeconfig 路径修复

**问题：** `__KUBECONFIG_PATH__` 默认 `/etc/rancher/k3s/k3s.yaml`，macOS 上不存在。

**修复：**
```bash
-e "s|__KUBECONFIG_PATH__|${KUBECONFIG:-$([ -f /etc/rancher/k3s/k3s.yaml ] && echo /etc/rancher/k3s/k3s.yaml || echo $HOME/.kube/config)}|g"
```

## 11. 心跳服务（回归 register.sh 原生方案）

**原设计：** `register.sh` 内置心跳安装逻辑（写入 `/usr/local/bin`、设置系统级 cron）。

**曾经的问题与尝试：** 该方式在 macOS 上不可用（权限、`timeout` 缺失），因此曾改为 Docker Compose 容器方案。但实际测试中发现容器方案引入了新问题——`setup_heartbeat()` 为获取 heartbeat token 会二次注册并发送空 kubeconfig，覆盖了 register.sh 已存储的完整 kubeconfig，导致集群状态异常。

**当前方案：** 删除自建的 Docker 容器心跳，完全依赖 register.sh 原生的宿主机 cron 方案。

### register.sh 原生心跳机制

```
register.sh 注册成功后自动执行:
├── 从 API 下载 heartbeat.sh → /usr/local/bin/k3s-heartbeat.sh
├── 写配置 → /etc/default/k3s-heartbeat
│   ├── K3S_REGISTRATION_SECRET=<token>
│   ├── LAUNCHPAD_API_URL=<url>
│   └── K3S_HOSTNAME=<hostname>
├── 注册 cron (每分钟, RUN_ONCE=true):
│   * * * * * . /etc/default/k3s-heartbeat && RUN_ONCE=true /usr/local/bin/k3s-heartbeat.sh
└── 测试一次心跳（10s timeout）
```

### 部署脚本的补充处理

register.sh 原生逻辑有两个小缺陷，部署脚本在注册完成后补充修复：

1. **selfsigned 证书支持** — patch `/usr/local/bin/k3s-heartbeat.sh`，将 `curl -s ` 替换为 `curl -sk ` 跳过证书验证
2. **cron 环境变量** — 给 `/etc/default/k3s-heartbeat` 追加 `PATH=/usr/local/bin:/usr/bin:/bin` 和 `KUBECONFIG=<path>`，因为 cron 默认 PATH 不含 `/usr/local/bin`（kubectl 所在目录），且未设置 KUBECONFIG

### 变更内容

- **删除** `setup_heartbeat()` 函数（k3s.sh）
- **删除** Docker Compose heartbeat 服务定义（render.sh）
- **简化** `register_cluster()`：下载 register.sh → patch selfsigned → 执行 → 补充 heartbeat 环境

### 已知限制

- 宿主机 cron 方式在 macOS 上不可用（权限、`timeout` 命令缺失），macOS 仅用于开发，不影响实际部署
- 卸载时需手动清理 cron：`crontab -l | grep -v k3s-heartbeat | crontab -`

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `deploy/scripts/lib/common.sh` | 新增 VERBOSE、log_done、verbose_filter、进度条 |
| `deploy/scripts/lib/certs.sh` | 自签名证书 symlink → cp |
| `deploy/scripts/lib/render.sh` | Nginx 健康检查、网络别名、内部 URL 默认值、副本数、NODE_TLS_REJECT、Gitea 默认值、kubeconfig 路径、crontab 复制（删除 heartbeat 服务定义） |
| `deploy/scripts/lib/deploy.sh` | 静默模式、进度条、resume 修复、show_result K3s 状态 |
| `deploy/scripts/lib/k3s.sh` | register.sh macOS 兼容 patch、selfsigned 证书 patch、cron 环境变量补充（删除 setup_heartbeat） |
| `deploy/scripts/lib/database.sh` | verbose_filter |
| `deploy/scripts/lib/interact.sh` | 副本数配置、部署摘要完善 |
| `deploy/setup.sh` | --verbose 标志 |
| `deploy/templates/crontab` | 新增（从 old/ 复制） |
| `deploy/templates/nginx.conf.template` | `listen 443 ssl` + `http2 on;` |
| `deploy/templates/env.template` | 内部 URL 变量化 |
