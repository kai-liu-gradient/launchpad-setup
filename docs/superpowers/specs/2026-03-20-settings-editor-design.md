# Settings Editor Design Spec

## Goal

Add `launchpad settings` CLI command with an interactive TUI for editing `settings.yml` — the hot-reloadable runtime configuration. Changes take effect immediately without service restarts.

## Context

`settings.yml` controls runtime behavior: registration rules, credential toggles, authentication modes, welcome credits, UI notices, etc. It is mounted into the API container at `/app/config/settings.yml` and the API watches it for changes (hot-reload). Currently it is generated from a Go template (`settings.yml.tmpl`) during initial deployment, but after that it should be edited directly.

This is separate from `launchpad configure` which handles infrastructure config (`.setup.yaml` — SSL, database, SMTP, etc.) and requires service restarts.

## Architecture

### Data Flow

```
settings.yml (on disk) → CLI reads → TUI edits → CLI writes back → API hot-reloads
```

No template rendering, no service restart. The CLI reads and writes the YAML file directly.

### Relationship with configure

`template.RenderAll()` currently renders `settings.yml.tmpl` to `generated/launchpad/config/settings.yml`. After this change:
- If the output file already exists, `RenderAll` skips rendering `settings.yml` (preserves user edits)
- Template only generates the initial file on first deploy
- `launchpad settings` owns the file after initial creation

## Command Structure

```
launchpad settings              # Interactive TUI editor
```

No subcommands needed initially.

## TUI Design

### Layout

Reuses the sidebar + content pattern from `launchpad configure`:

```
  AniLaunchpad Settings

  ▸ Registration       restrictDomain: true
    Plans              allowedDomains: [example.com]
    Credentials        requireEmailVerification: true
    Projects           notice: "Registration is restricted..."
    Authentication
    Welcome Credits    ↑/↓ navigate · enter edit · q quit
    Menu
    YAML Builder
    AI Marketplace
    Ops Report
    Email Suppress
    Notices
```

Left: vertical sidebar listing settings groups (12 items).
Right: summary view of current values for the selected group.

### Interaction

- `↑/↓` or `j/k`: Navigate sidebar
- `enter`: Open huh form to edit the selected group
- `esc` or `ctrl+c` in form: Cancel edit, return to sidebar
- `q`: Quit settings editor
- No "Apply" button or `a` key — unlike configure, edits save immediately on form completion (hot-reload means no restart step)
- Status bar hint: `"↑/↓ navigate · enter edit · q quit"`

### Edit → Save Flow

When user completes a form (not cancelled):
1. Apply form values to the in-memory Settings struct
2. Marshal to YAML and write to disk
3. Return to sidebar with updated summary view
4. API picks up the change via file watch

## Data Model

### Package: `internal/settings`

New package, separate from `internal/config`. Contains the Settings struct and YAML load/save.

```go
type Settings struct {
    Version         int              `yaml:"version"`
    Registration    Registration     `yaml:"registration"`
    Plans           Plans            `yaml:"plans"`
    Credentials     Credentials      `yaml:"credentials"`
    Projects        Projects         `yaml:"projects"`
    Authentication  Authentication   `yaml:"authentication"`
    WelcomeCredits  WelcomeCredits   `yaml:"welcomeCredits"`
    Menu            Menu             `yaml:"menu"`
    YAMLBuilder     YAMLBuilder      `yaml:"yamlBuilder"`
    AIMarketplace   AIMarketplace    `yaml:"aiMarketplace"`
    OpsReport       OpsReport        `yaml:"operationsReport"`
    EmailSuppression EmailSuppression `yaml:"emailSuppression"`
    Notices         Notices          `yaml:"notices"`
}
```

Each sub-struct mirrors the YAML structure exactly. Example:

```go
type Registration struct {
    RestrictDomain bool     `yaml:"restrictDomain"`
    AllowedDomains []string `yaml:"allowedDomains"`
    RequireEmailVerification bool `yaml:"requireEmailVerification"`
    Notice        string   `yaml:"notice"`
    BlockedNotice string   `yaml:"blockedNotice"`
}

type Credentials struct {
    ClaudeIncluded CredentialToggle `yaml:"claudeIncluded"`
    ZAIIncluded    CredentialToggle `yaml:"zaiIncluded"`
    ClaudePaygo    CredentialToggle `yaml:"claudePaygo"`
    ZAIPaygo       CredentialToggle `yaml:"zaiPaygo"`
    GeminiPaygo    CredentialToggle `yaml:"geminiPaygo"`
}

type CredentialToggle struct {
    Disabled bool   `yaml:"disabled"`
    Notice   string `yaml:"notice"`
}

type Authentication struct {
    Mode           string         `yaml:"mode"`
    PasswordLogin  PasswordLogin  `yaml:"passwordLogin"`
    OAuth          OAuthConfig    `yaml:"oauth"`
    ProviderSignup bool           `yaml:"providerSignup"`
    SSOOnlyNotice  string         `yaml:"ssoOnlyNotice"`
}

type PasswordLogin struct {
    Enabled        bool   `yaml:"enabled"`
    EnableSignup   bool   `yaml:"enableSignup"`
    DisabledNotice string `yaml:"disabledNotice"`
}

type OAuthConfig struct {
    Google    OAuthProvider `yaml:"google"`
    GitHub    OAuthProvider `yaml:"github"`
    Microsoft MicrosoftOAuth `yaml:"microsoft"`
}

type OAuthProvider struct {
    Enabled     bool `yaml:"enabled"`
    AllowSignup bool `yaml:"allowSignup"`
}

type MicrosoftOAuth struct {
    Enabled     bool `yaml:"enabled"`
    AllowSignup bool `yaml:"allowSignup"`
    EnforceOnly bool `yaml:"enforceOnly"`
}

type Plans struct {
    EnforceFreePlan bool     `yaml:"enforceFreePlan"`
    Disabled        []string `yaml:"disabled"`
    DisabledNotice  string   `yaml:"disabledNotice"`
}

type Projects struct {
    CreateDisabled       bool   `yaml:"createDisabled"`
    CreateDisabledNotice string `yaml:"createDisabledNotice"`
}

type WelcomeCredits struct {
    Enabled       bool   `yaml:"enabled"`
    Amount        int    `yaml:"amount"`
    CampaignStart string `yaml:"campaignStart"`
    CampaignEnd   string `yaml:"campaignEnd"`
    Message       string `yaml:"message"`
}

type Menu struct {
    Docs MenuDocs `yaml:"docs"`
}

type MenuDocs struct {
    Visible bool `yaml:"visible"`
}

type YAMLBuilder struct {
    AllowedUsers string `yaml:"allowedUsers"`
}

type AIMarketplace struct {
    Visible          bool     `yaml:"visible"`
    EnabledProviders []string `yaml:"enabledProviders"`
}

type OpsReport struct {
    Enabled             bool     `yaml:"enabled"`
    AdminEmails         []string `yaml:"adminEmails"`
    SendTime            string   `yaml:"sendTime"`
    BalanceForecastDays int      `yaml:"balanceForecastDays"`
    TopResourceCount    int      `yaml:"topResourceCount"`
}

type EmailSuppression struct {
    SuppressedRecipients []string `yaml:"suppressedRecipients"`
}

type Notice struct {
    Enabled     bool   `yaml:"enabled"`
    Severity    string `yaml:"severity"`
    Dismissible bool   `yaml:"dismissible"`
    Title       string `yaml:"title"`
    Message     string `yaml:"message"`
    LinkURL     string `yaml:"linkUrl"`
    LinkText    string `yaml:"linkText"`
}

type Notices struct {
    ProjectList Notice `yaml:"projectList"`
    CodingAgent Notice `yaml:"codingAgent"`
    Billing     Notice `yaml:"billing"`
    CreditDrawer Notice `yaml:"creditDrawer"`
}
```

### Load / Save

```go
func Load(path string) (*Settings, error)   // yaml.Unmarshal
func Save(s *Settings, path string) error    // yaml.Marshal + write
```

## TUI Components

### Package: `internal/tui/settings`

Reuses the same patterns as `internal/tui/panel`:
- **SideMenu**: Extract `SideMenu` and `MenuItem` from `internal/tui/panel/tab.go` into `internal/tui/components/sidemenu.go` so both panel and settings can import it.
- **New interface**: Define a `SettingsTab` interface in this package (not reusing `EditableTab` from panel, since that takes `*config.Config`):

```go
type SettingsTab interface {
    View() string
    Form() *huh.Form
    Apply(s *settings.Settings)
}
```

- Each group is a separate file implementing `SettingsTab`.

### Settings Tabs (12 files)

