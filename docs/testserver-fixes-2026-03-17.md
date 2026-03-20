# AniLaunchpad 测试服务器问题修复记录

**服务器**: root@10.233.201.133 (4C/7.4G, Debian 12, k3s)
**域名**: testclaw.com / launchpad.corp.testclaw.com
**日期**: 2026-03-17

---

## 问题列表与修复

### 1. Gateway 数据库表缺失

**现象**: 提交 task 后 API 返回 502，"No ani-code instance registered for this project"

**原因**: Gateway 使用 Prisma，数据库 schema `launchpad_gateway` 内无任何表

**修复**:
```bash
docker exec generated-gateway-1 sh -c "cd /app/api && npx prisma db push"
```

---

### 2. CoreDNS 崩溃 (loop detection + OOMKill)

**现象**: k8s pod 内无法解析任何域名

**原因**:
- `/etc/resolv.conf` 指向 `127.0.0.1` (dnsmasq)，CoreDNS `forward . /etc/resolv.conf` 形成回环，`loop` 插件检测到后杀死进程
- 增加自定义 zone 后内存超出默认 170Mi 限制，OOMKill

**修复**: 修改 CoreDNS ConfigMap `coredns` (kube-system):
```yaml
# 修改前
forward . /etc/resolv.conf
# 修改后 — 移除 loop 插件，直接转发到公共 DNS
forward . 223.5.5.5 8.8.8.8
```

增加内存限制:
```bash
kubectl edit deploy coredns -n kube-system
# resources.limits.memory: 170Mi → 512Mi
```

---

### 3. k8s Pod 无法访问 Docker Compose 服务 (端口 80/443/2222)

**现象**: Pod 内 `wget http://launchpad.corp.testclaw.com` 失败

**原因**: k8s pod 网络 (10.42.0.0/16) 无法直接访问宿主机上 Docker 监听的端口，Docker 的 DNAT 规则与 k8s 路由冲突

**修复**: 添加 iptables REDIRECT 规则:
```bash
# k8s pods → Docker nginx (HTTP/HTTPS)
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 80 -j REDIRECT --to-port 80
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 443 -j REDIRECT --to-port 443

# k8s pods → Docker gitea SSH
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 2222 -j REDIRECT --to-port 2222

# Docker containers → k8s ingress-nginx (端口 80 → NodePort 32009)
iptables -t nat -I PREROUTING -s 172.19.0.0/16 -d 10.233.201.133 -p tcp --dport 80 -j REDIRECT --to-port 32009
```

> **注意**: 这些规则不持久化，重启后丢失。

---

### 4. k8s Pod 无法访问 Gateway 服务 (端口 6555)

**现象**: Pod 内 `curl http://gateway:6555` 返回 ECONNREFUSED

**原因**: 三层问题叠加:
1. **DNS**: CoreDNS 不知道 `gateway` 这个主机名
2. **iptables**: REDIRECT 对 docker-proxy 端口不生效，需要 DNAT + MASQUERADE
3. **NetworkPolicy**: kube-router 网络策略只允许 egress 到端口 443/80/22，阻断 6555
4. **FORWARD chain**: policy DROP，缺少 k8s→Docker 跨网络转发规则

**修复**:

**(a) CoreDNS — 添加 gateway DNS 解析** (`coredns-custom` ConfigMap):
```yaml
gateway.server: |
  gateway:53 {
      hosts {
          10.233.201.133 gateway
          ttl 60
      }
  }
```

**(b) iptables — DNAT + MASQUERADE + FORWARD**:
```bash
# PREROUTING: k8s pod 流量 DNAT 到 gateway 容器 Docker IP
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 6555 \
  -j DNAT --to-destination 172.19.0.9:6555

# POSTROUTING: MASQUERADE 确保返回包正确路由
iptables -t nat -A POSTROUTING -s 10.42.0.0/16 -d 172.19.0.9 -p tcp --dport 6555 \
  -j MASQUERADE

# FORWARD: 允许跨网络转发 (插在 KUBE-ROUTER-FORWARD 之后)
iptables -I FORWARD -s 10.42.0.0/16 -d 172.19.0.9 -p tcp --dport 6555 -j ACCEPT
iptables -I FORWARD -s 172.19.0.9 -d 10.42.0.0/16 -p tcp --sport 6555 -j ACCEPT
```

> **注意**: `172.19.0.9` 是 gateway 容器的 Docker IP，容器重建后可能变化。

