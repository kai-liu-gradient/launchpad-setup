# Settings Editor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `launchpad settings` CLI command with interactive TUI for editing the hot-reloadable `settings.yml` runtime configuration.

**Architecture:** Standalone `launchpad settings` command reads/writes `generated/launchpad/config/settings.yml` directly. Sidebar + content TUI (same pattern as configure). No restart needed — API hot-reloads on file change. Template rendering skips settings.yml if it already exists.

**Tech Stack:** Go, charmbracelet/huh (forms), charmbracelet/bubbletea (TUI), charmbracelet/lipgloss (styling), gopkg.in/yaml.v3 (YAML), spf13/cobra (CLI)

**Spec:** `docs/superpowers/specs/2026-03-20-settings-editor-design.md`

---

## Chunk 1: Foundation

### Task 1: Settings Data Model

**Files:**
- Create: `internal/settings/settings.go`

All struct types for settings.yml, plus Load/Save functions.

- [ ] **Step 1: Create settings package with all struct types**

```go
// internal/settings/settings.go
package settings

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Version          int              `yaml:"version"`
	Registration     Registration     `yaml:"registration"`
	Plans            Plans            `yaml:"plans"`
	Credentials      Credentials      `yaml:"credentials"`
	Projects         Projects         `yaml:"projects"`
	Authentication   Authentication   `yaml:"authentication"`
	WelcomeCredits   WelcomeCredits   `yaml:"welcomeCredits"`
	Menu             Menu             `yaml:"menu"`
	YAMLBuilder      YAMLBuilder      `yaml:"yamlBuilder"`
	AIMarketplace    AIMarketplace    `yaml:"aiMarketplace"`
	OpsReport        OpsReport        `yaml:"operationsReport"`
	EmailSuppression EmailSuppression `yaml:"emailSuppression"`
	Notices          Notices          `yaml:"notices"`
}

type Registration struct {
	RestrictDomain           bool     `yaml:"restrictDomain"`
	AllowedDomains           []string `yaml:"allowedDomains"`
	RequireEmailVerification bool     `yaml:"requireEmailVerification"`
	Notice                   string   `yaml:"notice"`
	BlockedNotice            string   `yaml:"blockedNotice"`
}

type Plans struct {
	EnforceFreePlan bool     `yaml:"enforceFreePlan"`
	Disabled        []string `yaml:"disabled"`
	DisabledNotice  string   `yaml:"disabledNotice"`
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

type Projects struct {
	CreateDisabled       bool   `yaml:"createDisabled"`
	CreateDisabledNotice string `yaml:"createDisabledNotice"`
}

type Authentication struct {
	Mode           string        `yaml:"mode"`
	PasswordLogin  PasswordLogin `yaml:"passwordLogin"`
	OAuth          OAuthConfig   `yaml:"oauth"`
	ProviderSignup bool          `yaml:"providerSignup"`
	SSOOnlyNotice  string        `yaml:"ssoOnlyNotice"`
}

type PasswordLogin struct {
	Enabled        bool   `yaml:"enabled"`
	EnableSignup   bool   `yaml:"enableSignup"`
	DisabledNotice string `yaml:"disabledNotice"`
}

type OAuthConfig struct {
	Google    OAuthProvider  `yaml:"google"`
	GitHub    OAuthProvider  `yaml:"github"`
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
	ProjectList  Notice `yaml:"projectList"`
	CodingAgent  Notice `yaml:"codingAgent"`
	Billing      Notice `yaml:"billing"`
	CreditDrawer Notice `yaml:"creditDrawer"`
}

// Load reads a settings.yml file and returns a Settings struct.
func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the Settings struct to a YAML file atomically.
func Save(s *Settings, path string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
```

- [ ] **Step 2: Ensure yaml.v3 dependency is available**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go mod tidy`

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/settings/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/settings/settings.go go.mod go.sum
git commit -m "feat: add settings data model with Load/Save"
```

---

### Task 2: Extract SideMenu to Shared Components

**Files:**
- Create: `internal/tui/components/sidemenu.go`
- Modify: `internal/tui/panel/tab.go`
- Modify: `internal/tui/panel/panel.go`

Move `SideMenu` and `MenuItem` from `panel/tab.go` into `components/sidemenu.go` so both panel and settings can use them.

- [ ] **Step 1: Create `internal/tui/components/sidemenu.go`**

Copy the existing `SideMenu`, `MenuItem`, `Next()`, `Prev()`, `Render()` from `internal/tui/panel/tab.go` into the new file, changing the package to `components`:

```go
// internal/tui/components/sidemenu.go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type MenuItem struct {
	Name string
}

type SideMenu struct {
	Items  []MenuItem
	Active int
}

func (m *SideMenu) Next() {
	m.Active = (m.Active + 1) % len(m.Items)
}

func (m *SideMenu) Prev() {
	m.Active = (m.Active - 1 + len(m.Items)) % len(m.Items)
}

func (m *SideMenu) Render() string {
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1).
		Width(16)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1).
		Width(16)

	var lines []string
	for i, item := range m.Items {
		prefix := "  "
		if i == m.Active {
			prefix = "▸ "
			lines = append(lines, activeStyle.Render(fmt.Sprintf("%s%s", prefix, item.Name)))
		} else {
			lines = append(lines, inactiveStyle.Render(fmt.Sprintf("%s%s", prefix, item.Name)))
		}
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 2: Replace `internal/tui/panel/tab.go` with type aliases**

Replace the entire file content with imports from components:

```go
package panel

import "github.com/gradient8/launchpad/internal/tui/components"

// Type aliases so existing panel code doesn't need changes.
type MenuItem = components.MenuItem
type SideMenu = components.SideMenu
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/...`
Expected: no errors. The panel package still works because `MenuItem` and `SideMenu` are aliased.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/sidemenu.go internal/tui/panel/tab.go
git commit -m "refactor: extract SideMenu to shared components"
```

---

### Task 3: Settings Panel TUI

**Files:**
- Create: `internal/tui/settings/panel.go`
- Create: `internal/tui/settings/tabs/helpers.go`

