## MODIFIED Requirements

### 需求:域名预览展示
在安装向导中用户输入 domain 后，系统必须展示完整的待生成域名列表，包括 Launchpad 域名、Gitea 域名和项目泛域名。当 ProjectDomain 与 Domain 不同时，必须明确区分展示。

#### 场景:ProjectDomain 与 Domain 相同
- **当** 用户输入 domain 为 `example.com`，ProjectDomain 为空
- **那么** 预览必须展示：Launchpad 域名（`launchpad.example.com`）、Gitea 域名（`launchpad-gitea.example.com`）、泛域名（`*.example.com`）

#### 场景:ProjectDomain 与 Domain 不同
- **当** 用户输入 domain 为 `corp.example.com`，ProjectDomain 为 `example.com`
- **那么** 预览必须展示：Launchpad 域名（`launchpad.corp.example.com`）、Gitea 域名（`launchpad-gitea.corp.example.com`）、Project Domain（`*.example.com`）

### 需求:域名确认操作
用户查看域名预览后，必须能够选择"Confirm"直接使用、"Edit subdomains"修改前缀、或"Back"返回 wizard 修改基础域名。

#### 场景:用户确认域名
- **当** 用户查看域名预览后选择"Confirm"
- **那么** 系统必须使用展示的域名作为部署标准，继续后续安装流程

#### 场景:用户选择修改 subdomain
- **当** 用户查看域名预览后选择"Edit subdomains"
- **那么** 系统必须展示可编辑的 subdomain 输入框，允许用户修改 Launchpad subdomain 和 Gitea subdomain
- **并且** 修改完成后必须循环回预览页面展示更新后的域名

#### 场景:用户选择返回修改域名
- **当** 用户查看域名预览后选择"Back (change domain)"
- **那么** 系统必须重新运行安装向导（wizard），让用户重新输入基础域名
- **并且** 用户在 wizard 中确认后必须再次进入域名预览
