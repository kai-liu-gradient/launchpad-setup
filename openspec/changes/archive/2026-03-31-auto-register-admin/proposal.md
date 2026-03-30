## 为什么

当前部署完成后，launchpad 平台没有预创建 admin 用户。管理员必须手动到 web 界面注册，并且如果 `requireEmailVerification` 开启，还需要手动处理邮箱验证。这在私有部署场景中很不便：部署完就应该能直接登录管理后台。

目前只有 Gitea 的 admin 用户在 `bootstrapGitea` 步骤中自动创建，launchpad 平台自身缺少这一步。

## 变更内容

1. **Express 模式收集 admin 邮箱**：用户可在 Express 模式中输入自定义 admin 邮箱（可选，默认 `admin@{domain}`），密码全自动生成
2. **admin 邮箱域名自动加入注册白名单**：当 admin 邮箱域名与 Config.Domain 不同时，自动将其加入 settings.yml 的 `allowedDomains`
3. **部署流程自动注册 admin 用户**：在 API 健康检查通过后，通过 Node.js 调用 launchpad API 注册 admin 用户（容器内无 curl）
4. **自动通过邮箱验证**：注册完成后通过 SQL（`launchpad_main.users`）标记邮箱已验证
5. **密码满足 API 策略**：AdminPassword 生成追加 `!@A1`，满足 API 要求的特殊字符 + 大写 + 数字
6. **Domain Preview 展示 admin 邮箱**：预览界面增加 Admin 行
7. **部署结果展示凭据**：使用 `cfg.AdminEmail`（而非硬编码），醒目提示保存密码并告知 `.secrets.yaml` 查找方式

## 功能 (Capabilities)

### 新增功能
- `auto-register-admin`: 部署流程中自动注册 admin 用户并通过邮箱验证

### 修改功能
- `domain-preview`: 预览界面增加 Admin 邮箱展示

## 影响

- `internal/tui/wizard/express.go` — Express 模式增加 Admin Email 输入
- `internal/tui/wizard/wizard.go` — 传递 admin 邮箱到 Config + ExpressAdminPassword 清理
- `internal/tui/wizard/domain_preview.go` — 预览增加 Admin 行
- `internal/tui/panel/tabs/basic.go` — Custom 模式 Basic Tab 已有 Admin 相关字段
- `internal/secrets/secrets.go` — AdminPassword 单独生成，追加 `!@A1` 满足密码策略
- `internal/engine/admin.go` — 新增：通过 Node.js + docker exec 注册用户 + SQL 验证邮箱
- `internal/engine/steps.go` — 新增 "Registering admin user" 步骤
- `internal/template/derived.go` — 新增 `AllowedDomains` 字段（Domain + admin 邮箱域名去重）
- `internal/template/files/settings.yml.tmpl` — `allowedDomains` 使用 `Derived.AllowedDomains`
- `internal/cli/install.go` — 部署结果使用 `cfg.AdminEmail` 展示
