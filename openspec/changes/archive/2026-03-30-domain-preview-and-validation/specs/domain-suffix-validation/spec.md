## ADDED Requirements

### 需求:后缀一致性校验
系统必须校验 Launchpad 域名、Gitea 域名和泛域名（IngressDomain）共享相同的基础域后缀。后缀不一致时必须阻止继续并提示用户修正。

#### 场景:后缀一致通过校验
- **当** Launchpad 域名为 `launchpad.example.com`，Gitea 域名为 `launchpad-gitea.example.com`，泛域名为 `*.example.com`
- **那么** 校验必须通过，允许继续安装流程

#### 场景:后缀不一致阻止继续
- **当** 用户通过配置文件或其他方式设置了不同后缀的域名（如 Launchpad 为 `app.example.com`，Gitea 为 `git.other.com`）
- **那么** 系统必须返回校验错误，明确指出哪些域名后缀不一致，禁止继续安装

#### 场景:TUI 中实时校验
- **当** 用户在域名编辑模式中修改域名后确认
- **那么** 系统必须立即执行后缀一致性校验，不一致时在 TUI 中展示错误信息并要求修正

### 需求:后缀提取逻辑
系统必须从完整域名中正确提取基础域后缀用于比较。后缀定义为域名中最后两段（如 `example.com`），对于国家级域名则为最后三段（如 `co.uk`）。

#### 场景:标准二级域名后缀提取
- **当** 域名为 `launchpad.example.com`
- **那么** 提取的后缀必须为 `example.com`

#### 场景:多级子域名后缀提取
- **当** 域名为 `app.dev.example.com`
- **那么** 提取的后缀必须为 `example.com`（与 Config.Domain 字段一致）