The panel Model, SettingsTab interface, and the main TUI loop. This task creates the shell — tabs will be added in subsequent tasks.

- [ ] **Step 1: Create helpers**

```go
// internal/tui/settings/tabs/helpers.go
package tabs

import "fmt"

func boolDisplay(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func displayValue(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

func listDisplay(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("%v", items)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

- [ ] **Step 2: Create settings panel**

```go
// internal/tui/settings/panel.go
package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/tui/components"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type ExitReason int

const (
	ExitQuit ExitReason = iota
	ExitEdit
)

// SettingsTab is the interface each settings group implements.
type SettingsTab interface {
	View() string
	Form() *huh.Form
	Apply(s *settingsmod.Settings)
}

type Model struct {
	menu       components.SideMenu
	settings   *settingsmod.Settings
	Tabs       []SettingsTab
	exitReason ExitReason
}

func New(s *settingsmod.Settings, activeItem int) Model {
	items := []components.MenuItem{
		{Name: "Registration"},
		{Name: "Plans"},
		{Name: "Credentials"},
		{Name: "Projects"},
		{Name: "Authentication"},
		{Name: "Welcome Credits"},
		{Name: "Menu"},
		{Name: "YAML Builder"},
		{Name: "AI Marketplace"},
		{Name: "Ops Report"},
		{Name: "Email Suppress"},
		{Name: "Notices"},
	}

	tabs := []SettingsTab{
		// Will be populated as tabs are implemented
	}

	if activeItem < 0 || activeItem >= len(items) {
		activeItem = 0
	}

	return Model{
		menu:     components.SideMenu{Items: items, Active: activeItem},
		settings: s,
		Tabs:     tabs,
	}
}

func (m Model) ExitReason() ExitReason { return m.exitReason }
func (m Model) ActiveTab() int         { return m.menu.Active }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.menu.Prev()
		case "down", "j":
			m.menu.Next()
		case "q", "ctrl+c":
			m.exitReason = ExitQuit
			return m, tea.Quit
		case "enter":
			if m.menu.Active < len(m.Tabs) {
				m.exitReason = ExitEdit
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Settings")

	menuStr := m.menu.Render()

	var content string
	if m.menu.Active < len(m.Tabs) {
		content = m.Tabs[m.menu.Active].View()
	} else {
		content = components.MutedStyle.Render("  (not yet implemented)")
	}

	menuBox := lipgloss.NewStyle().
		Width(18).
		MarginRight(2).
		Render(menuStr)

	contentLines := strings.Split(content, "\n")
	contentStr := strings.Join(contentLines, "\n")
	contentBox := lipgloss.NewStyle().Render(contentStr)

	body := lipgloss.JoinHorizontal(lipgloss.Top, menuBox, contentBox)

	hint := "↑/↓ navigate · enter edit · q quit"
	statusBar := components.MutedStyle.Render(hint)

	return fmt.Sprintf("\n%s\n\n%s\n\n%s\n", header, body, statusBar)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/tui/settings/panel.go internal/tui/settings/tabs/helpers.go
git commit -m "feat: add settings panel TUI shell"
```

---

### Task 4: CLI Command + Template Skip

**Files:**
- Create: `internal/cli/settings.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/template/render.go`

Wire up the `launchpad settings` command and add the template render skip.

- [ ] **Step 1: Create CLI command**

```go
// internal/cli/settings.go
package cli

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
	settingsui "github.com/gradient8/launchpad/internal/tui/settings"
	"github.com/spf13/cobra"
)

func newSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "settings",
		Short: "Edit runtime settings (hot-reloadable)",
		Long:  "Open the interactive settings editor. Changes take effect immediately without restart.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettings(".")
		},
	}
}

func runSettings(dir string) error {
	settingsPath := filepath.Join(dir, "generated", "launchpad", "config", "settings.yml")

	s, err := settings.Load(settingsPath)
	if err != nil {
		return fmt.Errorf("cannot load settings.yml: %w\nRun initial deployment first if the file does not exist.", err)
	}

	activeTab := 0
	for {
		panelModel := settingsui.New(s, activeTab)
		p := tea.NewProgram(panelModel, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return fmt.Errorf("settings panel failed: %w", err)
		}

		m := result.(settingsui.Model)
		activeTab = m.ActiveTab()

		switch m.ExitReason() {
		case settingsui.ExitQuit:
			return nil

		case settingsui.ExitEdit:
			tabIdx := m.ActiveTab()
			if tabIdx >= len(m.Tabs) {
				continue
			}
			tab := m.Tabs[tabIdx]
			form := tab.Form()
			km := huh.NewDefaultKeyMap()
			km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
			form.WithKeyMap(km).WithProgramOptions(tea.WithAltScreen())
			if err := form.Run(); err == nil {
				tab.Apply(s)
				if saveErr := settings.Save(s, settingsPath); saveErr != nil {
					return fmt.Errorf("saving settings: %w", saveErr)
				}
			}
			continue
		}
		break
	}
	return nil
}
```

- [ ] **Step 2: Register in root command**

In `internal/cli/root.go`, add `newSettingsCmd()` to the `cmd.AddCommand(...)` call:

```go
cmd.AddCommand(
    newInstallCmd(),
    newConfigureCmd(),
    newSettingsCmd(),
    newStatusCmd(),
    // ... rest unchanged
)
```

- [ ] **Step 3: Add template render skip**

In `internal/template/render.go`, inside the `RenderAll` function, after `outName` is resolved (after line 61) and before `outPath` is constructed (before line 63), add:

```go
// Skip settings.yml if it already exists (preserve user edits)
if outName == "launchpad/config/settings.yml" {
    outPath := filepath.Join(outDir, outName)
    if _, statErr := os.Stat(outPath); statErr == nil {
        return nil
    }
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/cli/settings.go internal/cli/root.go internal/template/render.go
git commit -m "feat: add launchpad settings command and template skip"
```

---

## Chunk 2: Settings Tabs (Simple Groups)

### Task 5: Registration Tab

**Files:**
- Create: `internal/tui/settings/tabs/registration.go`
- Modify: `internal/tui/settings/panel.go` (add to tabs list)

- [ ] **Step 1: Create registration tab**

```go
// internal/tui/settings/tabs/registration.go
package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type RegistrationTab struct {
	RestrictDomain           bool
	AllowedDomains           string // comma-separated for editing
	RequireEmailVerification bool
	Notice                   string
	BlockedNotice            string
}

func NewRegistrationTab(s *settings.Settings) *RegistrationTab {
	return &RegistrationTab{
		RestrictDomain:           s.Registration.RestrictDomain,
		AllowedDomains:           strings.Join(s.Registration.AllowedDomains, ", "),
		RequireEmailVerification: s.Registration.RequireEmailVerification,
		Notice:                   s.Registration.Notice,
		BlockedNotice:            s.Registration.BlockedNotice,
	}
}

func (t *RegistrationTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Restrict Domain").
				Description("Only allow registration from specific domains").
				Value(&t.RestrictDomain),
			huh.NewInput().
				Title("Allowed Domains").
				Description("Comma-separated list of allowed domains").
				Value(&t.AllowedDomains),
			huh.NewConfirm().
				Title("Require Email Verification").
				Value(&t.RequireEmailVerification),
			huh.NewInput().
				Title("Registration Notice").
				Value(&t.Notice),
			huh.NewInput().
				Title("Blocked Notice").
				Description("Message shown when domain is not allowed").
				Value(&t.BlockedNotice),
		),
	)
}

