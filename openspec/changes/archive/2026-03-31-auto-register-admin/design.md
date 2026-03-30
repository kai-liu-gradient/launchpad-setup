## 上下文

当前部署流程中 admin 相关逻辑：
- `Config.AdminEmail` — 默认 `admin@{domain}`，用于 Gitea admin 和部署结果展示
- `secrets.AdminPassword` — 随机生成 24 字符 URL-safe base64，仅用于 Gitea admin 创建
- `bootstrapGitea` 步骤 — 用 `AdminEmail` 和 `AdminPassword` 创建 Gitea admin 用户
- 部署完成后打印 admin 邮箱和密码，但这只是 Gitea 的凭据

launchpad 平台本身没有 admin 用户预注册步骤，用户需要手动到 web 界面注册。

## 目标 / 非目标

**目标：**
- Express 模式支持用户输入自定义 admin 邮箱（可选，默认 `admin@{domain}`）
- 密码全自动生成，不让用户输入
- admin 邮箱域名自动加入 settings.yml 注册白名单
- Gitea admin 和 launchpad admin 统一使用同一套凭据
- 部署完成后 launchpad 平台有一个可直接登录的 admin 用户，邮箱已验证
- Domain Preview 展示 admin 邮箱
- 部署结果使用 `cfg.AdminEmail` 展示（而非硬编码），清晰展示账号密码并提示保存

**非目标：**
- 不让用户输入密码（密码全自动生成）
- 不实现多 admin 用户创建
- 不涉及 SSO/OAuth 登录流程

## 决策

### 1. 密码生成需满足 API 密码策略

launchpad API 的 `/auth/register` 端点要求密码包含特殊字符。原有的 `randBase64Safe(24)` 只生成字母数字和 `-_`，不满足要求。

**方案**：AdminPassword 单独生成，格式为 `randBase64Safe(20) + "!@A1"`，确保包含特殊字符（`!@`）、大写字母（`A`）和数字（`1`）。其他 DB 密码不受影响，继续使用纯 base64。

### 2. 注册 API 需要 username 字段

API 的 register 接口要求 `email`、`password`、`name`、`username` 四个字段。username 从 AdminEmail 的 `@` 前部分提取（如 `admin@example.com` → `admin`）。

### 3. 通过 Node.js 调用 API（容器内无 curl）

API 容器基于 Node.js 镜像，没有 curl。通过 `docker exec` 运行 Node.js 内联脚本发送 HTTP 请求。连接地址使用 `127.0.0.1`（而非 `localhost`）避免 IPv6 解析问题。

```js
// docker exec generated-api-1 node -e "..."
const http = require('http');
const data = JSON.stringify({email, password, name:'Admin', username});
http.request({hostname:'127.0.0.1', port:6802, path:'/api/v1/auth/register', ...})
```

### 4. 邮箱验证需指定 schema

users 表位于 `launchpad_main` schema 下，SQL 必须使用全限定表名：

```sql
UPDATE launchpad_main.users SET email_verified = true WHERE email = '{AdminEmail}';
```

builtin DB 通过 `docker exec postgres` 执行，external DB 通过 `psql` + 配置的 URL 执行。

### 5. Express 模式收集 admin 邮箱

Express 模式增加可选的 Admin Email 输入框（默认 `admin@{domain}`）。密码不暴露，全自动生成。用户输入的邮箱覆盖 `Config.AdminEmail`，同时传递到 Domain Preview 展示。

### 6. admin 邮箱域名自动加入注册白名单

settings.yml 的 `allowedDomains` 仅包含 `Config.Domain`，当 admin 邮箱域名不同（如 `admin@gmail.com` vs `corp.kola.com`）时注册会被拦截。

**方案**：在 `Derived.AllowedDomains` 中收集 `Config.Domain` + admin 邮箱域名（去重），模板渲染为：
```yaml
allowedDomains: ["corp.kola.com", "gmail.com"]
```

### 7. Domain Preview 展示 admin 邮箱

`RunDomainPreview` 增加 `adminEmail` 参数，预览中展示 Admin 行。未输入时显示默认的 `admin@{domain}`。

### 8. 部署结果使用 cfg.AdminEmail

部署完成后展示凭据时使用 `cfg.AdminEmail`（而非硬编码 `"admin@" + cfg.Domain`），确保展示用户实际输入的邮箱。

### 9. 幂等性

注册 API 返回 409/422/400（用户已存在）时跳过并继续，不阻塞部署。

### 6. 部署结果强调密码保存

部署完成后展示 admin 凭据，增加醒目提示：
```
⚠️  Admin Credentials (SAVE THESE!):
  Admin     : admin@example.com
  Password  : xxxxxxxxxxxxxx!@A1

  This password is auto-generated and will NOT be shown again.
  To retrieve later: cat .secrets.yaml | grep admin_password
```

## 风险 / 权衡

- **密码丢失** → 部署结果提示用户记录密码，并告知 `.secrets.yaml` 位置作为备查
- **API 密码策略变更** → 如果 API 修改密码规则，可能需要调整后缀 `!@A1`。当前覆盖常见要求：特殊字符 + 大写 + 数字
- **API 容器无 curl** → 使用 Node.js 内联脚本替代，依赖 API 容器中已有的 node 运行时
- **幂等** → 重部署时用户已存在则跳过注册，密码不会被覆盖
