## 为什么

当前系统假设平台域名（launchpad、gitea）和项目泛域名（`*.domain`）共享同一个基础域。但在企业场景中，平台可能部署在 `launchpad.corp.example.com`，而项目 pod 使用 `*.example.com` 作为泛域名。这两个域后缀不同，当前架构无法表达这种分离，导致 CoreDNS zone、SSL 证书 SAN、Ingress 路由都会用错误的域。

## 变更内容

1. **新增 Project Domain 配置字段**：在 Config 中新增 `ProjectDomain` 字段（可选），用于指定项目泛域名的基础域。为空时退化为 `Domain`，向后兼容。
2. **Domain Preview 适配**：预览界面展示平台域名和项目泛域名，当两者不同时明确区分显示。
3. **CoreDNS 模板分离**：wildcard zone 使用 `ProjectDomain` 而非 `Config.Domain`，specific zone 继续使用平台域名。
4. **SSL 证书 SAN 覆盖两个域**：selfsigned 模式下生成的证书 SAN 同时包含 `*.Domain` 和 `*.ProjectDomain`。
5. **Derived 值分离**：`IngressDomain`、`DomainEscaped` 等泛域名相关值使用 `ProjectDomain`。

## 功能 (Capabilities)

### 新增功能
- `project-domain`: 独立的项目泛域名配置，支持平台域名与项目域名使用不同的基础域后缀

### 修改功能
- `domain-preview`: 预览界面需要展示 Project Domain（当与 Domain 不同时）

## 影响

- `internal/config/config.go` — Config 结构新增 `ProjectDomain` 字段
- `internal/config/validate.go` — 校验 ProjectDomain 格式、后缀一致性调整
- `internal/template/derived.go` — `IngressDomain`、`DomainEscaped` 使用 ProjectDomain
- `internal/template/files/coredns-custom.yaml.tmpl` — wildcard zone 使用 ProjectDomain
- `internal/engine/certs.go` — 证书 SAN 覆盖两个域
- `internal/engine/dnsmasq.go` — dnsmasq 配置适配两个域
- `internal/tui/wizard/domain_preview.go` — 预览展示 Project Domain
- `internal/tui/panel/tabs/basic.go` — Custom 模式增加 Project Domain 输入
