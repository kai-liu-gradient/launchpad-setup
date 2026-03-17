# 初始化逻辑重构设计

**日期**: 2026-03-17
**范围**: setup.sh 初始化流程、URL 路由、k8s 组件部署、证书分发

---

## 背景

测试服务器部署中暴露了多个问题（详见 `docs/testserver-fixes-2026-03-17.md`）：
- k8s pod 无法访问 Docker Compose 服务，依赖手动 iptables 规则（不持久化）
- 自签名证书通过 `NODE_TLS_REJECT_UNAUTHORIZED=0` 绕过验证
- CoreDNS 泛域名 ClusterIP 需要手动硬编码
- 集群注册时 kubeconfig 为空导致 unhealthy
- 部分 URL 使用 Docker 内部地址，k8s pod 无法解析

本次重构解决以上问题，目标：**消除 iptables 依赖，所有 k8s pod 可访问的 URL 走 nginx 域名，自签名证书通过 Kyverno 正规分发**。

---

## 整体流程变更

### 修改前

```
setup_kubernetes() → setup_certificates() → deploy_services() → register_cluster()
```

### 修改后

```
setup_kubernetes() → setup_certificates() → setup_k8s_components()【新增】→ deploy_services() → register_cluster()
```

### 条件逻辑

- `setup_k8s_components()` 仅在 `K8S_MODE=builtin` 时执行
- Kyverno + 证书分发仅在 `SSL_MODE=selfsigned` 时执行
- CoreDNS 配置在 builtin 模式下始终执行
- `check_environment()` 新增 `helm` 命令检查（builtin 模式下必需）

---

## 变更 1：URL 走 nginx 域名

### 原则

只改 k8s pod 实际消费的 URL。Docker Compose 内部服务间通信保持 Docker DNS，不做无谓改动。

### 变量消费者分析

| 变量 | 消费者 | 是否改？ | 原因 |
|------|--------|---------|------|
| `GITEA_GIT_SSH` | k8s pod (git clone) | **改** | pod 是 SSH 客户端，需解析域名 |
| `DEFAULT_BACKEND` | router (Docker) | **改** | builtin 模式需指向 ingress-nginx NodePort |
| `ROUTER_LOCAL_URL` | API (Docker) | 不改 | Docker 内部直接调用，不经 k8s |
| `LAUNCHPAD_API_URL` (gateway env) | Gateway (Docker) | 不改 | Docker 内部直接调用，不经 k8s |
| `ANI_CODE_GATEWAY_URL` | k8s pod (daemon) | 不改 | 已走 nginx (`${GATEWAY_PUBLIC_URL}`) |

### render.sh 变更

```bash
# GITEA_GIT_SSH — k8s pod 做 git clone 时使用，必须用可解析域名
# 修改前
export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@gitea:2222}"
# 修改后
export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@${GITEA_DOMAIN}:2222}"

# DEFAULT_BACKEND — interact.sh 在 builtin 模式设为 "localhost"，需显式覆盖
# 注意：不能用 ${DEFAULT_BACKEND:-...} 因为 interact.sh 已赋值
local host_ip
host_ip=$(detect_internal_ip)
if [[ "$K8S_MODE" == "builtin" ]]; then
    export DEFAULT_BACKEND="http://${host_ip}:30080"
fi
```

### 不变的 URL（Docker 内部通信）

| 变量 | 值 | 原因 |
|------|-----|------|
| `ROUTER_LOCAL_URL` | `http://router:6580` | API (Docker) 直接调用 router |
| `LAUNCHPAD_API_URL` (gateway env) | `http://api:6802` | Gateway (Docker) 直接调用 API |
| `LAUNCHPAD_API_URL` (router env) | `http://api:6802/api/v1/public` | Router (Docker) 直接调用 API |

---

## 变更 2：nginx.conf — wildcard 流量路径不变

nginx.conf.template 无改动。泛域名流量路径保持：

```
*.domain.com → Docker nginx (443, TLS 终止) → router:6580 → ingress-nginx (NodePort 30080) → pod
```

不新增 `/router/` location — `ROUTER_LOCAL_URL` 保持 Docker 内部通信，API 不需要通过 nginx 访问 router。

---

## 变更 3：新增 setup_k8s_components() 阶段

新建 `deploy/scripts/lib/k8s-components.sh`，包含以下函数：

### 3.1 install_ingress_nginx()

