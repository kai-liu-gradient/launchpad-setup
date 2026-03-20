# HostIP 硬编码 + CA 重生成 + Kyverno 同步导致容器/Pod 连接失败

日期: 2026-03-19

## 问题现象

1. **Docker 容器内**：API 容器 health 页面显示 Gateway 和 Gitea 状态为 `unhealthy`，错误 `fetch failed`。原因是容器内 `curl https://launchpad.testclaw.com` 连接被拒绝。
2. **K3s Pod 内**：新部署的 Pod 执行 `curl https://launchpad.testclaw.com:443` 报 `Couldn't connect to server`，无法完成 bootstrap。
3. **K3s Pod 内**：即使域名解析修复后，Pod 仍报 `SSL certificate problem: unable to get local issuer certificate`。

## 根因

五个 bug：

### Bug 1: `ComputeDerived()` 中 HostIP 硬编码为 127.0.0.1

**文件**: `internal/template/derived.go:99`

```go
// 修复前
d.HostIP = "127.0.0.1"
```

`HostIP` 被用于 docker-compose 的 `extra_hosts` 和 CoreDNS 的 `launchpad-specific.server`。

- **Docker 容器影响**：`extra_hosts` 把 `launchpad.testclaw.com` 映射到 `127.0.0.1`，容器内 `127.0.0.1` 是容器自身而非宿主机，因此无法到达 nginx（运行在宿主机的 Docker 容器中，端口映射到 443）。
- **Pod 影响**：CoreDNS 把 `launchpad.testclaw.com` 解析为 `127.0.0.1`，Pod 内 `127.0.0.1` 是 Pod 自身。

### Bug 2: `configureCoreDNS()` 未将 IngressClusterIP 写入模板

**文件**: `internal/engine/steps.go:172-200`

`configureCoreDNS` 获取了 ingress-nginx 的 ClusterIP 但仅用于校验 service 是否存在，没有保存到 `RuntimeValues` 也没有重新渲染模板。导致 CoreDNS wildcard server 中的 IP 字段为空：

```
# 修复前渲染结果
answer "{{ .Name }} 60 IN A "    # ← IP 为空
```

Pod 访问 `*.testclaw.com` 的通配域名时 DNS 解析失败。

### Bug 3: `generateSelfsignedCerts()` 每次重新生成 CA

**文件**: `internal/engine/certs.go:29-50`

`--resume` 重新运行时，`generateSelfsignedCerts` 无条件重新生成 CA key 和 CA cert。新 CA 会签发新的 server 证书，但 Kyverno 已经分发到各 namespace 的 `ca-certificates.crt` bundle 里包含的是旧 CA。已运行的 Pod 挂载的是 configmap 的旧快照，无法验证新 CA 签发的证书。

时序：
1. 首次安装 → CA-A 生成 → bundle 含 CA-A → Pod 启动挂载 CA-A ✅
2. `--resume` → CA-B 生成 → cert 用 CA-B 签发 → Kyverno 更新 bundle 为 CA-B → **已运行 Pod 仍持有 CA-A** ❌

### Bug 4: `fullchain.pem` 仅含 server 证书，缺少 CA 证书

**文件**: `internal/engine/certs.go:79-99`

`openssl x509 -req` 签发证书时，输出文件仅包含 server 证书本身，不包含 CA 证书。nginx 返回的证书链只有 1 个证书，客户端无法构建完整链到 CA，报 `unable to get local issuer certificate`。

修复：签发后手动拼接 server cert + CA cert 生成 `fullchain.pem`：

```go
serverCert := filepath.Join(certDir, "server-cert.pem")
// ... 签发 server cert 到 server-cert.pem ...

// 构建 fullchain: server cert + CA cert
fullchainPath := filepath.Join(certDir, "fullchain.pem")
serverData, _ := os.ReadFile(serverCert)
caData, _ := os.ReadFile(caCert)
os.WriteFile(fullchainPath, append(serverData, caData...), 0644)
os.Remove(serverCert)
```

### Bug 5: Kyverno `synchronize: true` 未自动同步已更新的 ConfigMap 到已有 Namespace

**现象**: `kube-system` 中的 `launchpad-ca-cert` ConfigMap 已包含正确 CA (`4F:54:FA`)，但 `user-kai-liu` namespace 中的同名 ConfigMap 仍为旧 CA (`C4:E5:15`)。

**根因**: Kyverno 的 `synchronize: true` 在 `generate` 规则中的行为：当源 ConfigMap 在 `--resume` 期间被多次更新（多次 CA 重生成），Kyverno 的 sync 机制可能未能将每次变更传播到所有目标 namespace。特别是当 `skipBackgroundRequests: true` 时，后台同步受限。

**影响**: Pod 挂载的 CA ConfigMap 包含旧 CA，无法验证由新 CA 签发的证书。

**修复**: 删除目标 namespace 中的旧 ConfigMap，Kyverno 自动重新从 `kube-system` 克隆最新版本：

```bash
for ns in $(kubectl get ns -o name | grep -v "kube-system\|kyverno" | sed "s|namespace/||"); do
    kubectl delete configmap launchpad-ca-cert launchpad-ca-bundle -n "$ns" --ignore-not-found
done
# Kyverno 在 5 秒内自动重新生成正确的 ConfigMap
```

