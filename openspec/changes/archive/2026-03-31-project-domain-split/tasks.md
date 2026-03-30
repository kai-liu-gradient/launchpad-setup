## 1. Config 层变更

- [x] 1.1 在 `internal/config/config.go` 的 Config 结构中新增 `ProjectDomain string` 字段（yaml tag: `project_domain,omitempty`，validate tag: `omitempty,fqdn`）
- [x] 1.2 在 `internal/config/config.go` 新增 `ResolvedProjectDomain()` 方法：非空返回 `ProjectDomain`，否则返回 `Domain`
- [x] 1.3 在 `internal/config/validate.go` 新增校验：ProjectDomain 非空时必须是 Domain 的父域或相同（Domain 以 `.ProjectDomain` 结尾，或 Domain == ProjectDomain）
- [x] 1.4 调整现有的后缀一致性校验逻辑，适配 ProjectDomain 场景

## 2. Derived 值分离

- [x] 2.1 修改 `internal/template/derived.go` 的 `ComputeDerived()`：使用 `cfg.ResolvedProjectDomain()` 设置 `IngressDomain` 和 `DomainEscaped`
- [x] 2.2 确保 `ExternalDomain` 使用 resolved ProjectDomain
- [x] 2.3 为 ComputeDerived 的 ProjectDomain 逻辑编写单元测试

## 3. CoreDNS 模板适配

- [x] 3.1 修改 `internal/template/files/coredns-custom.yaml.tmpl`：wildcard zone 使用 `Derived.IngressDomain` 和 `Derived.DomainEscaped`（替换 `Config.Domain`）
- [x] 3.2 验证 specific zone（launchpad/gitea）继续使用 `Derived.LaunchpadDomain` / `Derived.GiteaDomain`，不受影响

## 4. SSL 证书适配

- [x] 4.1 修改 `internal/engine/certs.go` 的 `generateSelfsignedCerts()`：SAN 中增加 `*.ResolvedProjectDomain()` 和裸域（当与 Domain 不同时）
- [x] 4.2 去重逻辑：当 ProjectDomain 为空或等于 Domain 时不重复添加 SAN 条目

## 5. dnsmasq 适配

- [x] 5.1 修改 `internal/engine/dnsmasq.go`：当 ProjectDomain 与 Domain 不同时，生成两条 `address=` 规则

## 6. /etc/hosts 适配

- [x] 6.1 修改 `internal/engine/hosts.go`：确保 /etc/hosts 条目使用平台域名（Domain），不受 ProjectDomain 影响

## 7. TUI 适配

- [x] 7.1 修改 `internal/tui/wizard/domain_preview.go`：预览中泛域名行使用 resolved ProjectDomain（`\*.ProjectDomain`）
- [x] 7.2 修改 `internal/tui/panel/tabs/basic.go`：Custom 模式增加 Project Domain 输入框
- [x] 7.3 修改 `internal/tui/wizard/express.go`：Express 模式不暴露 ProjectDomain（默认为空）

## 8. 测试

- [x] 8.1 为 ProjectDomain 校验编写单元测试（父域通过、相同通过、无关域失败、空值通过）
- [x] 8.2 为 CoreDNS 模板渲染编写测试（验证 zone 域正确）
- [x] 8.3 手动测试 Express 和 Custom 模式的完整流程
