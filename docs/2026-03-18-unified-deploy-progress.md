# 统一部署进度显示 + Pod CA 证书修复

日期: 2026-03-18

## 1. 统一部署进度显示

### 背景

部署原有 5 个阶段分别调用，各自输出独立日志：

```bash
setup_kubernetes       # 独立输出
setup_certificates     # 独立输出
setup_k8s_components   # 独立输出
deploy_services        # 12步 Style C 进度条
register_cluster       # 独立调用
```

### 改动

合并为单个 `deploy_services()` 函数，所有阶段在一个 Style C 进度显示中完成。步骤列表根据运行时条件动态构建。

### 修改文件

| 文件 | 改动 |
|------|------|
| `deploy/scripts/lang/en.sh` | 新增 6 个 MSG 键: `MSG_DEPLOY_PROGRESS_K3S`, `_SSL`, `_INGRESS`, `_COREDNS`, `_KYVERNO`, `_CERTDIST` |
| `deploy/scripts/lang/zh.sh` | 同上，中文翻译 |
| `deploy/scripts/lib/deploy.sh` | 重写 `deploy_services()`，使用 `_add_step()`/`_ds_step_map` 关联数组动态构建步骤 |
| `deploy/scripts/lib/k3s.sh` | `install_k3s()` 检查 `_DS_K3S_STEP` 变量，有值时使用 Style C 进度而非独立进度条 |
| `deploy/setup.sh` | 4 个独立调用替换为单个 `deploy_services` |

### 动态步骤列表

| 步骤 | 条件 |
|------|------|
| K3s 安装 | `K8S_MODE=builtin` 且 k3s 未运行 |
| SSL 证书 | 始终 |
| Ingress-Nginx 就绪 | `K8S_MODE=builtin` |
| CoreDNS 配置 | `K8S_MODE=builtin` |
| Kyverno 安装 | `K8S_MODE=builtin` 且 `SSL_MODE=selfsigned` |
| CA 证书分发 | `K8S_MODE=builtin` 且 `SSL_MODE=selfsigned` |
| 基础设施启动 | 始终 |
| PostgreSQL 就绪 | 始终 (外部 DB 时即时完成) |
| Redis 就绪 | 始终 (外部 DB 时即时完成) |
| 数据库迁移 | 始终 |
| Gitea 就绪 | 始终 |
| Gitea 初始化 | 始终 |
| API 就绪 | 始终 |
| UI 就绪 | 始终 |
| Router 就绪 | 始终 |
| Nginx | 始终 |
| 模版导入 | 始终 |
| 集群注册 | 始终 |

### setup.sh 改动

```bash
# 改动前:
setup_kubernetes
setup_certificates
setup_k8s_components
deploy_services
register_cluster || log_warn "..."

# 改动后:
deploy_services
```

### K3s 进度集成 (`_DS_K3S_STEP` 模式)

`install_k3s()` 原本有自己的 `_draw_progress`/`progress_done` 进度条。当从 `deploy_services()` 的 tty 模式调用时，设置 `_DS_K3S_STEP=<step_idx>` 使其改用 `_update_step`/`_redraw`。安装阶段占 0-70%，等待就绪占 70-95%。独立调用 (`--setup-k3s`) 时行为不变。

> **2026-03-19 更新**: `--resume` 已改为直接调用 `deploy_services()`（不再使用独立的 `resume_deploy()`），因此也使用 Style C 统一进度显示。`_DS_K3S_STEP` 独立模式仅在 `--setup-k3s` 时生效。详见 `2026-03-19-gateway-tls-resume-coredns-fixes.md` #2。

---

## 2. Pod 自签名 CA 证书信任修复

### 问题

项目创建失败，Pod 内 `curl`/`wget` 下载 bootstrap 脚本时报错：
```
curl: (60) SSL certificate problem: unable to get local issuer certificate
```

### 根因

Kyverno 注入策略做了两件事，都不够：
- 挂载 CA 到 `/etc/ssl/certs/launchpad-ca.pem` — curl/wget 不扫描独立 PEM 文件
- 设置 `NODE_EXTRA_CA_CERTS` — 只对 Node.js 有效，curl/wget/git 不认

### 修复（初版 → 后续优化）

初版方案：initContainer(`node:24`) 在每个 pod 启动时合并 CA bundle。

后续优化（见 `2026-03-18-optimize-ca-inject-and-pod-connectivity.md`）：
- 改为部署时预合并 CA bundle 为 `launchpad-ca-bundle` ConfigMap
- Kyverno 仅挂载 ConfigMap，无 initContainer、无 env、无 profile script
- CA 证书升级为 V3 格式（兼容 OpenSSL 3.3+）

---

## 3. 实验性配置默认值

修改 `deploy/templates/env.template`，将两个实验性配置改为默认值：
- `EXP_BOOTSTRAP_USE_BUN_RUNTIME=false` (原来是 `true`)
- `EXP_BOOTSTRAP_DEBUG=false` (新增)

---

## 4. 模版导入时序修复

### 问题

全量部署时 `import_templates` 中 git push 静默失败，Gitea 里 repo 被创建但内容为空。

### 根因

`import_templates` 的 git push 使用 `https://${GITEA_DOMAIN}/...`，需要 Nginx 代理 + `/etc/hosts` 解析。但原来的执行顺序是 **Templates → Nginx**，push 时 Nginx 还没启动，域名也无法解析。加上 `>/dev/null 2>&1` 吞掉了所有错误，导致失败完全静默。

### 修复

三层防护：

1. **调整执行顺序**：Nginx 移到模版导入之前
   ```
   之前: ... → Router → Templates → Nginx → 集群注册
   现在: ... → Router → Nginx → Templates → 集群注册
   ```

2. **HTTPS 预检**：在 git push 前循环最多 15 次（30 秒）检查 `https://${GITEA_DOMAIN}` 是否可达，确认 DNS + TLS + Nginx + Gitea 整条链路通了

3. **Git push 重试**：每个 repo 最多重试 3 次，间隔 3 秒，失败时打印 warn 日志（不再静默）

4. **Nginx 就绪缓冲**：Nginx healthy 之后 sleep 5 秒，等待 HTTPS 代理完全初始化

### 修改文件

仅 `deploy/scripts/lib/deploy.sh`
