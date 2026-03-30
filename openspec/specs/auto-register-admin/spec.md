## ADDED Requirements

### 需求:Express 模式收集 admin 邮箱
Express 安装向导必须提供可选的 Admin Email 输入框，留空时使用默认值 `admin@{domain}`。密码全自动生成，不提供输入。

#### 场景:用户输入自定义 admin 邮箱
- **当** 用户在 Express 模式中输入了 admin 邮箱（如 `kai@gmail.com`）
- **那么** 系统必须使用该邮箱作为 admin 用户的邮箱

#### 场景:用户不输入 admin 邮箱
- **当** 用户未修改 admin 邮箱（留空）
- **那么** 系统必须使用 `admin@{domain}` 作为 admin 邮箱

### 需求:admin 邮箱域名自动加入注册白名单
当 admin 邮箱域名与 Config.Domain 不同时，系统必须自动将该域名加入 settings.yml 的 `allowedDomains`。

#### 场景:admin 邮箱域名与 Domain 不同
- **当** Domain 为 `corp.kola.com`，admin 邮箱为 `admin@gmail.com`
- **那么** settings.yml 的 `allowedDomains` 必须包含 `["corp.kola.com", "gmail.com"]`

#### 场景:admin 邮箱域名与 Domain 相同
- **当** Domain 为 `corp.kola.com`，admin 邮箱为 `admin@corp.kola.com`
- **那么** settings.yml 的 `allowedDomains` 必须只包含 `["corp.kola.com"]`（不重复）

### 需求:Gitea 和 launchpad 统一 admin 凭据
Gitea admin 用户和 launchpad admin 用户必须使用同一套邮箱和自动生成的密码。

#### 场景:凭据一致性
- **当** 部署完成
- **那么** Gitea admin 和 launchpad admin 必须使用相同的邮箱（`AdminEmail`）和密码（`AdminPassword`）登录

### 需求:密码满足 API 密码策略
自动生成的 AdminPassword 必须满足 launchpad API 的密码策略（包含特殊字符、大写字母、数字）。

#### 场景:密码格式
- **当** 系统生成 AdminPassword
- **那么** 密码必须包含至少一个特殊字符、一个大写字母和一个数字

### 需求:部署流程自动注册 launchpad admin
部署流程必须在 API 服务健康后自动注册 admin 用户到 launchpad 平台。

#### 场景:成功注册 admin
- **当** API 服务健康检查通过
- **那么** 系统必须通过 API 注册 admin 用户（使用 AdminEmail、AdminPassword 和从邮箱提取的 username）
- **并且** 注册完成后 admin 用户必须能直接登录

#### 场景:admin 用户已存在（重部署）
- **当** 重部署时 admin 用户已存在
- **那么** 注册步骤必须跳过并继续，不报错

### 需求:自动通过邮箱验证
注册 admin 用户后，系统必须自动标记其邮箱为已验证。

#### 场景:注册后邮箱自动验证
- **当** admin 用户注册成功
- **那么** 系统必须通过 SQL 更新 `launchpad_main.users` 表将邮箱标记为已验证
- **并且** 用户登录后不会看到邮箱未验证提示

### 需求:Domain Preview 展示 admin 邮箱
域名预览界面必须展示当前配置的 admin 邮箱。

#### 场景:展示用户输入的邮箱
- **当** 用户在 Express 中输入了 `kai@gmail.com`
- **那么** Domain Preview 必须展示 `Admin: kai@gmail.com`

#### 场景:展示默认邮箱
- **当** 用户未输入 admin 邮箱
- **那么** Domain Preview 必须展示 `Admin: admin@{domain}`

### 需求:部署结果展示并提示保存 admin 凭据
部署完成后必须醒目展示 admin 账号密码（使用 `cfg.AdminEmail` 而非硬编码），提示用户务必记住，并告知后续查找方式。

#### 场景:部署完成展示凭据
- **当** 部署流程完成
- **那么** 必须展示用户配置的 admin 邮箱和自动生成的密码
- **并且** 必须包含醒目提示："Admin Credentials (SAVE THESE!)"
- **并且** 必须告知密码存储位置：`.secrets.yaml` 文件中的 `admin_password` 字段
