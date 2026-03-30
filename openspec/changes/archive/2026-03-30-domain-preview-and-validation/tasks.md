## 1. Config 层变更

- [x] 1.1 在 `internal/config/config.go` 的 Config 结构中新增 `GiteaSubdomain string` 字段（yaml tag: `giteaSubdomain,omitempty`）
- [x] 1.2 在 `DefaultConfig()` 中设置 `GiteaSubdomain` 默认值为空字符串（空表示使用 `Subdomain + "-gitea"` 默认规则）
- [x] 1.3 在 `internal/config/validate.go` 的 `Validate()` 中新增后缀一致性校验：从 Subdomain+Domain 和 GiteaSubdomain+Domain 中提取后缀，验证它们与 Domain 一致

## 2. ComputeDerived 支持自定义 Gitea subdomain

- [x] 2.1 修改 `internal/template/derived.go` 的 `ComputeDerived()`：当 `cfg.GiteaSubdomain` 非空时，使用 `cfg.GiteaSubdomain + "." + cfg.Domain` 作为 giteaDomain，否则保持原逻辑 `cfg.Subdomain + "-gitea." + cfg.Domain`

## 3. 域名预览 TUI 组件

- [x] 3.1 创建 `internal/tui/wizard/domain_preview.go`，实现域名预览表单：展示 Launchpad 域名、Gitea 域名、泛域名三行，使用 `huh.Select` 提供 Confirm/Edit subdomains/Back 三选项
- [x] 3.2 实现"Edit subdomains"模式：展示可编辑的 Launchpad subdomain 和 Gitea subdomain 输入框，修改后循环回预览
- [x] 3.3 实现"Back (change domain)"模式：返回 `GoBack: true`，通知调用方重新运行 wizard

## 4. 集成到安装向导

- [x] 4.1 修改 `internal/cli/install.go`：在 wizard 和后续流程之间增加 wizard ↔ domain preview 循环，GoBack 时重新运行 wizard
- [x] 4.2 将预览步骤中用户确认的 subdomain 和 giteaSubdomain 值写入 Config
- [x] 4.3 修改部署完成后的 URL 展示，使用 GiteaSubdomain 覆盖值

## 5. 测试

- [x] 5.1 为后缀一致性校验编写单元测试（一致通过、不一致失败、边界情况）
- [x] 5.2 为 ComputeDerived 的 GiteaSubdomain 覆盖逻辑编写单元测试
- [x] 5.3 手动测试 Express 和 Custom 模式的完整域名预览流程（含 Back 回退）