**(c) NetworkPolicy — 允许 egress 到端口 6555**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-gateway-egress
  namespace: user-admin
  labels:
    anilaunchpad.io/policy-type: custom-egress
    app.kubernetes.io/managed-by: deploy-setup
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - ports:
    - port: 6555
      protocol: TCP
    to:
    - ipBlock:
        cidr: 10.0.0.0/8
    - ipBlock:
        cidr: 172.16.0.0/12
```

> **注意**: 每个新建的 user namespace 都需要创建此 NetworkPolicy。

---

### 5. Pod Bootstrap 脚本下载失败 (HTTP 404)

**现象**: 新建项目 pod 启动后报 `bootstrap.sh: 1: cannot open html: No such file`

**原因**: Pod 下载 `http://launchpad.corp.testclaw.com/api/bootstrap/pod-bootstrap-v2.sh` 时，DNS 将 `launchpad.corp.testclaw.com` 解析到 `10.43.180.235` (ingress-nginx ClusterIP)，但 ingress-nginx 没有该域名的路由规则，返回 404 HTML。Pod 将 HTML 当做 shell 脚本执行。

**根因**: CoreDNS 配置中，wildcard 模板 `.*\.testclaw\.com\.$` 覆盖了 hosts 插件中 `launchpad.corp.testclaw.com` 的显式记录 (10.233.201.133)，导致返回 10.43.180.235。

**修复**: 拆分 CoreDNS 自定义配置为三个独立 server block (`coredns-custom` ConfigMap):

```yaml
# 1. launchpad/gitea 特定域名 → 宿主机 IP (Docker nginx)
launchpad-specific.server: |
  launchpad.corp.testclaw.com:53 launchpad-gitea.corp.testclaw.com:53 gitea:53 {
      hosts {
          10.233.201.133 launchpad.corp.testclaw.com
          10.233.201.133 launchpad-gitea.corp.testclaw.com
          10.233.201.133 gitea
          ttl 60
      }
  }

# 2. 通配符子域名 → ingress-nginx ClusterIP (项目 pod)
wildcard-testclaw.server: |
  testclaw.com:53 {
      hosts {
          10.43.180.235 testclaw.com
          ttl 60
          fallthrough
      }
      template IN A {
          match ".*\.testclaw\.com\.$"
          answer "{{ .Name }} 60 IN A 10.43.180.235"
      }
  }

# 3. gateway 主机名 → 宿主机 IP
gateway.server: |
  gateway:53 {
      hosts {
          10.233.201.133 gateway
          ttl 60
      }
  }
```

应用后重启 CoreDNS:
```bash
kubectl apply -f coredns-custom.yaml
kubectl rollout restart deploy/coredns -n kube-system
```

---

### 6. Gateway SSL 证书验证失败

**现象**: Gateway 健康检查失败 5 次后移除 daemon 实例，前端显示 "No response from gateway"

**原因**: Gateway 回调 daemon 时访问 `https://stellar-creek-520-ad620c.testclaw.com/.ani-code-callback`，SSL 证书是自签名的，Node.js 默认拒绝

**修复（已过时 — 见下方）**: ~~在 `deploy/generated/gateway/.env` 添加 `NODE_TLS_REJECT_UNAUTHORIZED=0`~~

> **已被 2026-03-19 修复取代**: render.sh 的 gateway compose 模板已添加 `NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__`，render 时自动根据 SSL_MODE 设置正确的值。手动修改 .env 不再需要。详见 `2026-03-19-gateway-tls-resume-coredns-fixes.md` #1。

---

### 7. Gateway URL 不匹配导致 Daemon 拒绝消息

**现象**: 提交 task 后 daemon 日志显示 `Gateway trust bypass FAILED`，`Unauthorized access from user web-user`

**原因**: Gateway 转发请求时，`x-gateway-url` header 设为 `GATEWAY_PUBLIC_URL` 的值 (`https://launchpad.corp.testclaw.com/gatewayproxy`)，但 daemon 期望的是 `GATEWAY_URL=http://gateway:6555`，URL 不匹配导致信任验证失败

**修复**: 修改 `deploy/generated/gateway/.env`:
```
# 修改前
GATEWAY_PUBLIC_URL=https://launchpad.corp.testclaw.com/gatewayproxy
# 修改后
GATEWAY_PUBLIC_URL=http://gateway:6555
```

然后重建容器:
```bash
cd deploy/generated && docker compose up -d gateway --force-recreate
```

---

### 8. Show Bootstrap 按钮缺失

**现象**: 项目页面没有 "Show Bootstrap" 按钮

**原因**: 缺少环境变量 `EXP_BOOTSTRAP_DEBUG=true`

