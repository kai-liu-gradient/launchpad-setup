# 替换 Traefik 为 ingress-nginx

日期: 2026-03-18

## 背景

k3s 内置 Traefik 作为 ingress controller，但 API 应用创建 ingress 时硬编码 `ingressClassName: nginx`，导致 Traefik 不处理这些 ingress，泛域名 `*.testclaw.com` 访问 pod 返回 404。

项目中已有完整的 ingress-nginx helm values 文件 (`values-builtin.yml`)，说明原本就计划用 ingress-nginx。直接换掉 Traefik 是最干净的方案。

## 改动

### 1. k3s.sh — 禁用 Traefik

- 删除两处 `cp traefik-config.yaml` 到 k3s manifests 的逻辑
- k3s 安装命令（verbose 和非 verbose 两处）加 `--disable traefik`

### 2. k8s-components.sh — 替换 ingress controller

- `wait_for_traefik()` → `install_ingress_nginx()`
  - 幂等检查: `helm status ingress-nginx` 已安装则跳过
  - `helm install ingress-nginx` 使用 `values-builtin.yml`（NodePort 30080/30443）
  - 等待 controller pod ready（最多 60s 等 pod 出现 + 120s wait ready）
- `configure_coredns()` 中 ClusterIP 来源:
  - 之前: `kubectl get svc traefik -n kube-system`
  - 现在: `kubectl get svc ingress-nginx-controller -n ingress-nginx`

### 3. deploy.sh — 步骤重命名

- `_add_step "traefik"` → `_add_step "ingress"`
- `$(_step traefik)` → `$(_step ingress)`
- 调用 `install_ingress_nginx` 替换 `wait_for_traefik`

### 4. i18n

| 文件 | 之前 | 之后 |
|------|------|------|
| `en.sh` | `MSG_DEPLOY_PROGRESS_TRAEFIK="Traefik ready"` | `MSG_DEPLOY_PROGRESS_INGRESS="Ingress-Nginx ready"` |
| `zh.sh` | `MSG_DEPLOY_PROGRESS_TRAEFIK="Traefik 就绪"` | `MSG_DEPLOY_PROGRESS_INGRESS="Ingress-Nginx 就绪"` |

### 5. coredns-custom.yaml.template

注释更新: `Traefik ClusterIP` → `ingress-nginx ClusterIP`

## 删除的文件

| 文件 | 原用途 |
|------|--------|
| `deploy/templates/traefik-config.yaml` | Traefik HelmChartConfig (NodePort 配置) |
| `deploy/templates/kyverno-ingress-class.yaml` | Kyverno 策略: 将 `ingressClassName: nginx` 重写为 `traefik` |

## 未修改的文件

- `deploy/templates/values-builtin.yml` — 已有正确的 ingress-nginx 配置
- `deploy/scripts/lib/render.sh` — `DEFAULT_BACKEND=http://${host_ip}:30080` 不变（端口一致）

## 后续补充

- Traefik 卸载逻辑：`install_ingress_nginx()` 新增检测并卸载残留 Traefik（见 `2026-03-18-optimize-ca-inject-and-pod-connectivity.md`）
- Docker→k8s 路由：dnsmasq 配置让 Docker 容器通过 ClusterIP:80 访问 ingress-nginx（见 `2026-03-18-dnsmasq-docker-to-k8s-routing.md`）

## 验证

1. `bash -n` 所有修改文件语法检查通过
2. 部署后 `*.testclaw.com` 泛域名可正确路由到 pod