func (t *RegistrationTab) View() string {
	return fmt.Sprintf(
		"  Restrict Domain: %s\n  Allowed Domains: %s\n  Email Verify:    %s\n  Notice:          %s",
		boolDisplay(t.RestrictDomain),
		displayValue(t.AllowedDomains),
		boolDisplay(t.RequireEmailVerification),
		truncate(displayValue(t.Notice), 40))
}

func (t *RegistrationTab) Apply(s *settings.Settings) {
	s.Registration.RestrictDomain = t.RestrictDomain
	domains := strings.Split(t.AllowedDomains, ",")
	cleaned := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d != "" {
			cleaned = append(cleaned, d)
		}
	}
	s.Registration.AllowedDomains = cleaned
	s.Registration.RequireEmailVerification = t.RequireEmailVerification
	s.Registration.Notice = t.Notice
	s.Registration.BlockedNotice = t.BlockedNotice
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`

- [ ] **Step 3: Commit**

```bash
git add internal/tui/settings/tabs/registration.go
git commit -m "feat: add registration settings tab"
```

**Note:** Do NOT add to panel tabs list yet. All 12 tabs are wired together in Task 11 to avoid broken intermediate states.

---

### Task 6: Plans + Projects + Menu + YAML Builder Tabs

**Files:**
- Create: `internal/tui/settings/tabs/plans.go`
- Create: `internal/tui/settings/tabs/projects.go`
- Create: `internal/tui/settings/tabs/menu.go`
- Create: `internal/tui/settings/tabs/yaml_builder.go`
- Modify: `internal/tui/settings/panel.go` (add to tabs list)

These are all simple tabs with few fields. Grouped together for efficiency.

- [ ] **Step 1: Create plans tab**

```go
// internal/tui/settings/tabs/plans.go
package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type PlansTab struct {
	EnforceFreePlan bool
	Disabled        string // comma-separated plan names
	DisabledNotice  string
}

func NewPlansTab(s *settings.Settings) *PlansTab {
	return &PlansTab{
		EnforceFreePlan: s.Plans.EnforceFreePlan,
		Disabled:        strings.Join(s.Plans.Disabled, ", "),
		DisabledNotice:  s.Plans.DisabledNotice,
	}
}

func (t *PlansTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enforce Free Plan").
				Description("Force all users to free plan").
				Value(&t.EnforceFreePlan),
			huh.NewInput().
				Title("Disabled Plans").
				Description("Comma-separated plan names to disable").
				Value(&t.Disabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when plan upgrades are disabled").
				Value(&t.DisabledNotice),
		),
	)
}

func (t *PlansTab) View() string {
	return fmt.Sprintf(
		"  Enforce Free Plan: %s\n  Disabled Plans:    %s\n  Disabled Notice:   %s",
		boolDisplay(t.EnforceFreePlan),
		displayValue(t.Disabled),
		truncate(displayValue(t.DisabledNotice), 40))
}

func (t *PlansTab) Apply(s *settings.Settings) {
	s.Plans.EnforceFreePlan = t.EnforceFreePlan
	parts := strings.Split(t.Disabled, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	s.Plans.Disabled = cleaned
	s.Plans.DisabledNotice = t.DisabledNotice
}
```

- [ ] **Step 2: Create projects tab**

```go
// internal/tui/settings/tabs/projects.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type ProjectsTab struct {
	CreateDisabled       bool
	CreateDisabledNotice string
}

func NewProjectsTab(s *settings.Settings) *ProjectsTab {
	return &ProjectsTab{
		CreateDisabled:       s.Projects.CreateDisabled,
		CreateDisabledNotice: s.Projects.CreateDisabledNotice,
	}
}

func (t *ProjectsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Disable Project Creation").
				Description("Prevent users from creating new projects").
				Value(&t.CreateDisabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when creation is disabled").
				Value(&t.CreateDisabledNotice),
		),
	)
}

func (t *ProjectsTab) View() string {
	return fmt.Sprintf(
		"  Create Disabled: %s\n  Notice:          %s",
		boolDisplay(t.CreateDisabled),
		truncate(displayValue(t.CreateDisabledNotice), 40))
}

func (t *ProjectsTab) Apply(s *settings.Settings) {
	s.Projects.CreateDisabled = t.CreateDisabled
	s.Projects.CreateDisabledNotice = t.CreateDisabledNotice
}
```

- [ ] **Step 3: Create menu tab**

```go
// internal/tui/settings/tabs/menu.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type MenuTab struct {
	DocsVisible bool
}

func NewMenuTab(s *settings.Settings) *MenuTab {
	return &MenuTab{
		DocsVisible: s.Menu.Docs.Visible,
	}
}