**修复**: 在 `deploy/generated/launchpad/.env` 添加:
```
EXP_BOOTSTRAP_DEBUG=true
```

---

### 9. k3s 集群状态显示 Unhealthy + 心跳失败

**现象**: Admin 页面集群详情显示 Unhealthy，报错 "Cluster has no kubeconfig stored"

**原因**: 两个叠加问题：
1. 部署脚本自建了 `setup_heartbeat()` 函数，为获取 heartbeat token 向 `/k3s/register` 发送了一次 `"kubeconfig":""` 的请求，**覆盖**了 `register.sh` 之前存储的完整 kubeconfig
2. 心跳采用 Docker 容器常驻模式运行，而 `register.sh` 原生已通过宿主机 cron 实现心跳，完全重复

**修复**: 删除自建心跳逻辑，完全依赖 register.sh 原生流程：

**(a) 删除 `setup_heartbeat()` 函数** (`deploy/scripts/lib/k3s.sh`):
- 移除整个函数（~65 行），包括二次注册、下载心跳脚本、生成容器 kubeconfig、启动心跳容器的全部逻辑

**(b) 删除 Docker Compose 中的 heartbeat 服务** (`deploy/scripts/lib/render.sh`):
- 移除 `heartbeat:` 服务定义（bitnami/kubectl 容器）

**(c) 简化 `register_cluster()`**:
- 移除对 `setup_heartbeat()` 的调用
- register.sh 自己完成：注册 → 下载 heartbeat.sh → 写配置 → 注册 cron → 测试心跳

**(d) 补充 register.sh 的两个小缺陷**:
- selfsigned 模式下，注册完成后 patch `/usr/local/bin/k3s-heartbeat.sh` 加 `-k` 跳过证书验证
- 给 `/etc/default/k3s-heartbeat` 追加 `PATH` 和 `KUBECONFIG`，因为 cron 环境缺少这些变量

**register.sh 原生心跳机制**:
```
register.sh 注册成功后自动执行:
├── 下载 heartbeat.sh → /usr/local/bin/k3s-heartbeat.sh
├── 写配置 → /etc/default/k3s-heartbeat
│   ├── K3S_REGISTRATION_SECRET=<token>
│   ├── LAUNCHPAD_API_URL=<url>
│   ├── K3S_HOSTNAME=<hostname>
│   ├── PATH=/usr/local/bin:/usr/bin:/bin       ← 部署脚本补充
│   └── KUBECONFIG=/etc/rancher/k3s/k3s.yaml   ← 部署脚本补充
├── 注册 cron (每分钟):
│   * * * * * . /etc/default/k3s-heartbeat && RUN_ONCE=true /usr/local/bin/k3s-heartbeat.sh
└── 测试心跳（10s timeout）
```

---

## 完整配置变更清单

### CoreDNS ConfigMap `coredns` (kube-system)

| 项目 | 修改前 | 修改后 |
|------|--------|--------|
| loop 插件 | 启用 | **删除** |
| forward 目标 | `/etc/resolv.conf` | `223.5.5.5 8.8.8.8` |
| 内存限制 | 170Mi | **512Mi** |

### CoreDNS ConfigMap `coredns-custom` (kube-system)

**新建**，包含 3 个 server block:
- `gateway.server` — `gateway` → 10.233.201.133
- `launchpad-specific.server` — `launchpad.corp.testclaw.com` / `launchpad-gitea.corp.testclaw.com` / `gitea` → 10.233.201.133
- `wildcard-testclaw.server` — `*.testclaw.com` → 10.43.180.235 (ingress-nginx ClusterIP)

### iptables 规则 (非持久化)

```bash
# NAT PREROUTING
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 6555 -j DNAT --to-destination 172.19.0.9:6555
iptables -t nat -I PREROUTING -s 172.19.0.0/16 -d 10.233.201.133 -p tcp --dport 80 -j DNAT --to 127.0.0.1:32009
iptables -t nat -I PREROUTING -s 172.19.0.0/16 -d 10.233.201.133 -p tcp --dport 80 -j REDIRECT --to-port 32009
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 2222 -j REDIRECT --to-port 2222
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 443 -j REDIRECT --to-port 443
iptables -t nat -I PREROUTING -s 10.42.0.0/16 -d 10.233.201.133 -p tcp --dport 80 -j REDIRECT --to-port 80

# NAT POSTROUTING
iptables -t nat -A POSTROUTING -s 10.42.0.0/16 -d 172.19.0.9 -p tcp --dport 6555 -j MASQUERADE

# FORWARD
iptables -I FORWARD -s 10.42.0.0/16 -d 172.19.0.9 -p tcp --dport 6555 -j ACCEPT
iptables -I FORWARD -s 172.19.0.9 -d 10.42.0.0/16 -p tcp --sport 6555 -j ACCEPT
```

