# 优化 CA 注入策略 + Pod 连通性 E2E 测试

日期: 2026-03-18

## 背景

1. `kyverno-inject-ca.yaml` 用 `node:24`（~1GB）initContainer 在每个 pod 启动时合并 CA bundle，还注入了仅对 Node.js 有用的 `NODE_EXTRA_CA_CERTS`、`NODE_OPTIONS` 和 profile script。Pod 不一定是 Node.js，这些都多余。
2. E2E 测试缺少 pod 内域名连通性验证。
3. 实施过程中还发现并修复了两个独立问题：Traefik 残留和 CA 证书 V1 格式不兼容 OpenSSL 3.3+。

## 改动

### 1. kyverno-inject-ca.yaml — 从 70 行精简到 30 行

**删除：**
- initContainer `trust-ca`（`node:24` 镜像，~1GB）
- `launchpad-ca-cert` volume + 单独 PEM 挂载 `/etc/ssl/certs/launchpad-ca.pem`
- `ca-bundle` emptyDir volume
- 环境变量 `NODE_EXTRA_CA_CERTS` 和 `NODE_OPTIONS`
- profile script 挂载 `/etc/profile.d/launchpad-ca.sh`

**替换为：**
- volume `ca-bundle` 引用预合并的 `launchpad-ca-bundle` ConfigMap
- 主容器仅挂载 `ca-certificates.crt` 到 `/etc/ssl/certs/ca-certificates.crt`

### 2. k8s-components.sh — 预合并 CA bundle + 卸载 Traefik

**`setup_cert_distribution()`** 新增预合并逻辑：
```bash
cat /etc/ssl/certs/ca-certificates.crt "$ca_cert" > /tmp/ca-bundle.crt
kubectl create configmap launchpad-ca-bundle \
    --from-file=ca-certificates.crt=/tmp/ca-bundle.crt \
    -n kube-system --dry-run=client -o yaml | kubectl apply -f -
```

**`install_ingress_nginx()`** 新增 Traefik 卸载逻辑：
- k3s 的 `--disable traefik` 只对首次安装生效，已有集群不会自动卸载 Traefik
- 新增：检测 Traefik helm release 存在时，先 `helm uninstall traefik traefik-crd`，再删除 k3s manifests 防止重启恢复

### 3. kyverno-sync-ca.yaml — 新增 bundle 同步

新增 `sync-ca-bundle` 规则，将 `launchpad-ca-bundle` ConfigMap 从 kube-system 同步到所有 namespace。

### 4. certs.sh — CA 证书升级为 V3 格式

**问题：** OpenSSL 3.3+（alpine 3.21 的默认版本）对 V1 CA 证书 + RSA-PSS 签名（TLS 1.3 默认）验证报 `certificate signature failure (7)`。

**修复：** CA 证书生成加 V3 扩展：
```bash
openssl req -new -x509 -days 3650 -key ca.key -out ca.pem \
    -subj "/CN=AniLaunchpad Local CA" \
    -addext "basicConstraints=critical,CA:TRUE" \
    -addext "keyUsage=critical,keyCertSign,cRLSign"
```

### 5. E2E 新增 `tests/cases/09-pod-connectivity.sh`

在 k8s 集群中创建临时 pod（`node:24-alpine`），以 `node` 用户身份测试：
- Gitea HTTPS 连通性：`curl -sf https://${GITEA_DOMAIN}/api/v1/version`
- Gateway HTTPS 连通性：`curl -so /dev/null -w "%{http_code}" https://${LAUNCHPAD_DOMAIN}/gatewayproxy/`

验证的完整链路：CoreDNS 解析 → ingress-nginx 路由 → CA 信任 → 非 root 用户可用

### 6. E2E 测试顺序调整

- `09-uninstall-all.sh` → `10-uninstall-all.sh`（uninstall 会卸载 k3s，必须排最后）
- 新增 `09-pod-connectivity.sh`（在 uninstall 之前执行）

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `deploy/templates/kyverno-inject-ca.yaml` | 删除 initContainer/env/多余挂载，改用预合并 ConfigMap（70→30 行） |
| `deploy/templates/kyverno-sync-ca.yaml` | 新增 `launchpad-ca-bundle` 同步规则 |
| `deploy/scripts/lib/k8s-components.sh` | `setup_cert_distribution()` 预合并 CA bundle；`install_ingress_nginx()` 先卸载 Traefik |
| `deploy/scripts/lib/certs.sh` | CA 证书加 V3 扩展（basicConstraints + keyUsage） |
| `tests/cases/09-pod-connectivity.sh` | 新增 pod 内域名连通性测试（node 用户） |
| `tests/cases/10-uninstall-all.sh` | 从 09 重编号为 10 |

## 后续补充

### `su -` 导致 env 丢失问题

**问题**：pod 内 `bootstrap.sh` 通过 `su - node` 启动 PM2，`su -`（login shell）会清除所有继承的环境变量。导致 PM2 及其子进程（ani-code-daemon、web）都没有 `NODE_OPTIONS` 和 `NODE_EXTRA_CA_CERTS`，gateway 注册时报 `unable to verify the first certificate`。

**修复**：利用 `su -` 会 source `/etc/profile.d/*.sh` 的 Linux 标准机制：
1. `launchpad-ca-bundle` ConfigMap 增加 `launchpad-ca.sh` key：`export NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem`
2. Kyverno 注入策略挂载到 `/etc/profile.d/launchpad-ca.sh`
3. 容器级 env 只保留 `NODE_EXTRA_CA_CERTS`（覆盖非 `su -` 场景），去掉 `NODE_OPTIONS=--use-system-ca`（冗余）

**最终注入清单**：
| 挂载/env | 用途 |
|---------|------|
| `/etc/ssl/certs/ca-certificates.crt`（合并 bundle） | curl/wget/git 等非 Node 工具 |
| `/etc/ssl/certs/launchpad-ca.pem`（单独 CA PEM） | `NODE_EXTRA_CA_CERTS` 指向 |
| `/etc/profile.d/launchpad-ca.sh`（profile script） | `su -` 后恢复 env |
| env `NODE_EXTRA_CA_CERTS`  | 直接启动的 Node.js 进程 |

- dnsmasq 路由方案见 `2026-03-18-dnsmasq-docker-to-k8s-routing.md`

## 验证

E2E 测试全部通过。
