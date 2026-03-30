## 上下文

当前系统使用单一 `Config.Domain` 字段驱动所有域名生成：

- 平台域名：`subdomain.domain`（如 `launchpad.example.com`）
- Gitea 域名：`subdomain-gitea.domain`（如 `launchpad-gitea.example.com`）
- 项目泛域名：`*.domain`（如 `*.example.com`）
- CoreDNS wildcard zone：`domain:53`
- SSL 证书 SAN：`*.domain`

当企业用户需要平台部署在子域（如 `corp.example.com`）而项目泛域名保持在根域（`*.example.com`）时，这个假设不成立。

相关文件：
- `internal/config/config.go` — Config 结构
- `internal/template/derived.go` — 域名派生逻辑
- `internal/template/files/coredns-custom.yaml.tmpl` — CoreDNS zone 模板
- `internal/engine/certs.go` — SSL 证书 SAN 生成
- `internal/engine/dnsmasq.go` — dnsmasq 配置
- `internal/tui/wizard/domain_preview.go` — 域名预览

## 目标 / 非目标

**目标：**
- 支持平台域名和项目泛域名使用不同的基础域后缀
- `ProjectDomain` 为可选字段，为空时退化为 `Domain`，完全向后兼容
- CoreDNS、SSL、dnsmasq 等下游组件正确处理两个域
- TUI 预览清晰展示两种域名

**非目标：**
- 不支持多个项目泛域名（只有一个 ProjectDomain）
- 不改变项目 pod 的 Ingress 路由逻辑（仍然基于泛域名 + hostname）
- 不涉及 DNS 托管/公网解析（这是用户的责任）

## 决策

### 1. Config 新增 `ProjectDomain` 字段

```go
type Config struct {
    Domain         string `yaml:"domain" validate:"required,fqdn"`
    ProjectDomain  string `yaml:"project_domain,omitempty" validate:"omitempty,fqdn"`
    Subdomain      string `yaml:"subdomain"`
    GiteaSubdomain string `yaml:"giteaSubdomain,omitempty"`
    // ...
}
```

**选择新增独立字段而非复用 Domain**：语义清晰，`Domain` 是平台基准域，`ProjectDomain` 是项目泛域名域。为空时 `resolvedProjectDomain()` 返回 `Domain`，一处兼容逻辑，其他代码统一使用 resolved 值。

### 2. Derived 中使用 resolved ProjectDomain

```go
func ComputeDerived(...) {
    projectDomain := cfg.ProjectDomain
    if projectDomain == "" {
        projectDomain = cfg.Domain
    }
    d.IngressDomain = projectDomain
    d.DomainEscaped = strings.ReplaceAll(projectDomain, ".", "\\.")
}
```

平台域名（LaunchpadDomain、GiteaDomain）继续使用 `cfg.Domain`，泛域名相关值使用 `projectDomain`。

**选择在 Derived 层解析而非在每个消费方分别判断**：单点解析，所有模板和下游代码通过 `Derived.IngressDomain` 访问，无需关心 ProjectDomain 是否为空。

### 3. CoreDNS 模板适配

模板已经分离为 `launchpad-specific.server` 和 `wildcard.server` 两个 zone。只需将 wildcard zone 的域名从 `Config.Domain` 改为 `Derived.IngressDomain`（即 resolved ProjectDomain）。

当 Domain 和 ProjectDomain 不同时（如 `corp.example.com` vs `example.com`），两个 zone 的域后缀不同，CoreDNS 按最具体匹配自动路由，无需额外逻辑。

### 4. SSL 证书同时覆盖两个域

selfsigned 模式下，SAN 需要包含：
- `*.Domain`（覆盖平台域名）
- `*.ProjectDomain`（覆盖项目泛域名）
- 以及各自的裸域

当两者相同时自动去重。

### 5. TUI 展示

Domain Preview 增加 Project Domain 行（仅当不同于 Domain 时显示）：

```
  Launchpad:       launchpad.corp.example.com
  Gitea:           launchpad-gitea.corp.example.com
  Project Domain:  *.example.com
```

Express 模式下 ProjectDomain 默认为空（等于 Domain），大多数用户不需要关心。Custom 模式的 Basic Tab 增加 Project Domain 输入框。

### 6. 校验逻辑调整

ProjectDomain 必须是 Domain 的父域或相同。例如：
- `Domain=corp.example.com`, `ProjectDomain=example.com` ✓
- `Domain=example.com`, `ProjectDomain=example.com` ✓
- `Domain=corp.example.com`, `ProjectDomain=other.com` ✗

这确保泛域名证书 `*.ProjectDomain` 的范围总是包含平台域名所在的域树。

## 风险 / 权衡

- **证书复杂度增加** → selfsigned 模式下 SAN 列表变长，但仍是单张证书。letsencrypt 模式需要用户自行确保两个域都有解析，不在本次范围内。
- **向后兼容** → `ProjectDomain` 为空时所有行为与当前完全一致，旧配置文件无需修改。
- **Express 模式无感知** → Express 不暴露 ProjectDomain 字段，只有 Custom 模式支持配置，避免增加新用户认知负担。
- **dnsmasq 需要处理两个域** → 当 Domain ≠ ProjectDomain 时需要两条 `address=` 规则，改动量小。
