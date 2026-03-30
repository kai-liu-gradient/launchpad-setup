## ADDED Requirements

### 需求:域名预览展示
在 builtin 模式下用户输入 domain 后缀后，系统必须展示完整的待生成域名列表，包括 Launchpad 域名、Gitea 域名和泛域名（Ingress Domain）。

#### 场景:Express 模式域名预览
- **当** 用户在 Express 模式中输入 domain（如 `example.com`）并提交
- **那么** 系统必须展示域名预览列表，包含：Launchpad 域名（`launchpad.example.com`）、Gitea 域名（`launchpad-gitea.example.com`）、泛域名（`*.example.com`）

#### 场景:Custom 模式域名预览
- **当** 用户在 Custom 模式 BasicTab 中输入 domain 并提交
- **那么** 系统必须展示相同格式的域名预览列表

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

### 需求:域名自定义编辑
用户选择修改时，必须能够自定义 Launchpad 和 Gitea 的 subdomain 前缀。修改后系统必须实时更新预览。

#### 场景:修改 Launchpad subdomain
- **当** 用户将 Launchpad subdomain 从 `launchpad` 修改为 `app`
- **那么** 预览中 Launchpad 域名必须更新为 `app.example.com`，Gitea 域名默认联动更新为 `app-gitea.example.com`

#### 场景:独立修改 Gitea subdomain
- **当** 用户将 Gitea subdomain 修改为 `git`
- **那么** 预览中 Gitea 域名必须更新为 `git.example.com`，Launchpad 域名保持不变

#### 场景:修改后确认
- **当** 用户完成域名修改并确认
- **那么** 系统必须使用用户修改后的域名作为最终部署标准
