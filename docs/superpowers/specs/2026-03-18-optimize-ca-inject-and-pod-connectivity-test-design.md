# 优化 CA 注入策略 + Pod 连通性 E2E 测试

日期: 2026-03-18

## 背景

1. `kyverno-inject-ca.yaml` 当前用 `node:24`（~1GB）initContainer 在每个 pod 启动时合并 CA bundle，并注入了仅对 Node.js 有用的 `NODE_EXTRA_CA_CERTS`、`NODE_OPTIONS` 环境变量和 profile script。Pod 不一定跑 Node.js，这些都是多余的。
2. E2E 测试缺少 pod 内域名连通性验证——无法确认 CoreDNS 解析 + ingress-nginx 路由 + CA 信任链在 pod 内是否端到端工作。

## 设计

### 1. 部署时预合并 CA bundle

在 `k8s-components.sh` 的 `setup_cert_distribution()` 中，创建 `launchpad-ca-cert` ConfigMap 之后，额外生成 `launchpad-ca-bundle` ConfigMap：

```bash
cat /etc/ssl/certs/ca-certificates.crt "$ca_cert" > /tmp/ca-bundle.crt
kubectl create configmap launchpad-ca-bundle \
    --from-file=ca-certificates.crt=/tmp/ca-bundle.crt \
    -n kube-system --dry-run=client -o yaml | kubectl apply -f -
rm -f /tmp/ca-bundle.crt
```

### 2. Kyverno sync 策略增加 bundle 同步

`kyverno-sync-ca.yaml` 新增一条规则，将 `launchpad-ca-bundle` 从 kube-system 同步到所有 namespace（与现有 `launchpad-ca-cert` 同步逻辑一致）。

### 3. kyverno-inject-ca.yaml 精简

**删除：**
- initContainer `trust-ca`（`node:24` 镜像）
- `ca-bundle` emptyDir volume
- `launchpad-ca-cert` volume（单独 PEM 文件不再需要）
- 环境变量 `NODE_EXTRA_CA_CERTS` 和 `NODE_OPTIONS`
- volume mount `/etc/ssl/certs/launchpad-ca.pem`
- volume mount `/etc/profile.d/launchpad-ca.sh`

**保留/新增：**
- volume `ca-bundle` 改为引用 `launchpad-ca-bundle` ConfigMap
- 主容器挂载 `ca-certificates.crt` 到 `/etc/ssl/certs/ca-certificates.crt`

精简后完整策略：

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: inject-launchpad-ca-cert
spec:
  rules:
    - name: inject-ca-cert
      match:
        any:
          - resources:
              kinds:
                - Pod
      exclude:
        any:
          - resources:
              namespaces:
                - kube-system
                - kyverno
      mutate:
        patchStrategicMerge:
          spec:
            volumes:
              - name: ca-bundle
                configMap:
                  name: launchpad-ca-bundle
            containers:
              - (name): "*"
                volumeMounts:
                  - name: ca-bundle
                    mountPath: /etc/ssl/certs/ca-certificates.crt
                    subPath: ca-certificates.crt
                    readOnly: true
```

### 4. E2E 新增 `tests/cases/10-pod-connectivity.sh`

在 k8s 集群中创建临时 pod（node:24-alpine，自带 node 用户 + curl），以 node 用户身份测试域名连通性：

```bash
# 创建测试 pod
kubectl run test-conn --image=node:24-alpine --restart=Never --command -- sleep 300
kubectl wait --for=condition=ready pod/test-conn --timeout=60s

# 以 node 用户测试 gitea 域名 HTTPS 连通性
kubectl exec test-conn -- su -s /bin/sh node -c \
    "curl -sf https://${TEST_GITEA_DOMAIN}/api/v1/version"

# 以 node 用户测试 gateway 域名连通性
kubectl exec test-conn -- su -s /bin/sh node -c \
    "curl -sf https://${TEST_LAUNCHPAD_DOMAIN}/gateway/health"

# 清理
kubectl delete pod test-conn --grace-period=0 --force
```

验证内容：
- CoreDNS 正确解析域名
- ingress-nginx 正确路由请求
- 自签名 CA 信任链在非 root 用户下正常工作

## 修改文件清单

| 文件 | 改动 |
|------|------|
| `deploy/scripts/lib/k8s-components.sh` | `setup_cert_distribution()` 新增预合并 CA bundle ConfigMap |
| `deploy/templates/kyverno-inject-ca.yaml` | 删除 initContainer/env/多余挂载，改用预合并 ConfigMap |
| `deploy/templates/kyverno-sync-ca.yaml` | 新增 `launchpad-ca-bundle` 同步规则 |
| `tests/cases/10-pod-connectivity.sh` | 新增 pod 内域名连通性测试 |

## 不修改

- `deploy/templates/values-builtin.yml` — ingress-nginx 配置不变
- `tests/cases/01-09` — 现有测试用例不变
