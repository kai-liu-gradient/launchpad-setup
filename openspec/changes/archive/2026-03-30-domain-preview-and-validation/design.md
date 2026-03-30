## 上下文

当前安装向导（Express 和 Custom 模式）只要求用户输入基础域名（如 `example.com`），subdomain 固定为 `launchpad`。域名生成逻辑在 `internal/template/derived.go:64-66` 中硬编码：

- `launchpadDomain = subdomain + "." + domain`（如 `launchpad.example.com`）
- `giteaDomain = subdomain + "-gitea." + domain`（如 `launchpad-gitea.example.com`）
- `ingressDomain = domain`（用于泛域名证书 `*.example.com`）

用户在部署前无法看到这些衍生域名，也没有后缀一致性校验。

## 目标 / 非目标

**目标：**
- 在 domain 输入后、部署前展示域名预览列表供用户确认
- 允许用户自定义修改 subdomain 部分
- 允许用户返回 wizard 重新修改基础域名（domain）
- 校验三个关键域名（launchpad、gitea、泛域名）后缀一致
- 对 Express 和 Custom 两种模式都生效

**非目标：**
- 不改变域名的底层生成逻辑（仍基于 subdomain + domain 模式）
- 不支持完全自由输入域名（保持 subdomain.domain 格式约束）
- 不涉及 DNS、证书生成等下游逻辑变更

## 决策

### 1. 在 TUI 向导中新增域名预览步骤

域名输入完成后，插入一个新的 `huh.Form` 步骤展示预览：

```
━━ Domain Preview ━━
  Launchpad:  launchpad.example.com
  Gitea:      launchpad-gitea.example.com
  Wildcard:   *.example.com

  > Confirm
    Edit subdomains
    Back (change domain)
```

**选择 huh.Select 三选项而不是 huh.Confirm 二选项**：Confirm 只支持"确认/编辑"两个操作，无法表达"返回修改域名"。使用 Select 提供 Confirm/Edit subdomains/Back 三个选项，操作语义清晰。

### 2. 修改模式：编辑 subdomain 而非完整域名

用户选择"Edit subdomains"时，展示可编辑的 subdomain 输入框。域名后缀（domain）保持锁定，避免后缀不一致。

**选择限制 subdomain 编辑而非自由编辑**：自由编辑会极大增加校验复杂度，且当前架构强依赖 `subdomain + domain` 模式。限制编辑范围是最安全的方案。

### 3. 返回修改域名：wizard ↔ preview 循环

用户选择"Back (change domain)"时，`RunDomainPreview` 返回 `GoBack: true`，调用方重新运行 wizard，用户可以修改 domain 后再次进入预览。

```go
// install.go 中的循环
for {
    result := runWizard(mode)
    preview := RunDomainPreview(result.Domain, result.Subdomain)
    if preview.GoBack {
        continue // 重新运行 wizard
    }
    // 使用确认后的结果继续部署
    break
}
```

**选择在 install.go 层实现循环而非在 preview 内部回调 wizard**：保持组件职责单一，preview 只负责预览和选择，不感知 wizard 的存在。

### 4. 后缀校验在 Config.Validate() 中实现

在 `internal/config/validate.go` 的 `Validate()` 函数中新增校验规则：从 launchpad 域名、gitea 域名和 ingress domain 中提取后缀，检查是否一致。

**选择在 validate.go 而非 TUI 层校验**：校验逻辑作为 Config 层的约束，无论是 TUI、CLI 参数还是配置文件导入，都能统一生效。

### 5. Config 结构新增可选的域名覆盖字段

在 `Config` 中新增 `GiteaSubdomain` 字段（默认为 `subdomain + "-gitea"`）。当用户在预览步骤中修改 gitea 域名时，更新此字段。`ComputeDerived` 优先使用此覆盖值。

```go
type Config struct {
    Domain         string `yaml:"domain" validate:"required,fqdn"`
    Subdomain      string `yaml:"subdomain"`
    GiteaSubdomain string `yaml:"giteaSubdomain,omitempty"` // 新增
    // ...
}
```

**选择新增 GiteaSubdomain 而非 GiteaDomain**：保持 `subdomain + "." + domain` 的生成模式，只允许自定义前缀部分，架构改动最小。

## 风险 / 权衡

- **Express 模式交互增加** → 新增一个确认步骤，但默认可直接确认跳过，不影响快速部署体验
- **配置文件兼容性** → `GiteaSubdomain` 为可选字段，旧配置无此字段时自动使用默认值 `subdomain + "-gitea"`，向后兼容
- **域名修改后的级联影响** → 修改 subdomain 后，证书 SAN、DNS、Nginx 配置等都会随之变化，这些都通过 `ComputeDerived` 统一驱动，无需额外处理
- **Back 循环不保留 wizard 状态** → 返回后 wizard 重新初始化，用户需要重新输入所有配置。对于 Express 模式（只有一个 domain 字段）影响很小；Custom 模式用户可能需要重新配置，但这是合理的代价（修改 domain 通常意味着整体配置都需要调整）