func (t *MenuTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Show Docs Menu").
				Description("Show documentation link in the navigation menu").
				Value(&t.DocsVisible),
		),
	)
}

func (t *MenuTab) View() string {
	return fmt.Sprintf("  Docs Visible: %s", boolDisplay(t.DocsVisible))
}

func (t *MenuTab) Apply(s *settings.Settings) {
	s.Menu.Docs.Visible = t.DocsVisible
}
```

- [ ] **Step 4: Create YAML builder tab**

```go
// internal/tui/settings/tabs/yaml_builder.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type YAMLBuilderTab struct {
	AllowedUsers string
}

func NewYAMLBuilderTab(s *settings.Settings) *YAMLBuilderTab {
	return &YAMLBuilderTab{
		AllowedUsers: s.YAMLBuilder.AllowedUsers,
	}
}

func (t *YAMLBuilderTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Allowed Users").
				Description("'*' for all users, or comma-separated user IDs").
				Value(&t.AllowedUsers),
		),
	)
}

func (t *YAMLBuilderTab) View() string {
	return fmt.Sprintf("  Allowed Users: %s", displayValue(t.AllowedUsers))
}

func (t *YAMLBuilderTab) Apply(s *settings.Settings) {
	s.YAMLBuilder.AllowedUsers = t.AllowedUsers
}
```

- [ ] **Step 5: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/tui/settings/tabs/plans.go internal/tui/settings/tabs/projects.go internal/tui/settings/tabs/menu.go internal/tui/settings/tabs/yaml_builder.go
git commit -m "feat: add plans, projects, menu, yaml-builder settings tabs"
```

---

### Task 7: Credentials Tab

**Files:**
- Create: `internal/tui/settings/tabs/credentials.go`

5 credential toggles with conditional notice fields.

- [ ] **Step 1: Create credentials tab**

```go
// internal/tui/settings/tabs/credentials.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type CredentialsTab struct {
	ClaudeIncludedDisabled bool
	ClaudeIncludedNotice   string
	ZAIIncludedDisabled    bool
	ZAIIncludedNotice      string
	ClaudePaygoDisabled    bool
	ClaudePaygoNotice      string
	ZAIPaygoDisabled       bool
	ZAIPaygoNotice         string
	GeminiPaygoDisabled    bool
	GeminiPaygoNotice      string
}

func NewCredentialsTab(s *settings.Settings) *CredentialsTab {
	return &CredentialsTab{
		ClaudeIncludedDisabled: s.Credentials.ClaudeIncluded.Disabled,
		ClaudeIncludedNotice:   s.Credentials.ClaudeIncluded.Notice,
		ZAIIncludedDisabled:    s.Credentials.ZAIIncluded.Disabled,
		ZAIIncludedNotice:      s.Credentials.ZAIIncluded.Notice,
		ClaudePaygoDisabled:    s.Credentials.ClaudePaygo.Disabled,
		ClaudePaygoNotice:      s.Credentials.ClaudePaygo.Notice,
		ZAIPaygoDisabled:       s.Credentials.ZAIPaygo.Disabled,
		ZAIPaygoNotice:         s.Credentials.ZAIPaygo.Notice,
		GeminiPaygoDisabled:    s.Credentials.GeminiPaygo.Disabled,
		GeminiPaygoNotice:      s.Credentials.GeminiPaygo.Notice,
	}
}

func (t *CredentialsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Claude Included — Disabled").Value(&t.ClaudeIncludedDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("Claude Included — Notice").Value(&t.ClaudeIncludedNotice),
		).WithHideFunc(func() bool { return !t.ClaudeIncludedDisabled }),
		huh.NewGroup(
			huh.NewConfirm().Title("ZAI Included — Disabled").Value(&t.ZAIIncludedDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("ZAI Included — Notice").Value(&t.ZAIIncludedNotice),
		).WithHideFunc(func() bool { return !t.ZAIIncludedDisabled }),
		huh.NewGroup(
			huh.NewConfirm().Title("Claude PayGo — Disabled").Value(&t.ClaudePaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("Claude PayGo — Notice").Value(&t.ClaudePaygoNotice),
		).WithHideFunc(func() bool { return !t.ClaudePaygoDisabled }),
		huh.NewGroup(
			huh.NewConfirm().Title("ZAI PayGo — Disabled").Value(&t.ZAIPaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("ZAI PayGo — Notice").Value(&t.ZAIPaygoNotice),
		).WithHideFunc(func() bool { return !t.ZAIPaygoDisabled }),
		huh.NewGroup(
			huh.NewConfirm().Title("Gemini PayGo — Disabled").Value(&t.GeminiPaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("Gemini PayGo — Notice").Value(&t.GeminiPaygoNotice),
		).WithHideFunc(func() bool { return !t.GeminiPaygoDisabled }),
	)
}

func (t *CredentialsTab) View() string {
	return fmt.Sprintf(
		"  Claude Included: %s\n  ZAI Included:    %s\n  Claude PayGo:    %s\n  ZAI PayGo:       %s\n  Gemini PayGo:    %s",
		credStatus(t.ClaudeIncludedDisabled),
		credStatus(t.ZAIIncludedDisabled),
		credStatus(t.ClaudePaygoDisabled),
		credStatus(t.ZAIPaygoDisabled),
		credStatus(t.GeminiPaygoDisabled))
}

func credStatus(disabled bool) string {
	if disabled {
		return "disabled"
	}
	return "active"
}

func (t *CredentialsTab) Apply(s *settings.Settings) {
	s.Credentials.ClaudeIncluded = settings.CredentialToggle{Disabled: t.ClaudeIncludedDisabled, Notice: t.ClaudeIncludedNotice}
	s.Credentials.ZAIIncluded = settings.CredentialToggle{Disabled: t.ZAIIncludedDisabled, Notice: t.ZAIIncludedNotice}
	s.Credentials.ClaudePaygo = settings.CredentialToggle{Disabled: t.ClaudePaygoDisabled, Notice: t.ClaudePaygoNotice}
	s.Credentials.ZAIPaygo = settings.CredentialToggle{Disabled: t.ZAIPaygoDisabled, Notice: t.ZAIPaygoNotice}
	s.Credentials.GeminiPaygo = settings.CredentialToggle{Disabled: t.GeminiPaygoDisabled, Notice: t.GeminiPaygoNotice}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`

