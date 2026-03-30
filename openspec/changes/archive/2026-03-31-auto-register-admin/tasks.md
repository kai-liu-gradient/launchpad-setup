## 1. Express 模式收集 admin 邮箱

- [x] 1.1 修改 `internal/tui/wizard/express.go`：增加可选的 Admin Email 输入框（密码不暴露）
- [x] 1.2 修改 `internal/tui/wizard/wizard.go`：Express 完成后将 admin 邮箱传递到 Config

## 2. 密码生成满足 API 策略

- [x] 2.1 修改 `internal/secrets/secrets.go`：AdminPassword 单独生成，格式 `randBase64Safe(20) + "!@A1"`

## 3. admin 邮箱域名加入注册白名单

- [x] 3.1 修改 `internal/template/derived.go`：新增 `AllowedDomains` 字段（Domain + admin 邮箱域名去重）
- [x] 3.2 修改 `internal/template/files/settings.yml.tmpl`：`allowedDomains` 使用 `Derived.AllowedDomains`

## 4. 部署步骤：注册 launchpad admin

- [x] 4.1 在 `internal/engine/steps.go` 新增 `registerAdmin` 步骤，插入在 "Waiting for API health" 之后
- [x] 4.2 新增 `internal/engine/admin.go`：通过 Node.js + docker exec 调用 API 注册（`127.0.0.1:6802`），传入 email/password/name/username
- [x] 4.3 处理幂等性：注册返回 409/422/400 时跳过并继续

## 5. 自动邮箱验证

- [x] 5.1 注册成功后通过 docker exec 在 postgres 容器执行 `UPDATE launchpad_main.users SET email_verified = true`
- [x] 5.2 处理 builtin DB（docker exec）和 external DB（psql URL）两种模式

## 6. Domain Preview 展示 admin 邮箱

- [x] 6.1 修改 `internal/tui/wizard/domain_preview.go`：增加 adminEmail 参数，预览中展示 Admin 行

## 7. 部署结果展示

- [x] 7.1 修改 `internal/cli/install.go`：使用 `cfg.AdminEmail`（而非 `"admin@" + cfg.Domain`）
- [x] 7.2 修改 `internal/engine/result.go` 和 `internal/cli/install.go`：醒目展示凭据 + 保存提示

## 8. 测试

- [x] 8.1 构建验证通过
- [x] 8.2 单元测试无回归
- [x] 8.3 手动测试完整部署流程
