# TUI Form Redesign: TreeForm + MultiSelect

**Date:** 2026-04-08
**Status:** Draft

## Problem

The current TUI forms have poor UX for three scenarios:

1. **Comma-separated string input** (plans.go, email_suppression.go) — users manually type `"item1,item2"` with no visual feedback or validation
2. **Excessive toggle lists** (credentials.go) — 7 individual Confirm toggles stacked vertically, with conditional notice fields hidden in a separate page
3. **Conditional nested forms** (notices.go) — 4 toggle switches on page 1, then up to 4 additional pages of 6 fields each, with no visual hierarchy showing parent-child relationships

## Solution: Approach B — huh MultiSelect + Custom TreeForm

### Architecture Overview

Two complementary improvements:

1. **TreeForm component** — a new bubbletea Model for editing hierarchical data with inline expand/collapse and field editing
2. **huh MultiSelect adoption** — replace comma-separated string inputs with huh's built-in `NewMultiSelect` component

### TreeForm Component

**Location:** `internal/tui/components/treeform.go` + `treeform_field.go`

#### Data Model

```go
// TreeNode represents a collapsible node with optional child fields
type TreeNode struct {
    Label    string          // Display name, e.g. "Claude Included"
    Expanded bool            // Whether child fields are visible
    Fields   []TreeField     // Child fields (shown when expanded)
    OnToggle func() string   // Space key callback, returns new status text
    Status   func() string   // Returns current status label, e.g. "[disabled]"
}

// TreeField — a single editable sub-field
type TreeField struct {
    Label       string
    Type        FieldType       // FieldText / FieldSelect / FieldToggle
    TextValue   *string         // For FieldText — pointer to string value
    BoolValue   *bool           // For FieldToggle — pointer to bool value
    SelectValue *string         // For FieldSelect — pointer to selected value
    SelectOpts  []string        // For FieldSelect — available options
}

type FieldType int
const (
    FieldText FieldType = iota
    FieldSelect
    FieldToggle
)
```

#### Visual Layout

```
Credentials
─────────────────────────────────────
▼ Claude Included          [disabled]
    Notice: [Service unavailable____]
▶ ZAI Included             [active]
▼ Claude PayGo             [disabled]
    Notice: [Coming soon___________]
▶ ZAI PayGo                [active]
▼ ZAI Ghisha               [disabled]
    Hidden: Yes
    Notice: [Under maintenance_____]
▶ Gemini PayGo             [active]

[↑↓] navigate  [space] toggle  [enter] expand  [esc] done
```

#### Keyboard Bindings

| Key | Action |
|-----|--------|
| `↑/↓` | Move cursor between nodes (skips over child fields of collapsed nodes) |
| `Space` | Toggle current node's status (enabled/disabled) |
| `Enter` | Expand/collapse current node |
| `Tab` / `Shift+Tab` | Move focus between child fields within an expanded node |
| Text keys | Edit when a text field has focus |
| `Esc` | Save and exit the tree form |

#### Rendering Logic

- Each node renders on one line: `▼/▶ {Label}  [{Status}]`
- Expanded nodes show child fields indented below (4 spaces)
- Focused node highlighted with accent color (purple, matching existing theme)
- Focused child field shows input cursor or select indicator
- Status labels use color coding: green for active, yellow for disabled

### Tab Migrations

#### Credentials Tab → TreeForm

**Before:** 7 `huh.NewConfirm()` + 6 `huh.NewInput()` in conditional group
**After:** TreeForm with 6 nodes (one per credential type)

Each node:
- `Space` toggles disabled/active
- Expand reveals Notice text field (and Hidden toggle for ZAI Ghisha)
- Nodes auto-expand when status is "disabled" (to prompt notice entry)

#### Notices Tab → TreeForm

**Before:** 4 `huh.NewConfirm()` on page 1 + up to 4 pages of 6 fields each
**After:** TreeForm with 4 nodes (one per notice location)

Each node:
- `Space` toggles enabled/off
- Expand reveals: Severity (select), Dismissible (toggle), Title, Message, Link URL, Link Text

#### Plans Tab → huh MultiSelect

**Before:** `huh.NewInput()` with comma-separated plan IDs
**After:**

```go
huh.NewMultiSelect[string]().
    Title("Disabled Plans").
    Options(
        huh.NewOption("Free", "free"),
        huh.NewOption("Pro", "pro"),
        huh.NewOption("Enterprise", "enterprise"),
    ).
    Filterable(true).
    Value(&t.Disabled)  // type changes to []string
```

#### Other Tabs with comma-separated inputs

- **email_suppression.go** — keep as Input (email addresses are free-form, not from a fixed list)
- **ai_marketplace.go** — evaluate if providers list is fixed; if yes, convert to MultiSelect
- **registration.go** — domain restrictions are free-form, keep as Input

### Tab Interface Change

Current implicit interface:

```go
Form() *huh.Form
View() string
Apply(s *Settings)
```

New interface:

```go
Edit() error           // Runs huh Form or TreeForm internally
View() string          // Unchanged
Apply(s *Settings)     // Unchanged
```

For huh-based tabs, `Edit()` wraps the existing `form.Run()` call. For TreeForm-based tabs, `Edit()` starts the bubbletea TreeForm program.

### Caller Changes

In `settings/panel.go` and `panel/panel.go`:

```go
// Before
form := tab.Form()
form.WithKeyMap(components.FormKeyMap())
form.WithProgramOptions(tea.WithAltScreen())
form.Run()
tab.Apply(cfg)

// After
tab.Edit()
tab.Apply(cfg)
```

### File Impact

| File | Change |
|------|--------|
| `internal/tui/components/treeform.go` | **New** — TreeForm bubbletea Model (~250-350 lines) |
| `internal/tui/components/treeform_field.go` | **New** — Sub-field rendering and editing logic |
| `internal/tui/settings/tabs/credentials.go` | **Modify** — Form() → Edit() using TreeForm |
| `internal/tui/settings/tabs/notices.go` | **Modify** — Form() → Edit() using TreeForm |
| `internal/tui/settings/tabs/plans.go` | **Modify** — Input → MultiSelect, string → []string |
| `internal/tui/settings/panel.go` | **Modify** — Call tab.Edit() instead of form.Run() |
| `internal/tui/panel/panel.go` | **Modify** — Same caller change |
| All other tab files | **Minor** — Add Edit() method wrapping existing Form().Run() |

### Styling

TreeForm reuses colors and styles from `internal/tui/components/styles.go`:
- Purple accent for focused items
- Green/yellow status badges
- Same border and padding conventions as huh forms

### Testing Strategy

- Manual testing of TreeForm with keyboard navigation
- Verify all tab edits correctly persist to settings.yml
- Test edge cases: empty notices, toggling all credentials, expand/collapse rapid switching

### Non-Goals

- No custom tree components for panel/ tabs (config panel) in this iteration — only settings/ tabs
- No drag-and-drop reordering of nodes
- No search/filter within TreeForm (can add later if needed)