- [ ] **Step 3: Commit**

```bash
git add internal/tui/settings/tabs/credentials.go
git commit -m "feat: add credentials settings tab"
```

---

### Task 8: Authentication Tab

**Files:**
- Create: `internal/tui/settings/tabs/authentication.go`

The most complex tab — nested structs, conditional oauth fields.

- [ ] **Step 1: Create authentication tab**

```go
// internal/tui/settings/tabs/authentication.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type AuthenticationTab struct {
	Mode                  string
	PasswordEnabled       bool
	PasswordEnableSignup  bool
	PasswordDisabledNotice string
	GoogleEnabled         bool
	GoogleAllowSignup     bool
	GitHubEnabled         bool
	GitHubAllowSignup     bool
	MicrosoftEnabled      bool
	MicrosoftAllowSignup  bool
	MicrosoftEnforceOnly  bool
	ProviderSignup        bool
	SSOOnlyNotice         string
}

func NewAuthenticationTab(s *settings.Settings) *AuthenticationTab {
	return &AuthenticationTab{
		Mode:                   s.Authentication.Mode,
		PasswordEnabled:        s.Authentication.PasswordLogin.Enabled,
		PasswordEnableSignup:   s.Authentication.PasswordLogin.EnableSignup,
		PasswordDisabledNotice: s.Authentication.PasswordLogin.DisabledNotice,
		GoogleEnabled:          s.Authentication.OAuth.Google.Enabled,
		GoogleAllowSignup:      s.Authentication.OAuth.Google.AllowSignup,
		GitHubEnabled:          s.Authentication.OAuth.GitHub.Enabled,
		GitHubAllowSignup:      s.Authentication.OAuth.GitHub.AllowSignup,
		MicrosoftEnabled:       s.Authentication.OAuth.Microsoft.Enabled,
		MicrosoftAllowSignup:   s.Authentication.OAuth.Microsoft.AllowSignup,
		MicrosoftEnforceOnly:   s.Authentication.OAuth.Microsoft.EnforceOnly,
		ProviderSignup:         s.Authentication.ProviderSignup,
		SSOOnlyNotice:          s.Authentication.SSOOnlyNotice,
	}
}

func (t *AuthenticationTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Auth Mode").
				Options(
					huh.NewOption("Standard", "standard"),
					huh.NewOption("SSO Only", "sso"),
				).
				Value(&t.Mode),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Password Login Enabled").Value(&t.PasswordEnabled),
			huh.NewConfirm().Title("Password Signup Enabled").Value(&t.PasswordEnableSignup),
			huh.NewInput().Title("Password Disabled Notice").Value(&t.PasswordDisabledNotice),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Google OAuth").Value(&t.GoogleEnabled),
			huh.NewConfirm().Title("Google Allow Signup").Value(&t.GoogleAllowSignup),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("GitHub OAuth").Value(&t.GitHubEnabled),
			huh.NewConfirm().Title("GitHub Allow Signup").Value(&t.GitHubAllowSignup),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Microsoft OAuth").Value(&t.MicrosoftEnabled),
			huh.NewConfirm().Title("Microsoft Allow Signup").Value(&t.MicrosoftAllowSignup),
			huh.NewConfirm().Title("Microsoft Enforce Only").
				Description("When true, only Microsoft login is allowed").
				Value(&t.MicrosoftEnforceOnly),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Provider Signup").
				Description("Allow signup via OAuth providers").
				Value(&t.ProviderSignup),
			huh.NewInput().Title("SSO Only Notice").Value(&t.SSOOnlyNotice),
		),
	)
}

func (t *AuthenticationTab) View() string {
	s := fmt.Sprintf("  Mode: %s\n  Password Login: %s", t.Mode, boolDisplay(t.PasswordEnabled))
	if t.GoogleEnabled {
		s += "\n  Google:    enabled"
	}
	if t.GitHubEnabled {
		s += "\n  GitHub:    enabled"
	}
	if t.MicrosoftEnabled {
		s += "\n  Microsoft: enabled"
		if t.MicrosoftEnforceOnly {
			s += " (enforce only)"
		}
	}
	return s
}

func (t *AuthenticationTab) Apply(s *settings.Settings) {
	s.Authentication.Mode = t.Mode
	s.Authentication.PasswordLogin.Enabled = t.PasswordEnabled
	s.Authentication.PasswordLogin.EnableSignup = t.PasswordEnableSignup
	s.Authentication.PasswordLogin.DisabledNotice = t.PasswordDisabledNotice
	s.Authentication.OAuth.Google.Enabled = t.GoogleEnabled
	s.Authentication.OAuth.Google.AllowSignup = t.GoogleAllowSignup
	s.Authentication.OAuth.GitHub.Enabled = t.GitHubEnabled
	s.Authentication.OAuth.GitHub.AllowSignup = t.GitHubAllowSignup
	s.Authentication.OAuth.Microsoft.Enabled = t.MicrosoftEnabled
	s.Authentication.OAuth.Microsoft.AllowSignup = t.MicrosoftAllowSignup
	s.Authentication.OAuth.Microsoft.EnforceOnly = t.MicrosoftEnforceOnly
	s.Authentication.ProviderSignup = t.ProviderSignup
	s.Authentication.SSOOnlyNotice = t.SSOOnlyNotice
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`

- [ ] **Step 3: Commit**

```bash
git add internal/tui/settings/tabs/authentication.go
git commit -m "feat: add authentication settings tab"
```

---

## Chunk 3: Remaining Tabs + Final Wiring

### Task 9: Welcome Credits + AI Marketplace + Ops Report Tabs

**Files:**
- Create: `internal/tui/settings/tabs/welcome_credits.go`
- Create: `internal/tui/settings/tabs/ai_marketplace.go`
- Create: `internal/tui/settings/tabs/ops_report.go`

- [ ] **Step 1: Create welcome credits tab**

