# TUI Form Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ugly multi-toggle and comma-separated forms with a custom TreeForm component for hierarchical editing and huh MultiSelect for list inputs.

**Architecture:** New `TreeForm` bubbletea Model handles expand/collapse tree nodes with inline field editing. The `SettingsTab` interface gains an `Edit() error` method so tabs can run either huh forms or TreeForm internally. Callers (`settings.go`, `configure.go`) switch from `tab.Form().Run()` to `tab.Edit()`.

**Tech Stack:** Go, charmbracelet/bubbletea v1.3.10, charmbracelet/huh v1.0.0, charmbracelet/bubbles v1.0.0, charmbracelet/lipgloss v1.1.0

---

### Task 1: Define the SettingsTab Interface with Edit() Method

**Files:**
- Create: `internal/tui/settings/tabs/tab.go`
- Modify: `internal/tui/settings/panel.go:22-27`

This task introduces the formal `SettingsTab` interface with `Edit()` replacing `Form()` as the primary entry point, while keeping `Form()` available for backward compat during migration.

- [ ] **Step 1: Create the tab interface file**

Create `internal/tui/settings/tabs/tab.go` with a helper function that wraps any existing `Form()` call into `Edit()`:

```go
package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/tui/components"
)

// RunFormAsEdit takes a huh.Form, applies the standard keymap and alt-screen,
// runs it, and returns any error. Tabs that still use huh forms call this
// from their Edit() method.
func RunFormAsEdit(form *huh.Form) error {
	form.WithKeyMap(components.FormKeyMap()).
		WithProgramOptions(tea.WithAltScreen())
	return form.Run()
}
```

- [ ] **Step 2: Update the SettingsTab interface in panel.go**

In `internal/tui/settings/panel.go`, change the interface from `Form() *huh.Form` to `Edit() error`:

Replace lines 22-27:
```go
// SettingsTab is the interface each settings group implements.
type SettingsTab interface {
	View() string
	Form() *huh.Form
	Apply(s *settingsmod.Settings)
}
```

With:
```go
// SettingsTab is the interface each settings group implements.
type SettingsTab interface {
	View() string
	Edit() error
	Apply(s *settingsmod.Settings)
}
```

Also remove the `"github.com/charmbracelet/huh"` import from panel.go since it's no longer referenced there.

- [ ] **Step 3: Update the caller in settings.go**

In `internal/cli/settings.go`, replace lines 54-57:
```go
			tab := m.Tabs[tabIdx]
			form := tab.Form()
			form.WithKeyMap(components.FormKeyMap()).WithProgramOptions(tea.WithAltScreen())
			if err := form.Run(); err == nil {
```

With:
```go
			tab := m.Tabs[tabIdx]
			if err := tab.Edit(); err == nil {
```

This also removes the need for the `components` import in settings.go. Update the import block — remove `"github.com/gradient8/launchpad/internal/tui/components"`.

- [ ] **Step 4: Build to verify compilation**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Compilation errors because tab types don't have `Edit()` yet. This is expected — we'll add them in Task 2.

- [ ] **Step 5: Commit the interface change**

```bash
git add internal/tui/settings/tabs/tab.go internal/tui/settings/panel.go internal/cli/settings.go
git commit -m "refactor: define SettingsTab.Edit() interface, add RunFormAsEdit helper"
```

---

### Task 2: Add Edit() to All Existing Settings Tabs

**Files:**
- Modify: `internal/tui/settings/tabs/menu.go`
- Modify: `internal/tui/settings/tabs/plans.go`
- Modify: `internal/tui/settings/tabs/credentials.go`
- Modify: `internal/tui/settings/tabs/projects.go`
- Modify: `internal/tui/settings/tabs/authentication.go`
- Modify: `internal/tui/settings/tabs/welcome_credits.go`
- Modify: `internal/tui/settings/tabs/yaml_builder.go`
- Modify: `internal/tui/settings/tabs/ai_marketplace.go`
- Modify: `internal/tui/settings/tabs/ops_report.go`
- Modify: `internal/tui/settings/tabs/email_suppression.go`
- Modify: `internal/tui/settings/tabs/registration.go`
- Modify: `internal/tui/settings/tabs/notices.go`

