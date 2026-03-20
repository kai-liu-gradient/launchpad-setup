# dnsmasq 实现 Docker 容器到 k8s Pod 的域名路由

日期: 2026-03-18

## 背景

替换 Traefik 为 ingress-nginx 后，泛域名 `*.testclaw.com` 的完整流量链路为：

```
外部流量 → nginx(:443 wildcard) → router(:6580) → ingress-nginx → pod
```

Docker 容器（router、gateway）需要访问 `*.testclaw.com` 域名。

### 问题

Docker 容器解析 `*.testclaw.com` 需要到达正确的目标：
- **Gateway** 回调 daemon 走 HTTPS:443，需要到 nginx（获取正确的通配符证书）
- **Router** 从 API 拿到 backend 域名走 HTTP:80

### 关键发现

1. Router 有 fallback 机制：96% 请求走 `DEFAULT_BACKEND`（`http://宿主IP:30080` → ingress-nginx），仅 4% 走 backend 域名
2. Gateway 需要走 `nginx:443` 才能拿到正确的通配符 TLS 证书（直连 ingress-nginx ClusterIP:443 会拿到 fake cert）
3. dnsmasq 解析到宿主 IP 时，router 通过 backend 域名访问 `宿主IP:80` 会被 nginx 返回 301，自动 fallback 到 `DEFAULT_BACKEND`
4. 性能影响：301 fallback 比 ClusterIP 直连多 ~20ms，但只影响 4% 的请求，整体影响 <1ms

## 方案

dnsmasq 将 `*.DOMAIN` 解析到**宿主 IP**（不是 ClusterIP）：

```conf
# /etc/dnsmasq.d/launchpad.conf
address=/testclaw.com/10.233.201.133    # 宿主 IP
server=223.5.5.5
server=8.8.8.8
```

流量链路：
- **Gateway** → `宿主IP:443` → nginx(通配符证书) → router → ingress-nginx → pod ✓
- **Router** (system path) → `DEFAULT_BACKEND(宿主IP:30080)` → ingress-nginx → pod ✓
- **Router** (backend domain) → `宿主IP:80` → nginx 301 → fallback `DEFAULT_BACKEND` ✓

## 改动

### 1. detect.sh — 自动安装 dnsmasq

builtin 模式下，检测 dnsmasq 是否存在，没有则自动安装。兼容多发行版：

- Debian/Ubuntu: `apt-get install dnsmasq`
- CentOS/RHEL: `yum install dnsmasq`
- Rocky/Fedora: `dnf install dnsmasq`

### 2. k8s-components.sh — configure_coredns() 末尾配置 dnsmasq

在 CoreDNS 配置完成后（此时已拿到 ingress-nginx ClusterIP），写入 `/etc/dnsmasq.d/launchpad.conf`：

```bash
cat > /etc/dnsmasq.d/launchpad.conf << EOF
address=/${DOMAIN}/${cluster_ip}
server=223.5.5.5
server=8.8.8.8
EOF
systemctl restart dnsmasq
```

幂等：每次部署都会用最新的 ClusterIP 覆盖，解决 ingress-nginx 重装后 ClusterIP 变化的问题。

> **2026-03-19 修复**: `configure_coredns()` 的 `envsubst` 调用缺少 `DOMAIN`/`LAUNCHPAD_DOMAIN`/`GITEA_DOMAIN` 的 export，导致 `--resume` 时渲染为空、CoreDNS CrashLoop。已补上 export。详见 `2026-03-19-gateway-tls-resume-coredns-fixes.md` #4。

### 3. deploy.sh — uninstall_all() 清理 dnsmasq 配置

卸载时删除 `/etc/dnsmasq.d/launchpad.conf` 并重启 dnsmasq。不卸载 dnsmasq 本身（可能是用户预装的）。

### 4. kyverno-inject-ca.yaml — 恢复 NODE_OPTIONS

精简 kyverno 注入策略时删掉了 `NODE_OPTIONS=--use-system-ca`，但 pod 内跑的是 Node.js 应用，Node.js 不读系统 CA store，需要这个 flag。加回来（对非 Node.js 容器无害）。

### 5. tests/cases/06-cluster-register.sh — dnsmasq 断言

新增两个测试：
- 检查 `/etc/dnsmasq.d/launchpad.conf` 包含 `address=/${DOMAIN}/`
- 从 kubectl 获取 ingress-nginx ClusterIP，与 dnsmasq 实际解析结果精确匹配

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `deploy/scripts/lib/detect.sh` | builtin 模式下自动安装 dnsmasq（apt/yum/dnf） |
| `deploy/scripts/lib/k8s-components.sh` | `configure_coredns()` 末尾写入 dnsmasq 配置 |
| `deploy/scripts/lib/deploy.sh` | `uninstall_all()` 删除 dnsmasq 配置 |
| `deploy/templates/kyverno-inject-ca.yaml` | profile.d 挂载 + `NODE_EXTRA_CA_CERTS` env（覆盖 `su -` 场景） |
| `deploy/scripts/lib/render.sh` | selfsigned 模式下 gateway 容器挂载 CA + `NODE_EXTRA_CA_CERTS` |
| `deploy/scripts/lib/deploy.sh` | 数据库迁移阶段新增 gateway `npm run db:push` |
| `tests/cases/06-cluster-register.sh` | 新增 dnsmasq DNS 解析断言 |
| `tests/lib/helpers.sh` | `sync_files()` 修复：同步根目录 `scripts/`、`lang/`、`templates/` |

## 流量链路（最终）

```
外部浏览器 → nginx(:443) → router(:6580) → DEFAULT_BACKEND(:30080) → ingress-nginx → pod
Gateway    → nginx(:443) → router(:6580) → DEFAULT_BACKEND(:30080) → ingress-nginx → pod
```

- 不改应用代码
- 不开 iptables
- 所有流量统一走 nginx → router → ingress-nginx 链路

### 6. render.sh — gateway 容器 CA 信任

Gateway 回调 daemon 走 `https://xxx.testclaw.com` → nginx:443（通配符证书，由自签名 CA 签发）。

在 `render_compose()` 中，selfsigned 模式下给 gateway 容器：
- 挂载 `generated/nginx/certs/ca.pem` 到 `/etc/ssl/certs/launchpad-ca.pem:ro`
- 设置 `NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem`

> **2026-03-19 修复**: 原实现在 `gateway:` 行后新建 `environment:` 块注入 `NODE_EXTRA_CA_CERTS`，但 gateway 模板已有 `environment:` 块（含 `NODE_TLS_REJECT_UNAUTHORIZED`），导致 YAML 重复 key 报错。已改为在已有块内追加。详见 `2026-03-19-gateway-tls-resume-coredns-fixes.md` #5。

### 7. deploy.sh — gateway 数据库迁移

Gateway 有独立的 Prisma schema（`launchpad_gateway` schema，含 `gateway_tokens` 等表），但部署时未执行迁移。

在数据库迁移阶段末尾新增：
```bash
$COMPOSE_CMD run --rm gateway npm run db:push
```

## 验证

E2E 测试全部通过（含 dnsmasq 断言）。