```go
// internal/tui/settings/tabs/welcome_credits.go
package tabs

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type WelcomeCreditsTab struct {
	Enabled       bool
	Amount        string // stored as string for form binding, parsed to int in Apply
	CampaignStart string
	CampaignEnd   string
	Message       string
}

func NewWelcomeCreditsTab(s *settings.Settings) *WelcomeCreditsTab {
	return &WelcomeCreditsTab{
		Enabled:       s.WelcomeCredits.Enabled,
		Amount:        fmt.Sprintf("%d", s.WelcomeCredits.Amount),
		CampaignStart: s.WelcomeCredits.CampaignStart,
		CampaignEnd:   s.WelcomeCredits.CampaignEnd,
		Message:       s.WelcomeCredits.Message,
	}
}

func (t *WelcomeCreditsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Welcome Credits").
				Value(&t.Enabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Amount").
				Description("Number of credits to give new users").
				Value(&t.Amount),
			huh.NewInput().
				Title("Campaign Start").
				Description("ISO date or empty for always (e.g. 2026-01-01)").
				Value(&t.CampaignStart),
			huh.NewInput().
				Title("Campaign End").
				Description("ISO date or empty for no end").
				Value(&t.CampaignEnd),
			huh.NewInput().
				Title("Welcome Message").
				Description("Use {amount} as placeholder").
				Value(&t.Message),
		).WithHideFunc(func() bool { return !t.Enabled }),
	)
}

func (t *WelcomeCreditsTab) View() string {
	s := fmt.Sprintf("  Enabled: %s", boolDisplay(t.Enabled))
	if t.Enabled {
		s += fmt.Sprintf("\n  Amount: %s\n  Message: %s", t.Amount, truncate(displayValue(t.Message), 40))
	}
	return s
}

func (t *WelcomeCreditsTab) Apply(s *settings.Settings) {
	s.WelcomeCredits.Enabled = t.Enabled
	if v, err := strconv.Atoi(t.Amount); err == nil {
		s.WelcomeCredits.Amount = v
	}
	s.WelcomeCredits.CampaignStart = t.CampaignStart
	s.WelcomeCredits.CampaignEnd = t.CampaignEnd
	s.WelcomeCredits.Message = t.Message
}
```

- [ ] **Step 2: Create AI marketplace tab**

```go
// internal/tui/settings/tabs/ai_marketplace.go
package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type AIMarketplaceTab struct {
	Visible          bool
	EnabledProviders string // comma-separated for editing
}

func NewAIMarketplaceTab(s *settings.Settings) *AIMarketplaceTab {
	return &AIMarketplaceTab{
		Visible:          s.AIMarketplace.Visible,
		EnabledProviders: strings.Join(s.AIMarketplace.EnabledProviders, ", "),
	}
}

func (t *AIMarketplaceTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Marketplace Visible").
				Description("Show AI marketplace in the UI").
				Value(&t.Visible),
			huh.NewInput().
				Title("Enabled Providers").
				Description("Comma-separated: claude, zai, gemini").
				Value(&t.EnabledProviders),
		),
	)
}

func (t *AIMarketplaceTab) View() string {
	return fmt.Sprintf(
		"  Visible:   %s\n  Providers: %s",
		boolDisplay(t.Visible),
		displayValue(t.EnabledProviders))
}

func (t *AIMarketplaceTab) Apply(s *settings.Settings) {
	s.AIMarketplace.Visible = t.Visible
	parts := strings.Split(t.EnabledProviders, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	s.AIMarketplace.EnabledProviders = cleaned
}
```

- [ ] **Step 3: Create ops report tab**

```go
// internal/tui/settings/tabs/ops_report.go
package tabs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type OpsReportTab struct {
	Enabled             bool
	AdminEmails         string // comma-separated for editing
	SendTime            string
	BalanceForecastDays string // stored as string for form binding
	TopResourceCount    string // stored as string for form binding
}

func NewOpsReportTab(s *settings.Settings) *OpsReportTab {
	return &OpsReportTab{
		Enabled:             s.OpsReport.Enabled,
		AdminEmails:         strings.Join(s.OpsReport.AdminEmails, ", "),
		SendTime:            s.OpsReport.SendTime,
		BalanceForecastDays: fmt.Sprintf("%d", s.OpsReport.BalanceForecastDays),
		TopResourceCount:    fmt.Sprintf("%d", s.OpsReport.TopResourceCount),
	}
}

func (t *OpsReportTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Enable Operations Report").Value(&t.Enabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Admin Emails").
				Description("Comma-separated list of admin email addresses").
				Value(&t.AdminEmails),
			huh.NewInput().
				Title("Send Time").
				Description("Daily send time in HH:MM format (e.g. 04:00)").
				Value(&t.SendTime),
			huh.NewInput().
				Title("Balance Forecast Days").
				Value(&t.BalanceForecastDays),
			huh.NewInput().
				Title("Top Resource Count").
				Value(&t.TopResourceCount),
		).WithHideFunc(func() bool { return !t.Enabled }),
	)
}

func (t *OpsReportTab) View() string {
	s := fmt.Sprintf("  Enabled: %s", boolDisplay(t.Enabled))
	if t.Enabled {
		s += fmt.Sprintf("\n  Emails:   %s\n  Send At:  %s",
			displayValue(t.AdminEmails), displayValue(t.SendTime))
	}
	return s
}

func (t *OpsReportTab) Apply(s *settings.Settings) {
	s.OpsReport.Enabled = t.Enabled
	parts := strings.Split(t.AdminEmails, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	s.OpsReport.AdminEmails = cleaned
	s.OpsReport.SendTime = t.SendTime
	if v, err := strconv.Atoi(t.BalanceForecastDays); err == nil {
		s.OpsReport.BalanceForecastDays = v
	}
	if v, err := strconv.Atoi(t.TopResourceCount); err == nil {
		s.OpsReport.TopResourceCount = v
	}
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`

- [ ] **Step 5: Commit**

```bash
git add internal/tui/settings/tabs/welcome_credits.go internal/tui/settings/tabs/ai_marketplace.go internal/tui/settings/tabs/ops_report.go
git commit -m "feat: add welcome-credits, ai-marketplace, ops-report settings tabs"
```