**注意**: 此问题仅在修复 Bug 3（CA 复用）之前的多次 `--resume` 后出现。修复 Bug 3 后，CA 不再变化，因此不会再触发此问题。

## 修复

### 修复 1: 动态检测 HostIP

**文件**: `internal/template/derived.go`

新增 `detectHostIP()` 函数，通过 `net.InterfaceAddrs()` 获取第一个非回环 IPv4 地址：

```go
func detectHostIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return "127.0.0.1"
    }
    for _, addr := range addrs {
        if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
            return ipNet.IP.String()
        }
    }
    return "127.0.0.1"
}
```

`ComputeDerived()` 改为调用 `detectHostIP()`：

```go
// 修复后
d.HostIP = detectHostIP()
```

影响范围：
- `docker-compose.yml.tmpl` 中的 `extra_hosts`（4 处服务）
- `coredns-custom.yaml.tmpl` 中的 `launchpad-specific.server`
- `nginx.conf.tmpl` 中的 `DEFAULT_BACKEND`

### 修复 2: configureCoreDNS 保存 ClusterIP 并重新渲染

**文件**: `internal/engine/steps.go` — `configureCoreDNS()`

获取 ClusterIP 后，保存到 RuntimeValues 并调用 `renderWithRuntime()` 重新渲染所有模板，然后再 apply CoreDNS 配置：

```go
func (e *Engine) configureCoreDNS(ctx context.Context) error {
    clusterIP, err := RunWithOutput(ctx, "get-ingress-ip", 10*time.Second,
        "kubectl", "get", "svc", "ingress-nginx-controller",
        "-n", "ingress-nginx", "-o", "jsonpath={.spec.clusterIP}")
    if err != nil {
        return fmt.Errorf("getting ingress-nginx ClusterIP: %w", err)
    }
    clusterIP = strings.TrimSpace(clusterIP)

    // 保存到 runtime 并重新渲染
    runtimePath := filepath.Join(e.output, ".runtime.yaml")
    rv, _ := tmpl.LoadRuntime(runtimePath)
    rv.IngressClusterIP = clusterIP
    rv.Save(runtimePath)
    if err := e.renderWithRuntime(rv); err != nil {
        return fmt.Errorf("re-rendering templates with ingress ClusterIP: %w", err)
    }

    // 此时 coredns-custom.yaml 中 IP 已正确填入
    corednsPath := filepath.Join(e.output, "coredns-custom.yaml")
    // ... apply ...
}
```

### 修复 3: 复用已有 CA，避免重新生成

**文件**: `internal/engine/certs.go` — `generateSelfsignedCerts()`

检查 `ca.key` 是否已存在，已存在则跳过 CA 生成，仅重新签发 server 证书（server 证书重签无影响，因为 CA 不变）：

```go
caKey := filepath.Join(certDir, "ca.key")
caCert := filepath.Join(certDir, "ca.pem")
if _, err := os.Stat(caKey); os.IsNotExist(err) {
    // 首次安装：生成 CA key + CA cert
    RunWithTimeout(ctx, "ca-key", ..., "openssl", "genrsa", "-out", caKey, "2048")
    RunWithTimeout(ctx, "ca-cert", ..., "openssl", "req", "-new", "-x509", ...)
}
// 无论首次还是 resume，都用现有 CA 签发 server cert
```

验证：连续两次 `--resume` 后 CA 指纹不变：
```
BEFORE: sha256 Fingerprint=4F:54:FA:67:...
AFTER:  sha256 Fingerprint=4F:54:FA:67:...  # 完全一致
```

## 验证

修复后 CoreDNS 配置：

```
launchpad-specific.server:
    10.233.201.133 launchpad.testclaw.com      # 宿主机 IP → Docker nginx
    10.233.201.133 launchpad-gitea.testclaw.com

wildcard.server:
    10.43.14.79 testclaw.com                   # ingress-nginx ClusterIP
    answer "{{ .Name }} 60 IN A 10.43.14.79"
```

验证结果：
- Health 页面：Gateway ✅ healthy、Gitea ✅ healthy（含 SSH 和 Token 测试）
- Pod 连通性（跳过验证）：`kubectl run --rm -it test -- curl -sk https://launchpad.testclaw.com/` → 200
- Pod 连通性（CA 验证）：`kubectl run --rm -it test -- curl --cacert /path/ca.pem https://launchpad.testclaw.com/` → 200
- Pod 连通性（Bundle 验证）：挂载 `launchpad-ca-bundle` 的 `ca-certificates.crt` → 200
- 全部 8 个 health 模块均为 healthy

## 涉及文件

| 文件 | 改动 |
|------|------|
| `internal/template/derived.go` | 新增 `detectHostIP()`，`ComputeDerived` 调用它替换硬编码 |
| `internal/engine/steps.go` | `configureCoreDNS` 保存 ClusterIP 到 RuntimeValues 并重新渲染 |
| `internal/engine/certs.go` | `generateSelfsignedCerts` 复用已有 CA，仅首次安装时生成；fullchain 拼接 server cert + CA cert |
