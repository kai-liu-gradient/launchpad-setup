# nginx 项目域名路由修复 — 循环重定向 + vibe iframe 加载

日期: 2026-03-19

## 问题一：项目域名无限重定向 (ERR_TOO_MANY_REDIRECTS)

### 背景

访问项目域名 `zesty-relay-75-e91ec7.testclaw.com` 时，浏览器报 `ERR_TOO_MANY_REDIRECTS`。

### 根因分析

Router 从 API 获取集群级 backend 域名（`ecs-dev-liukai-backend.testclaw.com`），用它作为**连接地址**，但 **Host 头保留原始项目域名**。

来自 Router 源码 `/app/core/router.js` 的 `buildBackendHeaders()`：

```javascript
// CRITICAL: Set Host header to original hostname for backend ingress routing
headers['Host'] = hostname;  // 原始项目域名，不是 backend 域名
```

#### 循环路径

```
浏览器 → zesty-relay-75-e91ec7.testclaw.com (HTTPS:443)
  → nginx:443 → *.testclaw.com server block → Router (:6580)
  → Router 查 API → backend: ecs-dev-liukai-backend.testclaw.com
  → Router 连接 ecs-dev-liukai-backend.testclaw.com:80
    (Host 头: zesty-relay-75-e91ec7.testclaw.com)
  → dnsmasq → 宿主机 IP → nginx:80
  → 没有匹配的 server block → default server → 301 → HTTPS
  → nginx:443 → *.testclaw.com → Router → ...
  🔁 无限循环
```

关键发现：最初尝试用 `server_name ~^.+-backend\.DOMAIN$` 匹配 backend 域名，但这**不可能生效**——nginx 按 Host 头匹配 server_name，而 Router 发送的 Host 头是原始项目域名，不是 backend 域名。

### 修复方案

在 port 80 添加 `*.DOMAIN` wildcard server block，直接转发到 ingress-nginx NodePort。

Router 的请求到达 nginx:80 时 Host 头是 `zesty-relay-75-e91ec7.testclaw.com`，能被 `*.testclaw.com` 匹配。直接转发到 ingress-nginx:30080，ingress-nginx 用同一个 Host 头匹配 Ingress 资源 → Pod → 200。

不会影响其他域名：
- `launchpad.corp.testclaw.com` — 两级子域名，不匹配 `*.testclaw.com`
- `launchpad-gitea.corp.testclaw.com` — 同上
- 浏览器通过 SSH 隧道走 HTTPS:443，不走 HTTP:80

#### 修复后流量路径

```
浏览器 → zesty-relay-75-e91ec7.testclaw.com (HTTPS:443)
  → SSH 隧道 → 宿主机 → nginx:443
  → *.testclaw.com server block → Router (:6580)
  → Router 查 API → backend: ecs-dev-liukai-backend.testclaw.com
  → Router 连接 ecs-dev-liukai-backend.testclaw.com:80
    (Host 头: zesty-relay-75-e91ec7.testclaw.com)
  → dnsmasq → 宿主机 IP → nginx:80
  → *.testclaw.com server block (port 80) 匹配 ✓
  → proxy_pass ingress-nginx:30080
  → Ingress 匹配 Host: zesty-relay-75-e91ec7.testclaw.com → Pod
  → 200 ✓
```

---

## 问题二：vibe 模式 iframe 加载失败

### 背景

Dashboard 的 vibe 模式（`https://launchpad.corp.testclaw.com/project/.../vibe`）通过 iframe 嵌入项目的公网 URL（`https://zesty-relay-75-e91ec7.testclaw.com/`）。直接访问项目 URL 正常返回 200，但 iframe 中 response 为空。

### 根因

nginx 的 `*.DOMAIN` server block（port 443）设置了 `X-Frame-Options: SAMEORIGIN`。Dashboard 域名 `launchpad.corp.testclaw.com` 与项目域名 `zesty-relay-75-e91ec7.testclaw.com` 不同源，浏览器拒绝在 iframe 中加载。

### 修复方案

将 `X-Frame-Options: SAMEORIGIN` 替换为 CSP `frame-ancestors`，允许 dashboard 域名嵌入。

`frame-ancestors` 比 `X-Frame-Options` 更灵活：可以指定多个允许的来源，且是 CSP Level 2 标准。

---

## 改动

### 1. nginx.conf.template — port 80 wildcard server block

替换原有的 `~^.+-backend` 正则 server block：

```nginx
# 4. Internal HTTP proxy (port 80) — Docker containers → ingress-nginx
# Router connects to backend domain on port 80 with the original project
# Host header preserved. Forward to ingress-nginx which matches the Host.
server {
    listen 80;
    server_name *.${DOMAIN};

    location / {
        proxy_pass http://${HOST_IP}:30080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_connect_timeout 60s;
        proxy_send_timeout 86400s;
        proxy_read_timeout 86400s;
    }
}
```

default server block 需要 `listen 80 default_server` 以确保不匹配 wildcard 的请求（如健康检查）落到正确的 block。

### 2. nginx.conf.template — frame-ancestors 替换 X-Frame-Options

`*.DOMAIN` server block（port 443）中：

```diff
- add_header X-Frame-Options "SAMEORIGIN" always;
+ add_header Content-Security-Policy "frame-ancestors 'self' https://${LAUNCHPAD_DOMAIN}" always;
```

允许 `'self'`（同域名）和 dashboard 域名嵌入项目 iframe。

### 3. render.sh — 导出 HOST_IP 变量

复用已有的 `detect_internal_ip`，导出 `HOST_IP` 并加入 envsubst 变量列表：

```bash
if [[ "${K8S_MODE:-}" == "builtin" ]]; then
    host_ip=$(detect_internal_ip 2>/dev/null || echo "127.0.0.1")
    export DEFAULT_BACKEND="http://${host_ip}:30080"
    export HOST_IP="${host_ip}"
else
    export HOST_IP="127.0.0.1"
fi

envsubst '${DOMAIN} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN} ${HOST_IP}' < ...
```

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `deploy/templates/nginx.conf.template` | block #4: `*-backend` 正则 → `*.DOMAIN` wildcard on port 80 |
| `deploy/templates/nginx.conf.template` | block #3 + #1: `X-Frame-Options` → `Content-Security-Policy: frame-ancestors` |
| `deploy/templates/nginx.conf.template` | block #5: `listen 80` → `listen 80 default_server` |
| `deploy/scripts/lib/render.sh` | 导出 `HOST_IP`，加入 envsubst 变量列表 |

## 验证

1. `curl -sI http://宿主机:80/ -H 'Host: project.testclaw.com'` → 200（非 301）
2. `curl -ksI https://宿主机/ -H 'Host: project.testclaw.com' | grep frame` → `frame-ancestors 'self' https://launchpad.corp.testclaw.com`
3. `curl -s http://127.0.0.1:80/nginx-health` → `healthy`
4. 浏览器访问项目域名 → 正常显示（无重定向循环）
5. 浏览器访问 vibe 模式 → iframe 正常加载项目预览
6. Dashboard 不受影响