Every tab gets an `Edit() error` method that delegates to `RunFormAsEdit(t.Form())`. The `Form()` method stays — it's just no longer called by the interface. This makes every tab compile against the new interface immediately.

- [ ] **Step 1: Add Edit() to each tab file**

Add this method to every tab struct (each file listed above). Example for `MenuTab` in `menu.go` — add after the `Form()` method:

```go
func (t *MenuTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

The exact same pattern for all 12 tabs:

**plans.go** — add after `Form()`:
```go
func (t *PlansTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**credentials.go** — add after `Form()`:
```go
func (t *CredentialsTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**projects.go** — add after `Form()`:
```go
func (t *ProjectsTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**authentication.go** — add after `Form()`:
```go
func (t *AuthenticationTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**welcome_credits.go** — add after `Form()`:
```go
func (t *WelcomeCreditsTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**yaml_builder.go** — add after `Form()`:
```go
func (t *YAMLBuilderTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**ai_marketplace.go** — add after `Form()`:
```go
func (t *AIMarketplaceTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**ops_report.go** — add after `Form()`:
```go
func (t *OpsReportTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**email_suppression.go** — add after `Form()`:
```go
func (t *EmailSuppressionTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**registration.go** — add after `Form()`:
```go
func (t *RegistrationTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

**notices.go** — add after `Form()`:
```go
func (t *NoticesTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

- [ ] **Step 2: Build to verify everything compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/settings/tabs/
git commit -m "refactor: add Edit() method to all settings tabs"
```

---

### Task 3: Update Config Panel (panel/panel.go) Interface

**Files:**
- Create: `internal/tui/panel/tabs/tab.go`
- Modify: `internal/tui/panel/panel.go:25-29`
- Modify: `internal/cli/configure.go:68-75`
- Modify: all files in `internal/tui/panel/tabs/` (13 tab files)

Same pattern as Tasks 1-2 but for the config panel side (`EditableTab` interface).

- [ ] **Step 1: Create the panel tab helper**

Create `internal/tui/panel/tabs/tab.go`:

```go
package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/tui/components"
)

// RunFormAsEdit takes a huh.Form, applies the standard keymap and alt-screen,
// runs it, and returns any error.
func RunFormAsEdit(form *huh.Form) error {
	form.WithKeyMap(components.FormKeyMap()).
		WithProgramOptions(tea.WithAltScreen())
	return form.Run()
}
```

- [ ] **Step 2: Update EditableTab interface in panel.go**

In `internal/tui/panel/panel.go`, replace lines 25-29:
```go
type EditableTab interface {
	View() string
	Form() *huh.Form
	Apply(cfg *config.Config)
}
```

With:
```go
type EditableTab interface {
	View() string
	Edit() error
	Apply(cfg *config.Config)
}
```

Remove the `"github.com/charmbracelet/huh"` import.

- [ ] **Step 3: Update caller in configure.go**

In `internal/cli/configure.go`, replace lines 70-74:
```go
			tab := m.Tabs[tabIdx]
			form := tab.Form()
			form.WithKeyMap(components.FormKeyMap()).WithProgramOptions(tea.WithAltScreen())
			if err := form.Run(); err == nil {
				tab.Apply(cfg)
```

With:
```go
			tab := m.Tabs[tabIdx]
			if err := tab.Edit(); err == nil {
				tab.Apply(cfg)
```

Remove the `"github.com/gradient8/launchpad/internal/tui/components"` import from configure.go.

- [ ] **Step 4: Add Edit() to all panel tab files**

For each tab file in `internal/tui/panel/tabs/`, add an `Edit()` method. The tabs are: `ssl.go`, `database.go`, `smtp.go`, `sso.go`, `storage.go`, `telegram.go`, `ai.go`, `stripe.go`, `performance.go`, `experimental.go`, `kubernetes.go`, `basic.go`.

**Note:** `overview.go` is read-only (no Form/Apply), and `helpers.go` has no tab struct. Skip these.

For each tab, add after the `Form()` method:

```go
func (t *SSLTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
```

(Replace `SSLTab` with the correct struct name for each file: `DatabaseTab`, `SMTPTab`, `SSOTab`, `StorageTab`, `TelegramTab`, `AITab`, `StripeTab`, `PerformanceTab`, `ExperimentalTab`, `KubernetesTab`, `BasicTab`.)

**Special case — overview.go:** The `OverviewTab` is used for read-only display. If it's in the `Tabs` slice (check panel.go), it needs an Edit() that's a no-op:

```go
func (t *OverviewTab) Edit() error {
	return nil
}
```

Check if OverviewTab is in the `Tabs` slice in `panel.go`. If it's only stored in `m.overview` and not in `m.Tabs`, skip it.

- [ ] **Step 5: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/panel/ internal/cli/configure.go
git commit -m "refactor: add Edit() method to config panel tabs"
```

---

### Task 4: Build the TreeForm Component — Data Model and Init

**Files:**
- Create: `internal/tui/components/treeform.go`

This is the core new component. We'll build it incrementally: data model + Init/constructor first, then Update, then View.

- [ ] **Step 1: Create treeform.go with types and constructor**

Create `internal/tui/components/treeform.go`:

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldType determines how a TreeField renders and edits.
type FieldType int

const (
	FieldText   FieldType = iota // Single-line text input
	FieldSelect                  // Cycle through options with left/right
	FieldToggle                  // Yes/No toggle
)

// TreeField is an editable sub-field displayed under an expanded node.
type TreeField struct {
	Label       string
	Type        FieldType
	TextValue   *string   // FieldText: pointer to the string being edited
	BoolValue   *bool     // FieldToggle: pointer to the bool being edited
	SelectValue *string   // FieldSelect: pointer to the selected value
	SelectOpts  []string  // FieldSelect: available options
}

// TreeNode is a collapsible row in the tree. Space toggles its status,
// Enter expands/collapses it, and expanded nodes show their Fields inline.
type TreeNode struct {
	Label    string         // Display name, e.g. "Claude Included"
	Expanded bool           // Whether child fields are visible
	Fields   []TreeField    // Child fields shown when expanded
	OnToggle func()         // Called when user presses Space on this node
	Status   func() string  // Returns current status text, e.g. "disabled"
}

// TreeForm is a bubbletea Model for editing hierarchical settings.
type TreeForm struct {
	title      string
	nodes      []TreeNode
	cursor     int  // Index into nodes
	fieldFocus int  // -1 = node row focused, 0..N = child field index
	editing    bool // True when a text field has active input
	textInput  textinput.Model
	quitting   bool
}

// NewTreeForm creates a TreeForm with the given title and nodes.
func NewTreeForm(title string, nodes []TreeNode) TreeForm {
	ti := textinput.New()
	ti.CharLimit = 200
	return TreeForm{
		title:      title,
		nodes:      nodes,
		cursor:     0,
		fieldFocus: -1,
		textInput:  ti,
	}
}

func (m TreeForm) Init() tea.Cmd {
	return nil
}
```

- [ ] **Step 2: Build to verify the file compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./internal/tui/components/`

Expected: Clean compilation (unused import warnings are OK at this stage since we use `fmt`, `strings`, `lipgloss` in later steps — add blank identifiers if needed to compile, we'll remove them next task).

If there are unused import errors, temporarily add:

```go
var (
	_ = fmt.Sprintf
	_ = strings.Join
	_ = lipgloss.NewStyle
)
```

We'll remove these in the next task when View() uses them.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/treeform.go
git commit -m "feat: add TreeForm data model and constructor"
```

---

### Task 5: Build TreeForm — Update (Keyboard Handling)

**Files:**
- Modify: `internal/tui/components/treeform.go`

- [ ] **Step 1: Add the Update method**

Add to `internal/tui/components/treeform.go`, after the `Init()` method:

```go
func (m TreeForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If we're editing a text field, handle text input first
		if m.editing {
			switch msg.String() {
			case "esc", "enter":
				// Commit the text value and stop editing
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldText && f.TextValue != nil {
					*f.TextValue = m.textInput.Value()
				}
				m.editing = false
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.fieldFocus > 0 {
				// Move up within child fields
				m.fieldFocus--
			} else if m.fieldFocus == 0 {
				// Move from first child field back to node row
				m.fieldFocus = -1
			} else {
				// Move to previous node
				if m.cursor > 0 {
					m.cursor--
					m.fieldFocus = -1
				}
			}

		case "down", "j":
			node := &m.nodes[m.cursor]
			if m.fieldFocus == -1 && node.Expanded && len(node.Fields) > 0 {
				// Move from node row into first child field
				m.fieldFocus = 0
			} else if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields)-1 {
				// Move down within child fields
				m.fieldFocus++
			} else {
				// Move to next node
				if m.cursor < len(m.nodes)-1 {
					m.cursor++
					m.fieldFocus = -1
				}
			}

		case " ":
			// Toggle node status
			if m.fieldFocus == -1 {
				node := &m.nodes[m.cursor]
				if node.OnToggle != nil {
					node.OnToggle()
				}
			} else {
				// Toggle a FieldToggle child
				node := &m.nodes[m.cursor]
				if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields) {
					f := &node.Fields[m.fieldFocus]
					if f.Type == FieldToggle && f.BoolValue != nil {
						*f.BoolValue = !*f.BoolValue
					}
				}
			}

		case "enter":
			if m.fieldFocus == -1 {
				// Expand/collapse node
				node := &m.nodes[m.cursor]
				node.Expanded = !node.Expanded
				if !node.Expanded {
					m.fieldFocus = -1
				}
			} else {
				// Start editing a text field
				node := &m.nodes[m.cursor]
				if m.fieldFocus >= 0 && m.fieldFocus < len(node.Fields) {
					f := &node.Fields[m.fieldFocus]
					if f.Type == FieldText && f.TextValue != nil {
						m.editing = true
						m.textInput.SetValue(*f.TextValue)
						m.textInput.Focus()
						return m, textinput.Blink
					}
				}
			}

		case "left", "h":
			// Cycle select option backward
			if m.fieldFocus >= 0 {
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldSelect && f.SelectValue != nil && len(f.SelectOpts) > 0 {
					idx := selectIndex(f.SelectOpts, *f.SelectValue)
					if idx > 0 {
						*f.SelectValue = f.SelectOpts[idx-1]
					} else {
						*f.SelectValue = f.SelectOpts[len(f.SelectOpts)-1]
					}
				}
			}

		case "right", "l":
			// Cycle select option forward
			if m.fieldFocus >= 0 {
				node := &m.nodes[m.cursor]
				f := &node.Fields[m.fieldFocus]
				if f.Type == FieldSelect && f.SelectValue != nil && len(f.SelectOpts) > 0 {
					idx := selectIndex(f.SelectOpts, *f.SelectValue)
					if idx < len(f.SelectOpts)-1 {
						*f.SelectValue = f.SelectOpts[idx+1]
					} else {
						*f.SelectValue = f.SelectOpts[0]
					}
				}
			}
		}
	}
	return m, nil
}

// selectIndex finds the index of val in opts, or 0 if not found.
func selectIndex(opts []string, val string) int {
	for i, o := range opts {
		if o == val {
			return i
		}
	}
	return 0
}
```

- [ ] **Step 2: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./internal/tui/components/`

Expected: Clean compilation.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/treeform.go
git commit -m "feat: add TreeForm keyboard handling (Update method)"
```

---

### Task 6: Build TreeForm — View (Rendering)

**Files:**
- Modify: `internal/tui/components/treeform.go`

- [ ] **Step 1: Add the View method and Run helper**

Add to `internal/tui/components/treeform.go`, after the `Update()` method:

```go
func (m TreeForm) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")
	b.WriteString(MutedStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")

	activeStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	statusActive := lipgloss.NewStyle().Foreground(SuccessColor)
	statusDisabled := lipgloss.NewStyle().Foreground(WarningColor)
	fieldLabelStyle := lipgloss.NewStyle().Foreground(SubtitleColor).Width(14).Align(lipgloss.Right)
	fieldFocusStyle := lipgloss.NewStyle().Foreground(PrimaryColor)

	for i, node := range m.nodes {
		isNodeFocused := i == m.cursor && m.fieldFocus == -1

		// Arrow indicator
		arrow := "▶"
		if node.Expanded {
			arrow = "▼"
		}

		// Status badge
		status := ""
		if node.Status != nil {
			s := node.Status()
			if s == "active" || s == "enabled" || s == "on" {
				status = statusActive.Render("[" + s + "]")
			} else {
				status = statusDisabled.Render("[" + s + "]")
			}
		}

		// Node line
		label := fmt.Sprintf("%s %s", arrow, node.Label)
		if isNodeFocused {
			label = activeStyle.Render(label)
		}

		// Pad label to align status badges
		padding := 30 - lipgloss.Width(node.Label)
		if padding < 2 {
			padding = 2
		}
		line := fmt.Sprintf("%s%s%s", label, strings.Repeat(" ", padding), status)
		b.WriteString(line)
		b.WriteString("\n")

		// Child fields (only if expanded)
		if node.Expanded {
			for fi, f := range node.Fields {
				isFieldFocused := i == m.cursor && m.fieldFocus == fi

				flabel := fieldLabelStyle.Render(f.Label + ":")
				var fvalue string

				switch f.Type {
				case FieldText:
					if m.editing && isFieldFocused {
						fvalue = m.textInput.View()
					} else {
						val := ""
						if f.TextValue != nil {
							val = *f.TextValue
						}
						if val == "" {
							val = MutedStyle.Render("(empty)")
						}
						if isFieldFocused {
							fvalue = fieldFocusStyle.Render(val)
						} else {
							fvalue = val
						}
					}

				case FieldSelect:
					val := ""
					if f.SelectValue != nil {
						val = *f.SelectValue
					}
					if isFieldFocused {
						fvalue = fieldFocusStyle.Render("◀ " + val + " ▶")
					} else {
						fvalue = val
					}

				case FieldToggle:
					val := false
					if f.BoolValue != nil {
						val = *f.BoolValue
					}
					display := "No"
					if val {
						display = "Yes"
					}
					if isFieldFocused {
						fvalue = fieldFocusStyle.Render("[" + display + "]")
					} else {
						fvalue = display
					}
				}

				b.WriteString(fmt.Sprintf("    %s %s\n", flabel, fvalue))
			}
		}
	}

	// Help bar
	b.WriteString("\n")
	if m.editing {
		b.WriteString(MutedStyle.Render("type to edit · enter/esc confirm"))
	} else {
		b.WriteString(MutedStyle.Render("↑↓ navigate · space toggle · enter expand · ←→ select · esc done"))
	}

	return b.String()
}

// Run starts the TreeForm as a fullscreen bubbletea program and blocks
// until the user exits. Returns nil on normal exit.
func (m TreeForm) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 2: Remove any blank identifier imports added in Task 4**

If you added `_ = fmt.Sprintf` etc. in Task 4, remove them now — they're no longer needed since `View()` uses all the imports.

- [ ] **Step 3: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./internal/tui/components/`

Expected: Clean compilation.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/treeform.go
git commit -m "feat: add TreeForm View rendering and Run helper"
```

---

### Task 7: Migrate Credentials Tab to TreeForm

**Files:**
- Modify: `internal/tui/settings/tabs/credentials.go`

- [ ] **Step 1: Rewrite the Edit() method to use TreeForm**

Replace the `Edit()` method (added in Task 2) and update imports. The `Form()` method can stay for reference but won't be called anymore.

In `internal/tui/settings/tabs/credentials.go`, replace the `Edit()` method with:

```go
func (t *CredentialsTab) Edit() error {
	nodes := []components.TreeNode{
		{
			Label: "Claude Included",
			OnToggle: func() {
				t.ClaudeIncludedDisabled = !t.ClaudeIncludedDisabled
			},
			Status: func() string {
				return credStatus(t.ClaudeIncludedDisabled)
			},
			Expanded: t.ClaudeIncludedDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ClaudeIncludedNotice},
			},
		},
		{
			Label: "ZAI Included",
			OnToggle: func() {
				t.ZAIIncludedDisabled = !t.ZAIIncludedDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIIncludedDisabled)
			},
			Expanded: t.ZAIIncludedDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIIncludedNotice},
			},
		},
		{
			Label: "Claude PayGo",
			OnToggle: func() {
				t.ClaudePaygoDisabled = !t.ClaudePaygoDisabled
			},
			Status: func() string {
				return credStatus(t.ClaudePaygoDisabled)
			},
			Expanded: t.ClaudePaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ClaudePaygoNotice},
			},
		},
		{
			Label: "ZAI PayGo",
			OnToggle: func() {
				t.ZAIPaygoDisabled = !t.ZAIPaygoDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIPaygoDisabled)
			},
			Expanded: t.ZAIPaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIPaygoNotice},
			},
		},
		{
			Label: "ZAI Ghisha",
			OnToggle: func() {
				t.ZAIGhishaDisabled = !t.ZAIGhishaDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIGhishaDisabled)
			},
			Expanded: t.ZAIGhishaDisabled,
			Fields: []components.TreeField{
				{Label: "Hidden", Type: components.FieldToggle, BoolValue: &t.ZAIGhishaHidden},
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIGhishaNotice},
			},
		},
		{
			Label: "Gemini PayGo",
			OnToggle: func() {
				t.GeminiPaygoDisabled = !t.GeminiPaygoDisabled
			},
			Status: func() string {
				return credStatus(t.GeminiPaygoDisabled)
			},
			Expanded: t.GeminiPaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.GeminiPaygoNotice},
			},
		},
	}

	return components.NewTreeForm("Credentials", nodes).Run()
}
```

Update the imports to add `"github.com/gradient8/launchpad/internal/tui/components"`. The `credStatus` helper function already exists and returns "disabled" or "active".

- [ ] **Step 2: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation.

- [ ] **Step 3: Manual test**

Run the settings command and navigate to Credentials. Verify:
- Nodes display with ▶/▼ arrows and [disabled]/[active] status badges
- Space toggles disabled/active
- Enter expands/collapses nodes
- Disabled nodes auto-expand to show Notice field
- Arrow keys navigate between nodes and fields
- Enter on a text field enables editing, Esc/Enter confirms
- ZAI Ghisha shows both Hidden (toggle) and Notice (text) fields
- Esc on node level exits and returns to panel

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go run . settings`

- [ ] **Step 4: Commit**

```bash
git add internal/tui/settings/tabs/credentials.go
git commit -m "feat: migrate Credentials tab to TreeForm"
```

---

### Task 8: Migrate Notices Tab to TreeForm

**Files:**
- Modify: `internal/tui/settings/tabs/notices.go`

- [ ] **Step 1: Rewrite the Edit() method to use TreeForm**

Replace the `Edit()` method in `notices.go`:

```go
func (t *NoticesTab) Edit() error {
	severityOpts := []string{"warning", "info", "error"}

	nodes := []components.TreeNode{
		{
			Label: "Project List",
			OnToggle: func() {
				t.ProjectListEnabled = !t.ProjectListEnabled
			},
			Status: func() string {
				if t.ProjectListEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.ProjectListEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.ProjectListSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.ProjectListDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.ProjectListTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.ProjectListMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.ProjectListLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.ProjectListLinkText},
			},
		},
		{
			Label: "Coding Agent",
			OnToggle: func() {
				t.CodingAgentEnabled = !t.CodingAgentEnabled
			},
			Status: func() string {
				if t.CodingAgentEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.CodingAgentEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.CodingAgentSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.CodingAgentDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.CodingAgentTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.CodingAgentMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.CodingAgentLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.CodingAgentLinkText},
			},
		},
		{
			Label: "Billing",
			OnToggle: func() {
				t.BillingEnabled = !t.BillingEnabled
			},
			Status: func() string {
				if t.BillingEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.BillingEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.BillingSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.BillingDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.BillingTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.BillingMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.BillingLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.BillingLinkText},
			},
		},
		{
			Label: "Credit Drawer",
			OnToggle: func() {
				t.CreditDrawerEnabled = !t.CreditDrawerEnabled
			},
			Status: func() string {
				if t.CreditDrawerEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.CreditDrawerEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.CreditDrawerSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.CreditDrawerDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.CreditDrawerTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.CreditDrawerMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.CreditDrawerLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.CreditDrawerLinkText},
			},
		},
	}

	return components.NewTreeForm("Notices", nodes).Run()
}
```

Add `"github.com/gradient8/launchpad/internal/tui/components"` to the imports.

- [ ] **Step 2: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation.

- [ ] **Step 3: Manual test**

Run `go run . settings` and navigate to Notices. Verify:
- 4 notice nodes display with [enabled]/[off] status
- Space toggles enabled/off
- Expanding a node shows 6 child fields: Severity (select), Dismissible (toggle), Title, Message, Link URL, Link Text
- Left/right arrows cycle through severity options (warning/info/error)
- Space toggles Dismissible Yes/No
- Enter on text fields enables inline editing
- Esc exits to panel
- Changes persist after Apply (check settings.yml)

- [ ] **Step 4: Commit**

```bash
git add internal/tui/settings/tabs/notices.go
git commit -m "feat: migrate Notices tab to TreeForm"
```

---

### Task 9: Migrate Plans Tab to MultiSelect

**Files:**
- Modify: `internal/tui/settings/tabs/plans.go`

- [ ] **Step 1: Change the Disabled field from string to []string**

In `internal/tui/settings/tabs/plans.go`, update the struct and constructor.

Replace the struct (lines 12-16):
```go
type PlansTab struct {
	EnforceFreePlan bool
	Disabled        string
	DisabledNotice  string
}
```

With:
```go
type PlansTab struct {
	EnforceFreePlan bool
	Disabled        []string
	DisabledNotice  string
}
```

Replace the constructor (lines 18-23):
```go
func NewPlansTab(s *settingsmod.Settings) *PlansTab {
	return &PlansTab{
		EnforceFreePlan: s.Plans.EnforceFreePlan,
		Disabled:        strings.Join(s.Plans.Disabled, ","),
		DisabledNotice:  s.Plans.DisabledNotice,
	}
}
```

With:
```go
func NewPlansTab(s *settingsmod.Settings) *PlansTab {
	// Copy the slice to avoid aliasing
	disabled := make([]string, len(s.Plans.Disabled))
	copy(disabled, s.Plans.Disabled)
	return &PlansTab{
		EnforceFreePlan: s.Plans.EnforceFreePlan,
		Disabled:        disabled,
		DisabledNotice:  s.Plans.DisabledNotice,
	}
}
```

- [ ] **Step 2: Update Form() to use MultiSelect**

Replace the `Form()` method (lines 26-42):
```go
func (t *PlansTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enforce Free Plan").
				Description("Force all users onto the free plan").
				Value(&t.EnforceFreePlan),
			huh.NewInput().
				Title("Disabled Plans").
				Description("Comma-separated list of plan IDs to disable").
				Value(&t.Disabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when a plan is disabled").
				Value(&t.DisabledNotice),
		),
	)
}
```

With:
```go
func (t *PlansTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enforce Free Plan").
				Description("Force all users onto the free plan").
				Value(&t.EnforceFreePlan),
			huh.NewMultiSelect[string]().
				Title("Disabled Plans").
				Description("Select plans to disable").
				Options(
					huh.NewOption("Free", "free"),
					huh.NewOption("Pro", "pro"),
					huh.NewOption("Team", "team"),
					huh.NewOption("Enterprise", "enterprise"),
				).
				Value(&t.Disabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when a plan is disabled").
				Value(&t.DisabledNotice),
		),
	)
}
```

- [ ] **Step 3: Update View() for []string**

Replace the View() method:
```go
func (t *PlansTab) View() string {
	return fmt.Sprintf(
		"  Enforce Free Plan: %s\n  Disabled Plans:    %s\n  Disabled Notice:   %s",
		boolDisplay(t.EnforceFreePlan),
		displayValue(t.Disabled),
		truncate(displayValue(t.DisabledNotice), 40),
	)
}
```

With:
```go
func (t *PlansTab) View() string {
	return fmt.Sprintf(
		"  Enforce Free Plan: %s\n  Disabled Plans:    %s\n  Disabled Notice:   %s",
		boolDisplay(t.EnforceFreePlan),
		listDisplay(t.Disabled),
		truncate(displayValue(t.DisabledNotice), 40),
	)
}
```

- [ ] **Step 4: Simplify Apply()**

Replace the Apply() method (lines 54-71):
```go
func (t *PlansTab) Apply(s *settingsmod.Settings) {
	s.Plans.EnforceFreePlan = t.EnforceFreePlan
	s.Plans.DisabledNotice = t.DisabledNotice

	if t.Disabled == "" {
		s.Plans.Disabled = nil
	} else {
		parts := strings.Split(t.Disabled, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.Plans.Disabled = result
	}
}
```

With:
```go
func (t *PlansTab) Apply(s *settingsmod.Settings) {
	s.Plans.EnforceFreePlan = t.EnforceFreePlan
	s.Plans.DisabledNotice = t.DisabledNotice
	s.Plans.Disabled = t.Disabled
}
```

- [ ] **Step 5: Remove unused strings import**

The `strings` package is no longer needed in plans.go — remove it from the import block.

- [ ] **Step 6: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation.

- [ ] **Step 7: Manual test**

Run `go run . settings`, navigate to Plans. Verify:
- MultiSelect shows plan options with checkboxes
- Space toggles plan selection
- Pre-existing disabled plans show as selected
- Changes persist correctly to settings.yml

- [ ] **Step 8: Commit**

```bash
git add internal/tui/settings/tabs/plans.go
git commit -m "feat: migrate Plans tab from comma input to MultiSelect"
```

---

### Task 10: Clean Up — Remove Dead Form() From Migrated Tabs

**Files:**
- Modify: `internal/tui/settings/tabs/credentials.go`
- Modify: `internal/tui/settings/tabs/notices.go`

Now that Credentials and Notices use TreeForm via `Edit()`, their `Form()` methods are dead code.

- [ ] **Step 1: Remove Form() from credentials.go**

Delete the `Form()` method (the one returning `*huh.Form`) from credentials.go. Also remove unused imports: `"github.com/charmbracelet/huh"` and `"github.com/charmbracelet/lipgloss"` (check if `lipgloss` is still used by other methods — it likely isn't since `credStatus` uses plain strings, but verify).

Keep: `NewCredentialsTab`, `credStatus`, `Edit`, `View`, `Apply`.

- [ ] **Step 2: Remove Form() from notices.go**

Delete the `Form()` method from notices.go. Also remove unused imports: `"github.com/charmbracelet/huh"`, `"github.com/charmbracelet/lipgloss"`. The `severityOptions` variable at package level can also be removed since the severity options are now defined inline in `Edit()`.

Keep: `NewNoticesTab`, `noticeStatus`, `Edit`, `View`, `Apply`.

- [ ] **Step 3: Build to verify**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/settings/tabs/credentials.go internal/tui/settings/tabs/notices.go
git commit -m "refactor: remove dead Form() methods from TreeForm-migrated tabs"
```

---

### Task 11: End-to-End Verification

**Files:** None (testing only)

- [ ] **Step 1: Full build**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup && go build ./...`

Expected: Clean compilation with zero errors.

- [ ] **Step 2: Test all settings tabs**

Run `go run . settings` and click through every tab:
1. **Registration** — still uses huh form, toggles and inputs work
2. **Plans** — MultiSelect for disabled plans, confirm for enforce free plan
3. **Credentials** — TreeForm with expand/collapse, toggle disabled, inline notice editing
4. **Projects** — still uses huh form
5. **Authentication** — still uses huh form with 3 groups
6. **Welcome Credits** — still uses huh form with conditional group
7. **Menu** — still uses huh form
8. **YAML Builder** — still uses huh form
9. **AI Marketplace** — still uses huh form
10. **Ops Report** — still uses huh form with conditional group
11. **Email Suppress** — still uses huh form
12. **Notices** — TreeForm with 4 expandable notice nodes, each with 6 fields

- [ ] **Step 3: Test configure command tabs**

Run `go run . configure` and verify at least one tab opens and edits correctly.

- [ ] **Step 4: Verify settings persistence**

1. Open settings, edit Credentials (disable one, add notice text)
2. Exit and re-open settings — verify the changes persisted
3. Edit Notices (enable one, set severity/title/message)
4. Exit and re-open — verify persistence
5. Edit Plans (select some disabled plans)
6. Exit and re-open — verify persistence

- [ ] **Step 5: Commit if any fixes were needed**

If any fixes were made during testing:
```bash
git add -A
git commit -m "fix: address issues found during end-to-end testing"
```
