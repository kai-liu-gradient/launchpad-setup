# Configure Panel Design Spec

## Goal

Replace the placeholder `launchpad configure` panel with a working single-form configuration UI that lets users edit all settings and applies changes (save → re-render → restart services).

## Current State

- `panel.go` has a bubbletea Model with TabBar, `original *config.Config` / `dirty bool` fields, but only renders `[TabName tab content]` placeholder text
- 7 editable tab files exist (`ssl.go`, `database.go`, `smtp.go`, `sso.go`, `storage.go`, `telegram.go`, `ai.go`) each with `Form() *huh.Form` and `Apply(cfg *config.Config)` methods
- `overview.go` is read-only (has `View()` only) — 8 tabs total in the TabBar
- `configure.go` loads config and runs the bubbletea Model but does NOT save config, re-render templates, or restart services after exit. The `configure export` subcommand is unaffected by this work.
- `tab.go` provides TabBar rendering (will be removed)
- `engine/reconfigure.go` exists with `BuildReconfigureSteps` for diff-based targeted restarts, but it is not yet fully wired into the CLI. The configure panel uses a simpler full-restart approach for now.

## Design

### Approach: Single `huh.Form` with section Groups

Replace the bubbletea Model + TabBar with a single `huh.Form` containing Groups (one per configuration section). Each Group has a Title acting as section header. Users fill out sections sequentially; `huh` handles navigation (Tab/Shift+Tab between fields, Enter to confirm and advance), focus, input, and validation using its default keybindings — no custom key handling needed.

### Changes

#### 1. Add `Groups()` method to each tab

Each tab gains a `Groups() []*huh.Group` method that returns its fields wrapped in `huh.Group` instances with a descriptive Title. The existing `Form()` method stays for backward compatibility with standalone commands like `setup-telegram`.

Example for SSLTab:

```go
func (t *SSLTab) Groups() []*huh.Group {
    return []*huh.Group{
        huh.NewGroup(
            huh.NewSelect[string]().Title("SSL Mode").Options(...).Value(&t.Mode),
            huh.NewInput().Title("DNS Provider").Value(&t.DNSProvider),
            huh.NewInput().Title("DNS API Token").EchoMode(huh.EchoModePassword).Value(&t.DNSAPIToken),
            huh.NewInput().Title("Certificate Path").Value(&t.CertPath),
            huh.NewInput().Title("Key Path").Value(&t.KeyPath),
        ).Title("SSL"),
    }
}
```

StorageTab returns 3 groups (mode selection, S3 fields, Azure fields). Note: all 3 groups are always shown regardless of the selected mode — conditional visibility is out of scope for this iteration.

#### 2. Rewrite `panel.go`

Remove bubbletea Model, TabBar, Update(), View(), and the `original`/`dirty` fields (intentionally removed — dirty checking is out of scope). Replace with:

```go
func RunConfigureForm(cfg *config.Config) (applied bool, err error)
```

This function:
1. Creates all 7 tab instances from `cfg`
2. Collects `Groups()` from each tab into a flat `[]*huh.Group` slice
3. Builds one `huh.NewForm(groups...)` and runs it (using default `huh` theme)
4. On success, calls `Apply(cfg)` on each tab
5. Returns `applied=true` so the caller knows to proceed with save/render/restart
6. On cancel (user presses Ctrl+C / Esc), returns `applied=false, err=nil`

Before running the form, prints a one-line config summary (domain, SSL mode, DB mode, storage mode) so the user sees current state.

#### 3. Complete `configure.go` post-apply flow

Follow the same pattern as `setup_telegram.go` (proper error handling, stdout/stderr wiring):

```go
func runConfigure(dir string) error {
    cfgPath := filepath.Join(dir, ".setup.yaml")
    cfg, err := config.Load(cfgPath)
    if err != nil { return fmt.Errorf("loading config: %w", err) }

    secPath := filepath.Join(dir, ".secrets.yaml")
    sec, err := secrets.Load(secPath)
    if err != nil { return fmt.Errorf("loading secrets: %w", err) }

    runtimePath := filepath.Join(dir, "generated", ".runtime.yaml")
    runtime, _ := template.LoadRuntime(runtimePath) // optional, may not exist

    applied, err := panel.RunConfigureForm(cfg)
    if err != nil { return err }
    if !applied { return nil } // user cancelled

    // Save config
    if err := config.Save(cfg, cfgPath); err != nil {
        return fmt.Errorf("saving config: %w", err)
    }
    fmt.Println("✓ Configuration saved.")

    // Re-render templates
    outputDir := filepath.Join(dir, "generated")
    derived := template.ComputeDerived(cfg, sec)
    renderCtx := &template.RenderContext{Config: cfg, Secrets: sec, Derived: derived, Runtime: runtime}
    if err := template.RenderAll(renderCtx, outputDir); err != nil {
        return fmt.Errorf("rendering templates: %w", err)
    }
    fmt.Println("✓ Templates re-rendered.")

    // Restart all services
    composePath := filepath.Join(outputDir, "docker-compose.yml")
    fmt.Println("Restarting services...")
    c := exec.Command("docker", "compose", "-f", composePath, "up", "-d", "--force-recreate")
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    if err := c.Run(); err != nil {
        return fmt.Errorf("restarting services: %w", err)
    }

    fmt.Println("✓ Configure complete.")
    return nil
}
```

The `configure export` subcommand remains unchanged.

#### 4. Delete `tab.go`

The TabBar is no longer used. Remove `internal/tui/panel/tab.go`.

#### 5. Overview — print before form

The OverviewTab's `View()` output is printed to stdout before the form starts, giving users a snapshot of current configuration. OverviewTab does not participate in the form (it has no editable fields).

### Files Changed

| File | Action |
|------|--------|
| `internal/tui/panel/panel.go` | Rewrite — remove bubbletea Model, export `RunConfigureForm()` |
| `internal/tui/panel/tab.go` | Delete |
| `internal/tui/panel/tabs/ssl.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/database.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/smtp.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/sso.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/storage.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/telegram.go` | Add `Groups()` method |
| `internal/tui/panel/tabs/ai.go` | Add `Groups()` method |
| `internal/cli/configure.go` | Complete post-apply flow (save, render, restart) |

### User Experience

```
$ launchpad configure

AniLaunchpad — Current Configuration
Domain: example.com | SSL: selfsigned | DB: builtin | Storage: local

─── SSL ─────────────────────────────
> SSL Mode: [selfsigned ▼]
  DNS Provider: ___
  DNS API Token: ****
  Certificate Path: ___
  Key Path: ___
                        (Tab/Enter → next field, Enter on last → next section)

─── Database ────────────────────────
> Database Mode: [builtin ▼]
  External DB URL: ___

... (SMTP, SSO, Storage, Telegram, AI) ...

✓ Configuration saved.
✓ Templates re-rendered.
Restarting services...
✓ Configure complete.
```

### Out of Scope

- Conditional field visibility (e.g., hiding S3 fields when storage mode is local) — future enhancement
- Dirty checking / partial restart — always full apply + restart all services. The `original`/`dirty` fields in current `panel.go` are intentionally removed.
- Validation beyond what `huh` provides by default
- Custom `huh` theme — uses default theme
- Changes to `engine/reconfigure.go` — not used in this iteration
