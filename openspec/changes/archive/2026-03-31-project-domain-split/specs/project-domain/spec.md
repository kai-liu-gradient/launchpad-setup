## ADDED Requirements

### 需求:Project Domain 配置
系统必须支持通过 `ProjectDomain` 字段配置独立的项目泛域名基础域。该字段为可选，为空时必须退化为 `Domain` 的值。

#### 场景:ProjectDomain 为空时使用 Domain
- **当** 配置中 `ProjectDomain` 为空
- **那么** 系统必须使用 `Domain` 作为项目泛域名基础域，行为与当前完全一致

#### 场景:ProjectDomain 与 Domain 不同
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `example.com`
- **那么** 平台域名必须基于 `corp.example.com`（如 `launchpad.corp.example.com`），项目泛域名必须基于 `example.com`（如 `*.example.com`）

### 需求:ProjectDomain 校验
系统必须校验 `ProjectDomain`（非空时）为合法 FQDN 格式，且必须是 `Domain` 的父域或与 `Domain` 相同。

#### 场景:ProjectDomain 是 Domain 的父域
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `example.com`
- **那么** 校验必须通过

#### 场景:ProjectDomain 与 Domain 相同
- **当** `Domain` 为 `example.com`，`ProjectDomain` 为 `example.com`
- **那么** 校验必须通过

#### 场景:ProjectDomain 与 Domain 无关
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `other.com`
- **那么** 校验必须失败并返回错误信息

### 需求:CoreDNS 泛域名 zone 使用 ProjectDomain
CoreDNS 的 wildcard server zone 必须使用 resolved ProjectDomain（而非 Config.Domain）作为泛域名匹配域。

#### 场景:Domain 与 ProjectDomain 不同时的 CoreDNS zone
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `example.com`
- **那么** CoreDNS specific zone 必须处理 `launchpad.corp.example.com` 和 `launchpad-gitea.corp.example.com`，wildcard zone 必须处理 `*.example.com`

### 需求:SSL 证书覆盖两个域
selfsigned 模式下，生成的 SSL 证书 SAN 必须同时包含 `*.Domain` 和 `*.ProjectDomain`（当两者不同时）。

#### 场景:两个域不同时的证书 SAN
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `example.com`
- **那么** 证书 SAN 必须包含 `*.corp.example.com`、`corp.example.com`、`*.example.com`、`example.com`

#### 场景:两个域相同时的证书 SAN
- **当** `Domain` 为 `example.com`，`ProjectDomain` 为空
- **那么** 证书 SAN 必须包含 `*.example.com` 和 `example.com`（不重复）

### 需求:dnsmasq 覆盖两个域
selfsigned 模式下，dnsmasq 配置必须同时为 Domain 和 ProjectDomain 添加地址解析规则。

#### 场景:两个域不同时的 dnsmasq 配置
- **当** `Domain` 为 `corp.example.com`，`ProjectDomain` 为 `example.com`
- **那么** dnsmasq 配置必须同时包含 `address=/corp.example.com/{hostIP}` 和 `address=/example.com/{hostIP}`
