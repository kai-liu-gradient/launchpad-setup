## 为什么

用户在 builtin 模式下输入 domain 后缀后，系统自动生成 launchpad、gitea 和泛域名，但用户无法在部署前预览和确认这些域名。如果生成的域名不符合预期，用户只能在部署失败后才能发现问题。此外，当前没有校验机制确保 gitea、launchpad 和泛域名使用相同的后缀，后缀不一致会导致证书、DNS、Ingress 等一系列级联故障。

## 变更内容

1. **域名预览确认**：在 builtin 模式下用户填写 domain 后，展示完整的待生成域名列表（launchpad 域名、gitea 域名、泛域名），让用户确认或自定义修改
2. **域名自定义编辑**：用户可以逐条修改生成的域名（如修改 subdomain 前缀），修改后的结果作为最终部署标准
3. **返回修改域名**：用户在预览界面可以选择返回 wizard 重新修改基础域名（domain），无需取消整个安装流程
4. **后缀一致性校验**：新增校验逻辑，确保 launchpad 域名、gitea 域名和泛域名（IngressDomain）共享相同的基础域后缀，不一致时阻止继续并提示用户修正

## 功能 (Capabilities)

### 新增功能
- `domain-preview`: 域名预览与确认交互流程，在安装向导中展示和编辑待生成域名，支持返回 wizard 修改域名
- `domain-suffix-validation`: 域名后缀一致性校验，确保所有域名共享相同后缀

### 修改功能

## 影响

- `internal/cli/install.go` — 安装流程增加 wizard ↔ domain preview 循环，支持 GoBack 回退
- `internal/tui/wizard/domain_preview.go` — 新增域名预览组件，提供 Confirm/Edit/Back 三选项
- `internal/config/config.go` — Config 结构新增 GiteaSubdomain 字段存储自定义域名
- `internal/config/validate.go` — 新增后缀一致性校验规则
- `internal/template/derived.go` — ComputeDerived 支持 GiteaSubdomain 覆盖默认值