### NetworkPolicy (user-admin namespace)

**新建** `allow-gateway-egress` — 允许 pod egress 到端口 6555 (10.0.0.0/8 + 172.16.0.0/12)

### Gateway 环境变量 (`deploy/generated/gateway/.env`)

| 变量 | 修改前 | 修改后 |
|------|--------|--------|
| GATEWAY_PUBLIC_URL | `https://launchpad.corp.testclaw.com/gatewayproxy` | `http://gateway:6555` |
| NODE_TLS_REJECT_UNAUTHORIZED | (未设置) | `0` |

### Launchpad 环境变量 (`deploy/generated/launchpad/.env`)

| 变量 | 修改前 | 修改后 |
|------|--------|--------|
| EXP_BOOTSTRAP_DEBUG | (未设置) | `true` |

### Docker Compose 模板 (`deploy/scripts/lib/render.sh`)

- Gateway 服务添加端口映射 `6555:6555`
- **删除** heartbeat 服务定义（改用 register.sh 原生 cron 心跳）

### 集群注册 (`deploy/scripts/lib/k3s.sh`)

- **删除** `setup_heartbeat()` 函数（~65 行）
- `register_cluster()` 不再二次注册，完全依赖 register.sh
- 注册后补充：selfsigned 模式 patch heartbeat.sh 加 `-k`；给 cron 环境追加 PATH 和 KUBECONFIG

---

### 10. Gateway 访问架构优化 — 消除 NetworkPolicy 和 iptables 6555 依赖

**现象**: 问题 4 和 7 的修复方案相互矛盾：
- 问题 7 将 `GATEWAY_PUBLIC_URL` 改为 `http://gateway:6555`，要求 daemon 直连 6555
- 问题 4 为此添加了 NetworkPolicy (egress 6555)、iptables DNAT/MASQUERADE、CoreDNS gateway 记录

**根因**: daemon pod 应该通过公共 URL (443端口) 访问 gateway，而不是直连内部端口 6555。使用外部 k8s 集群时，走 nginx 443 是标准路径。

**修复**: 修改 `deploy/scripts/lib/render.sh`，让 `ANI_CODE_GATEWAY_URL` 默认使用 `GATEWAY_PUBLIC_URL`:
```bash
# 修改前
export ANI_CODE_GATEWAY_URL="${ANI_CODE_GATEWAY_URL:-http://gateway:6555}"
# 修改后
export ANI_CODE_GATEWAY_URL="${ANI_CODE_GATEWAY_URL:-${GATEWAY_PUBLIC_URL}}"
```

同时将 `GATEWAY_PUBLIC_URL` 恢复为公共 URL（撤销问题 7 的修改）:
```
GATEWAY_PUBLIC_URL=https://${LAUNCHPAD_DOMAIN}/gatewayproxy
```
（render.sh 中已正确设置，无需额外修改）

**效果**:
- daemon 的 `GATEWAY_URL` = `https://launchpad.corp.testclaw.com/gatewayproxy` (443端口，已有 egress 放行)
- gateway 的 `x-gateway-url` header = 同上 → 信任校验通过
- 不再需要: `allow-gateway-egress` NetworkPolicy、6555 端口 iptables 规则、CoreDNS `gateway` 记录
- 自签名证书环境需确保 daemon pod 设置 `NODE_TLS_REJECT_UNAUTHORIZED=0`

**流量路径**:
```
daemon (k8s pod) → 443 → nginx → gateway:6555 (Docker 内部)
```

---

## 已知遗留问题

1. **iptables 规则不持久化** — 服务器重启后需要重新添加所有 iptables 规则
2. **Gateway Docker IP 不固定** — 容器重建后 IP 可能变化 (当前 172.19.0.9)，DNAT 规则需要更新
3. ~~**NetworkPolicy 需要逐 namespace 创建**~~ — 已通过问题 10 解决，daemon 走 443 不再需要 6555 egress 策略
4. **SSH 隧道不持久化** — SOCKS5 代理隧道 (`ssh -D 1081`) 断开后需要手动重连
5. **Claude CLI 需要 API Key** — Pod 内 `claude` 命令需要 `ANTHROPIC_API_KEY` 环境变量，需在项目 Settings/Credentials 中配置