使用 Helm 安装 ingress-nginx-controller，配置文件为 `deploy/templates/values-builtin.yml`。

```bash
install_ingress_nginx() {
    if helm status ingress-nginx -n ingress-nginx &>/dev/null; then
        log_ok "ingress-nginx already installed"
        return 0
    fi

    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
    helm repo update
    helm install ingress-nginx ingress-nginx/ingress-nginx \
        -n ingress-nginx --create-namespace \
        -f "${DEPLOY_DIR}/templates/values-builtin.yml"

    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=120s
}
```

### 3.2 configure_coredns()

动态获取 ingress-nginx ClusterIP，渲染 CoreDNS ConfigMap 模板。

```bash
configure_coredns() {
    local host_ip
    host_ip=$(detect_internal_ip)

    local cluster_ip
    cluster_ip=$(kubectl get svc ingress-nginx-controller \
        -n ingress-nginx -o jsonpath='{.spec.clusterIP}')

    local domain_escaped="${DOMAIN//./\\.}"

    export HOST_IP="$host_ip"
    export INGRESS_NGINX_CLUSTER_IP="$cluster_ip"
    export DOMAIN_ESCAPED="$domain_escaped"

    # 限定变量列表，避免覆盖 CoreDNS 的 {{ .Name }} 模板语法
    envsubst '${HOST_IP} ${INGRESS_NGINX_CLUSTER_IP} ${DOMAIN} ${DOMAIN_ESCAPED} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN}' \
        < "${DEPLOY_DIR}/templates/coredns-custom.yaml.template" \
        > /tmp/coredns-custom.yaml

    kubectl apply -f /tmp/coredns-custom.yaml

    # 修补默认 CoreDNS ConfigMap：移除 loop 插件，forward 到公共 DNS
    # 注意：sed 使用精确匹配避免重复执行时产生 "# # loop"
    kubectl get cm coredns -n kube-system -o yaml \
        | sed '/^[^#]*loop$/s/loop/# loop/' \
        | sed 's|forward \. /etc/resolv\.conf|forward . 223.5.5.5 8.8.8.8|' \
        | kubectl apply -f -

    # k3s 原生支持 coredns-custom ConfigMap 约定（自动 import）
    # 如果迁移到 kubeadm 集群需要在 Corefile 中手动添加 import 指令
    kubectl rollout restart deploy/coredns -n kube-system
    rm -f /tmp/coredns-custom.yaml
}
```

### 3.3 install_kyverno()

```bash
install_kyverno() {
    if helm status kyverno -n kyverno &>/dev/null; then
        log_ok "Kyverno already installed"
        return 0
    fi

    helm repo add kyverno https://kyverno.github.io/kyverno/
    helm repo update
    helm install kyverno kyverno/kyverno \
        -n kyverno --create-namespace

    kubectl wait --namespace kyverno \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=admission-controller \
        --timeout=120s
}
```

### 3.4 setup_cert_distribution()

仅在 `SSL_MODE=selfsigned` 时执行。

步骤 1 — 创建 CA 证书 ConfigMap（kube-system，作为源）：

```bash
kubectl create configmap launchpad-ca-cert \
    --from-file=ca.pem="${DEPLOY_DIR}/generated/nginx/certs/ca.pem" \
    -n kube-system --dry-run=client -o yaml | kubectl apply -f -
```

步骤 2 — Apply 两个 Kyverno ClusterPolicy：

**Policy 1 — 分发 ConfigMap 到每个 namespace** (`deploy/templates/kyverno-sync-ca.yaml`)：

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: sync-launchpad-ca-cert
spec:
  generateExisting: true
  rules:
    - name: sync-ca-configmap
      match:
        any:
          - resources:
              kinds:
                - Namespace
      exclude:
        any:
          - resources:
              namespaces:
                - kube-system
                - kyverno
      generate:
        synchronize: true
        apiVersion: v1
        kind: ConfigMap
        name: launchpad-ca-cert
        namespace: "{{request.object.metadata.name}}"
        clone:
          namespace: kube-system
          name: launchpad-ca-cert
```

**Policy 2 — 注入 volume + 环境变量到 pod** (`deploy/templates/kyverno-inject-ca.yaml`)：

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
                - ingress-nginx
      mutate:
        patchStrategicMerge:
          spec:
            volumes:
              - name: launchpad-ca-cert
                configMap:
                  name: launchpad-ca-cert
            containers:
              - (name): "*"
                volumeMounts:
                  - name: launchpad-ca-cert
                    mountPath: /etc/ssl/certs/launchpad-ca.pem
                    subPath: ca.pem
                    readOnly: true
                env:
                  - name: NODE_EXTRA_CA_CERTS
                    value: /etc/ssl/certs/launchpad-ca.pem
```