---

### Task 10: Email Suppression + Notices Tabs

**Files:**
- Create: `internal/tui/settings/tabs/email_suppression.go`
- Create: `internal/tui/settings/tabs/notices.go`

- [ ] **Step 1: Create email suppression tab**

```go
// internal/tui/settings/tabs/email_suppression.go
package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type EmailSuppressionTab struct {
	SuppressedRecipients string // comma-separated for editing
}

func NewEmailSuppressionTab(s *settings.Settings) *EmailSuppressionTab {
	return &EmailSuppressionTab{
		SuppressedRecipients: strings.Join(s.EmailSuppression.SuppressedRecipients, ", "),
	}
}

func (t *EmailSuppressionTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Suppressed Recipients").
				Description("Comma-separated email addresses to block").
				Value(&t.SuppressedRecipients),
		),
	)
}

func (t *EmailSuppressionTab) View() string {
	if t.SuppressedRecipients == "" {
		return "  Suppressed: (none)"
	}
	return fmt.Sprintf("  Suppressed: %s", truncate(t.SuppressedRecipients, 50))
}

func (t *EmailSuppressionTab) Apply(s *settings.Settings) {
	parts := strings.Split(t.SuppressedRecipients, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	s.EmailSuppression.SuppressedRecipients = cleaned
}
```

- [ ] **Step 2: Create notices tab**

```go
// internal/tui/settings/tabs/notices.go
package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/settings"
)

type NoticesTab struct {
	ProjectListEnabled     bool
	ProjectListSeverity    string
	ProjectListDismissible bool
	ProjectListTitle       string
	ProjectListMessage     string
	ProjectListLinkURL     string
	ProjectListLinkText    string

	CodingAgentEnabled     bool
	CodingAgentSeverity    string
	CodingAgentDismissible bool
	CodingAgentTitle       string
	CodingAgentMessage     string
	CodingAgentLinkURL     string
	CodingAgentLinkText    string

	BillingEnabled     bool
	BillingSeverity    string
	BillingDismissible bool
	BillingTitle       string
	BillingMessage     string
	BillingLinkURL     string
	BillingLinkText    string

	CreditDrawerEnabled     bool
	CreditDrawerSeverity    string
	CreditDrawerDismissible bool
	CreditDrawerTitle       string
	CreditDrawerMessage     string
	CreditDrawerLinkURL     string
	CreditDrawerLinkText    string
}

func NewNoticesTab(s *settings.Settings) *NoticesTab {
	return &NoticesTab{
		ProjectListEnabled: s.Notices.ProjectList.Enabled, ProjectListSeverity: s.Notices.ProjectList.Severity,
		ProjectListDismissible: s.Notices.ProjectList.Dismissible, ProjectListTitle: s.Notices.ProjectList.Title,
		ProjectListMessage: s.Notices.ProjectList.Message, ProjectListLinkURL: s.Notices.ProjectList.LinkURL,
		ProjectListLinkText: s.Notices.ProjectList.LinkText,

		CodingAgentEnabled: s.Notices.CodingAgent.Enabled, CodingAgentSeverity: s.Notices.CodingAgent.Severity,
		CodingAgentDismissible: s.Notices.CodingAgent.Dismissible, CodingAgentTitle: s.Notices.CodingAgent.Title,
		CodingAgentMessage: s.Notices.CodingAgent.Message, CodingAgentLinkURL: s.Notices.CodingAgent.LinkURL,
		CodingAgentLinkText: s.Notices.CodingAgent.LinkText,

		BillingEnabled: s.Notices.Billing.Enabled, BillingSeverity: s.Notices.Billing.Severity,
		BillingDismissible: s.Notices.Billing.Dismissible, BillingTitle: s.Notices.Billing.Title,
		BillingMessage: s.Notices.Billing.Message, BillingLinkURL: s.Notices.Billing.LinkURL,
		BillingLinkText: s.Notices.Billing.LinkText,

		CreditDrawerEnabled: s.Notices.CreditDrawer.Enabled, CreditDrawerSeverity: s.Notices.CreditDrawer.Severity,
		CreditDrawerDismissible: s.Notices.CreditDrawer.Dismissible, CreditDrawerTitle: s.Notices.CreditDrawer.Title,
		CreditDrawerMessage: s.Notices.CreditDrawer.Message, CreditDrawerLinkURL: s.Notices.CreditDrawer.LinkURL,
		CreditDrawerLinkText: s.Notices.CreditDrawer.LinkText,
	}
}

func (t *NoticesTab) Form() *huh.Form {
	severityOptions := []huh.Option[string]{
		huh.NewOption("Warning", "warning"),
		huh.NewOption("Info", "info"),
		huh.NewOption("Error", "error"),
	}
	return huh.NewForm(
		// Project List notice
		huh.NewGroup(
			huh.NewConfirm().Title("Project List Notice — Enabled").Value(&t.ProjectListEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Severity").Options(severityOptions...).Value(&t.ProjectListSeverity),
			huh.NewConfirm().Title("Dismissible").Value(&t.ProjectListDismissible),
			huh.NewInput().Title("Title").Value(&t.ProjectListTitle),
			huh.NewInput().Title("Message").Value(&t.ProjectListMessage),
			huh.NewInput().Title("Link URL").Value(&t.ProjectListLinkURL),
			huh.NewInput().Title("Link Text").Value(&t.ProjectListLinkText),
		).WithHideFunc(func() bool { return !t.ProjectListEnabled }),

		// Coding Agent notice
		huh.NewGroup(
			huh.NewConfirm().Title("Coding Agent Notice — Enabled").Value(&t.CodingAgentEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Severity").Options(severityOptions...).Value(&t.CodingAgentSeverity),
			huh.NewConfirm().Title("Dismissible").Value(&t.CodingAgentDismissible),
			huh.NewInput().Title("Title").Value(&t.CodingAgentTitle),
			huh.NewInput().Title("Message").Value(&t.CodingAgentMessage),
			huh.NewInput().Title("Link URL").Value(&t.CodingAgentLinkURL),
			huh.NewInput().Title("Link Text").Value(&t.CodingAgentLinkText),
		).WithHideFunc(func() bool { return !t.CodingAgentEnabled }),

		// Billing notice
		huh.NewGroup(
			huh.NewConfirm().Title("Billing Notice — Enabled").Value(&t.BillingEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Severity").Options(severityOptions...).Value(&t.BillingSeverity),
			huh.NewConfirm().Title("Dismissible").Value(&t.BillingDismissible),
			huh.NewInput().Title("Title").Value(&t.BillingTitle),
			huh.NewInput().Title("Message").Value(&t.BillingMessage),
			huh.NewInput().Title("Link URL").Value(&t.BillingLinkURL),
			huh.NewInput().Title("Link Text").Value(&t.BillingLinkText),
		).WithHideFunc(func() bool { return !t.BillingEnabled }),

		// Credit Drawer notice
		huh.NewGroup(
			huh.NewConfirm().Title("Credit Drawer Notice — Enabled").Value(&t.CreditDrawerEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Severity").Options(severityOptions...).Value(&t.CreditDrawerSeverity),
			huh.NewConfirm().Title("Dismissible").Value(&t.CreditDrawerDismissible),
			huh.NewInput().Title("Title").Value(&t.CreditDrawerTitle),
			huh.NewInput().Title("Message").Value(&t.CreditDrawerMessage),
			huh.NewInput().Title("Link URL").Value(&t.CreditDrawerLinkURL),
			huh.NewInput().Title("Link Text").Value(&t.CreditDrawerLinkText),
		).WithHideFunc(func() bool { return !t.CreditDrawerEnabled }),
	)
}

func (t *NoticesTab) View() string {
	return fmt.Sprintf(
		"  Project List:   %s\n  Coding Agent:   %s\n  Billing:        %s\n  Credit Drawer:  %s",
		noticeStatus(t.ProjectListEnabled, t.ProjectListTitle),
		noticeStatus(t.CodingAgentEnabled, t.CodingAgentTitle),
		noticeStatus(t.BillingEnabled, t.BillingTitle),
		noticeStatus(t.CreditDrawerEnabled, t.CreditDrawerTitle))
}

func noticeStatus(enabled bool, title string) string {
	if !enabled {
		return "off"
	}
	if title != "" {
		return fmt.Sprintf("on — %s", truncate(title, 30))
	}
	return "on"
}

func (t *NoticesTab) Apply(s *settings.Settings) {
	s.Notices.ProjectList = settings.Notice{
		Enabled: t.ProjectListEnabled, Severity: t.ProjectListSeverity,
		Dismissible: t.ProjectListDismissible, Title: t.ProjectListTitle,
		Message: t.ProjectListMessage, LinkURL: t.ProjectListLinkURL, LinkText: t.ProjectListLinkText,
	}
	s.Notices.CodingAgent = settings.Notice{
		Enabled: t.CodingAgentEnabled, Severity: t.CodingAgentSeverity,
		Dismissible: t.CodingAgentDismissible, Title: t.CodingAgentTitle,
		Message: t.CodingAgentMessage, LinkURL: t.CodingAgentLinkURL, LinkText: t.CodingAgentLinkText,
	}
	s.Notices.Billing = settings.Notice{
		Enabled: t.BillingEnabled, Severity: t.BillingSeverity,
		Dismissible: t.BillingDismissible, Title: t.BillingTitle,
		Message: t.BillingMessage, LinkURL: t.BillingLinkURL, LinkText: t.BillingLinkText,
	}
	s.Notices.CreditDrawer = settings.Notice{
		Enabled: t.CreditDrawerEnabled, Severity: t.CreditDrawerSeverity,
		Dismissible: t.CreditDrawerDismissible, Title: t.CreditDrawerTitle,
		Message: t.CreditDrawerMessage, LinkURL: t.CreditDrawerLinkURL, LinkText: t.CreditDrawerLinkText,
	}
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/settings/...`