| File | Group | Key Fields |
|------|-------|------------|
| `registration.go` | Registration | restrictDomain, allowedDomains, requireEmailVerification, notice, blockedNotice |
| `plans.go` | Plans | enforceFreePlan, disabled (list), disabledNotice |
| `credentials.go` | Credentials | 5 credential toggles (disabled + notice each) |
| `projects.go` | Projects | createDisabled, createDisabledNotice |
| `authentication.go` | Authentication | mode, passwordLogin, oauth providers, ssoOnlyNotice |
| `welcome_credits.go` | Welcome Credits | enabled, amount, campaignStart/End, message |
| `menu.go` | Menu | docs.visible |
| `yaml_builder.go` | YAML Builder | allowedUsers |
| `ai_marketplace.go` | AI Marketplace | visible, enabledProviders |
| `ops_report.go` | Ops Report | enabled, adminEmails, sendTime, balanceForecastDays, topResourceCount |
| `email_suppression.go` | Email Suppression | suppressedRecipients |
| `notices.go` | Notices | 4 notice blocks (projectList, codingAgent, billing, creditDrawer) |

Each tab follows the same pattern as configure tabs:
- `View()`: formatted summary of current values
- `Form()`: huh form with appropriate field types
- `Apply(*Settings)`: write form values back to Settings struct (note: takes `*Settings` not `*Config`)

### Conditional Fields

- Authentication: oauth provider fields hidden when provider not enabled
- Credentials: each toggle's notice hidden when not disabled
- Notices: each notice's detail fields hidden when not enabled

## CLI Integration

### Package: `internal/cli`

New file `settings.go`:

```go
func newSettingsCmd() *cobra.Command {
    // Use: "settings"
    // RunE: runSettings(".")
}
```

`runSettings(dir)`:
1. Load `generated/launchpad/config/settings.yml`
2. Enter sidebar loop (same exit-and-reenter pattern as configure)
3. On form completion: apply values, save file immediately
4. On quit: exit

### Register in root command

Add `newSettingsCmd()` to root command alongside `newConfigureCmd()`.

## Template Render Skip

In `internal/template/render.go`, inside the `RenderAll` function, after `outName` is resolved (line 58-61) and before `outPath` is constructed (line 63), add a skip check:

```go
outName := outputNameMap[name]
if outName == "" {
    outName = strings.TrimSuffix(name, ".tmpl")
}

// Skip settings.yml if it already exists (preserve user edits from launchpad settings)
if outName == "launchpad/config/settings.yml" {
    outPath := filepath.Join(outDir, outName)
    if _, err := os.Stat(outPath); err == nil {
        return nil // file exists, skip rendering
    }
}

outPath := filepath.Join(outDir, outName)
```

This ensures both `configure apply` and initial deploy's `RenderAll` don't overwrite user's settings edits. The variable is `outName` (not `outputName`), and we use `return nil` (not `continue`) since we're inside a `WalkDir` callback.

## File Structure Summary

New files:
- `internal/settings/settings.go` — Settings struct, Load, Save (with yaml.Node round-trip)
- `internal/tui/components/sidemenu.go` — Extracted SideMenu + MenuItem (shared by panel and settings)
- `internal/tui/settings/panel.go` — Settings panel Model (sidebar + content), SettingsTab interface
- `internal/tui/settings/tabs/registration.go` — through `notices.go` (12 tab files)
- `internal/tui/settings/tabs/helpers.go` — shared display helpers (can import from panel/tabs if identical)
- `internal/cli/settings.go` — CLI command

Modified files:
- `internal/cli/root.go` — register settings command
- `internal/template/render.go` — skip settings.yml if exists
- `internal/tui/panel/tab.go` — remove SideMenu/MenuItem (moved to components)
- `internal/tui/panel/panel.go` — import SideMenu from components

## Edge Cases

- **File not found**: If `settings.yml` doesn't exist (pre-deploy), show error: "No settings.yml found. Run initial deployment first."
- **Invalid YAML**: If parse fails, show error with details and exit.
- **Concurrent edits**: CLI reads → edits → writes atomically (write to temp file in same directory, then `os.Rename` for atomic same-filesystem swap). No locking needed since single-user CLI.
- **Unknown fields**: Use `yaml.Node` based round-trip to preserve unknown keys. Load the file into both a typed `Settings` struct (for editing) and a raw `yaml.Node` tree. On save, merge edited fields back into the `yaml.Node` tree and marshal that. This ensures fields added by future API versions are not dropped during round-trip.
- **Version mismatch**: The `Version` field is informational. CLI reads and preserves whatever version is in the file. No migration or validation on version number.