### 3.5 setup_k8s_components() 编排

```bash
setup_k8s_components() {
    [[ "$K8S_MODE" != "builtin" ]] && return 0

    log_info "Setting up k8s components..."

    install_ingress_nginx
    configure_coredns

    if [[ "$SSL_MODE" == "selfsigned" ]]; then
        install_kyverno
        setup_cert_distribution
    fi

    log_ok "k8s components ready"
}
```

---

## 变更 4：ingress-nginx values-builtin.yml

新增 `deploy/templates/values-builtin.yml`：

```yaml
controller:
  replicaCount: 1

  ingressClass: nginx

  ingressClassResource:
    name: nginx
    default: true

  config:
    enable-real-ip: "true"
    use-forwarded-headers: "true"
    compute-full-forwarded-for: "true"

    worker-processes: "auto"
    max-worker-connections: "30000"

    proxy-body-size: 50m
    keep-alive: "300"
    keep-alive-requests: "65535"

    ssl-redirect: "false"

    log-format-escape-json: "true"
    log-format-upstream: |
      { "time": "$time_iso8601",
        "remote_addr": "$remote_addr",
        "x-forward-for": "$proxy_add_x_forwarded_for",
        "request_id": "$request_id",
        "status": $status,
        "host": "$host",
        "path": "$uri",
        "method": "$request_method",
        "request_time": $request_time,
        "upstream_addr": "$upstream_addr"
      }

  service:
    type: NodePort
    nodePorts:
      http: 30080
      https: 30443
```

与 `old/helm/values.yml` 差异：replicaCount 1、NodePort、无 Azure 注解、无 topology 约束。

---

## 变更 5：CoreDNS 模板

新增 `deploy/templates/coredns-custom.yaml.template`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-custom
  namespace: kube-system
data:
  launchpad-specific.server: |
    ${LAUNCHPAD_DOMAIN}:53 ${GITEA_DOMAIN}:53 {
        hosts {
            ${HOST_IP} ${LAUNCHPAD_DOMAIN}
            ${HOST_IP} ${GITEA_DOMAIN}
            ttl 60
        }
    }

  wildcard.server: |
    ${DOMAIN}:53 {
        hosts {
            ${INGRESS_NGINX_CLUSTER_IP} ${DOMAIN}
            ttl 60
            fallthrough
        }
        template IN A {
            match ".*\.${DOMAIN_ESCAPED}\.$"
            answer "{{ .Name }} 60 IN A ${INGRESS_NGINX_CLUSTER_IP}"
        }
    }
```

---

## 不变的部分

- `register_cluster()` / `setup_heartbeat()` — 保持现有逻辑不改
- `deploy_services()` — Docker Compose 部署流程不变
- `setup_certificates()` — 证书生成逻辑不变
- nginx.conf wildcard server block — 保持 proxy 到 `launchpad-router`
- Docker Compose 内部服务间通信 URL — 保持 Docker DNS

---

## 新增文件清单

| 文件 | 类型 | 用途 |
|------|------|------|
| `deploy/scripts/lib/k8s-components.sh` | Bash 模块 | `setup_k8s_components()` 及子函数 |
| `deploy/templates/values-builtin.yml` | Helm values | ingress-nginx builtin 配置 |
| `deploy/templates/coredns-custom.yaml.template` | K8s 模板 | CoreDNS 自定义配置 |
| `deploy/templates/kyverno-sync-ca.yaml` | K8s manifest | Kyverno 证书分发策略 |
| `deploy/templates/kyverno-inject-ca.yaml` | K8s manifest | Kyverno 证书注入策略 |

## 修改文件清单

| 文件 | 变更 |
|------|------|
| `deploy/setup.sh` | 加载 k8s-components.sh，流程中插入 `setup_k8s_components()` |
| `deploy/scripts/lib/render.sh` | `GITEA_GIT_SSH` 默认值改域名，`DEFAULT_BACKEND` builtin 模式显式覆盖 |
| `deploy/scripts/lib/detect.sh` | `check_environment()` 新增 helm 检查（builtin 模式） |