- [ ] **Step 4: Commit**

```bash
git add internal/tui/settings/tabs/email_suppression.go internal/tui/settings/tabs/notices.go
git commit -m "feat: add email-suppression and notices settings tabs"
```

---

### Task 11: Wire All Tabs into Panel

**Files:**
- Modify: `internal/tui/settings/panel.go`

Connect all 12 tabs to the panel in the correct order matching the sidebar items.

- [ ] **Step 1: Update panel.go with complete tabs list**

In `internal/tui/settings/panel.go`, update the `New()` function's `tabs` slice:

```go
tabs := []SettingsTab{
    stabs.NewRegistrationTab(s),
    stabs.NewPlansTab(s),
    stabs.NewCredentialsTab(s),
    stabs.NewProjectsTab(s),
    stabs.NewAuthenticationTab(s),
    stabs.NewWelcomeCreditsTab(s),
    stabs.NewMenuTab(s),
    stabs.NewYAMLBuilderTab(s),
    stabs.NewAIMarketplaceTab(s),
    stabs.NewOpsReportTab(s),
    stabs.NewEmailSuppressionTab(s),
    stabs.NewNoticesTab(s),
}
```

Ensure the import has: `stabs "github.com/gradient8/launchpad/internal/tui/settings/tabs"`

- [ ] **Step 2: Build and do a quick smoke test**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./...`
Expected: clean compile, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/settings/panel.go
git commit -m "feat: wire all 12 settings tabs into panel"
```

---

### Task 12: Cross-compile and Deploy to Test Server

**Files:** None (build + deploy only)

- [ ] **Step 1: Cross-compile for Linux**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && GOOS=linux GOARCH=amd64 go build -o launchpad-linux ./cmd/launchpad`
Expected: produces `launchpad-linux` binary.

- [ ] **Step 2: Deploy to test server**

```bash
scp launchpad-linux root@10.233.201.133:/usr/local/bin/launchpad
```

- [ ] **Step 3: Verify on server**

```bash
ssh root@10.233.201.133 "launchpad settings"
```

Expected: Settings TUI opens with sidebar showing all 12 groups, values loaded from `/root/generated/launchpad/config/settings.yml`.
