# Configure Panel Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the placeholder configure panel with a working single-form UI that edits config, re-renders templates, and restarts services.

**Architecture:** Remove bubbletea Model + TabBar. Each tab exposes `Groups() []*huh.Group`. A new `RunConfigureForm()` function assembles all groups into one `huh.Form`. `configure.go` handles the post-apply flow (save → render → restart).

**Tech Stack:** Go, charmbracelet/huh, charmbracelet/lipgloss

**Spec:** `docs/superpowers/specs/2026-03-20-configure-panel-design.md`

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `internal/tui/panel/tabs/ssl.go` | SSL config fields | Add `Groups()` |
| `internal/tui/panel/tabs/database.go` | Database config fields | Add `Groups()` |
| `internal/tui/panel/tabs/smtp.go` | SMTP config fields | Add `Groups()` |
| `internal/tui/panel/tabs/sso.go` | SSO config fields | Add `Groups()` |
| `internal/tui/panel/tabs/storage.go` | Storage config fields (3 groups) | Add `Groups()` |
| `internal/tui/panel/tabs/telegram.go` | Telegram config fields | Add `Groups()` |
| `internal/tui/panel/tabs/ai.go` | AI config fields | Add `Groups()` |
| `internal/tui/panel/panel.go` | Assemble form + run | Rewrite |
| `internal/tui/panel/tab.go` | TabBar (unused) | Delete |
| `internal/cli/configure.go` | CLI command + post-apply | Modify |

---

## Chunk 1: Tab Groups

### Task 1: Add `Groups()` to SSLTab

**Files:**
- Modify: `internal/tui/panel/tabs/ssl.go`

- [ ] **Step 1: Add Groups() method**

Add after the existing `Form()` method in `ssl.go`:

```go
func (t *SSLTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSL Mode").
				Options(
					huh.NewOption("Self-signed", "selfsigned"),
					huh.NewOption("Let's Encrypt", "letsencrypt"),
					huh.NewOption("Custom Certificate", "custom"),
				).
				Value(&t.Mode),
			huh.NewInput().Title("DNS Provider").Value(&t.DNSProvider),
			huh.NewInput().
				Title("DNS API Token").
				EchoMode(huh.EchoModePassword).
				Value(&t.DNSAPIToken),
			huh.NewInput().Title("Certificate Path").Value(&t.CertPath),
			huh.NewInput().Title("Key Path").Value(&t.KeyPath),
		).Title("SSL"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 2: Add `Groups()` to DatabaseTab

**Files:**
- Modify: `internal/tui/panel/tabs/database.go`

- [ ] **Step 1: Add Groups() method**

Add after the existing `Form()` method:

```go
func (t *DatabaseTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Database Mode").
				Options(
					huh.NewOption("Built-in (managed)", "builtin"),
					huh.NewOption("External", "external"),
				).
				Value(&t.Mode),
			huh.NewInput().
				Title("External DB URL").
				Description("postgres://user:pass@host:5432/db (for external mode)").
				Value(&t.ExternalDB),
		).Title("Database"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 3: Add `Groups()` to SMTPTab

**Files:**
- Modify: `internal/tui/panel/tabs/smtp.go`

- [ ] **Step 1: Add Groups() method**

```go
func (t *SMTPTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().Title("SMTP Host").Value(&t.Host),
			huh.NewInput().Title("SMTP Port").Value(&t.Port),
			huh.NewInput().Title("SMTP User").Value(&t.User),
			huh.NewInput().
				Title("SMTP Password").
				EchoMode(huh.EchoModePassword).
				Value(&t.Password),
			huh.NewInput().Title("From Address").Value(&t.From),
		).Title("SMTP"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 4: Add `Groups()` to SSOTab

**Files:**
- Modify: `internal/tui/panel/tabs/sso.go`

- [ ] **Step 1: Add Groups() method**

```go
func (t *SSOTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().
				Title("Entra Tenant ID").
				Description("Microsoft Entra ID (Azure AD) tenant identifier").
				Value(&t.EntraTenantID),
			huh.NewInput().
				Title("Entra Client ID").
				Description("Application (client) ID registered in Entra").
				Value(&t.EntraClientID),
			huh.NewInput().
				Title("Entra Client Secret").
				EchoMode(huh.EchoModePassword).
				Value(&t.EntraSecret),
		).Title("SSO"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 5: Add `Groups()` to StorageTab

**Files:**
- Modify: `internal/tui/panel/tabs/storage.go`

Note: StorageTab returns 3 groups — one for mode selection, one for S3 fields, one for Azure fields. All groups are always shown (no conditional visibility).

- [ ] **Step 1: Add Groups() method**

```go
func (t *StorageTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Storage Mode").
				Options(
					huh.NewOption("Local", "local"),
					huh.NewOption("S3-compatible", "s3"),
					huh.NewOption("Azure Blob Storage", "azure"),
				).
				Value(&t.Mode),
		).Title("Storage"),
		huh.NewGroup(
			huh.NewInput().Title("S3 Bucket").Value(&t.S3Bucket),
			huh.NewInput().Title("S3 Region").Value(&t.S3Region),
			huh.NewInput().Title("S3 Access Key").Value(&t.S3Key),
			huh.NewInput().
				Title("S3 Secret Key").
				EchoMode(huh.EchoModePassword).
				Value(&t.S3Secret),
		).Title("Storage — S3"),
		huh.NewGroup(
			huh.NewInput().
				Title("Azure Connection String").
				EchoMode(huh.EchoModePassword).
				Value(&t.AzureConn),
			huh.NewInput().Title("Azure Container").Value(&t.AzureContainer),
		).Title("Storage — Azure"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 6: Add `Groups()` to TelegramTab

**Files:**
- Modify: `internal/tui/panel/tabs/telegram.go`

- [ ] **Step 1: Add Groups() method**

```go
func (t *TelegramTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().
				Title("Bot Token").
				Description("Obtain from @BotFather on Telegram").
				EchoMode(huh.EchoModePassword).
				Value(&t.BotToken),
			huh.NewInput().
				Title("Bot Username").
				Description("e.g. MyLaunchpadBot (without @)").
				Value(&t.BotUsername),
		).Title("Telegram"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

---

### Task 7: Add `Groups()` to AITab

**Files:**
- Modify: `internal/tui/panel/tabs/ai.go`

- [ ] **Step 1: Add Groups() method**

```go
func (t *AITab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().
				Title("CRS2 Endpoint").
				Description("AI service endpoint URL").
				Value(&t.CRS2Endpoint),
			huh.NewInput().
				Title("CRS2 Token").
				Description("Authentication token for the AI service").
				EchoMode(huh.EchoModePassword).
				Value(&t.CRS2Token),
		).Title("AI"),
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/panel/tabs/`
Expected: No errors

- [ ] **Step 3: Commit all Groups() additions**

```bash
git add internal/tui/panel/tabs/ssl.go internal/tui/panel/tabs/database.go internal/tui/panel/tabs/smtp.go internal/tui/panel/tabs/sso.go internal/tui/panel/tabs/storage.go internal/tui/panel/tabs/telegram.go internal/tui/panel/tabs/ai.go
git commit -m "feat(configure): add Groups() method to all config tabs"
```

---

## Chunk 2: Panel Rewrite + CLI Integration

### Task 8: Rewrite panel.go — RunConfigureForm

**Files:**
- Rewrite: `internal/tui/panel/panel.go`

- [ ] **Step 1: Replace panel.go with RunConfigureForm**

Replace the entire file with:

```go
package panel

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/panel/tabs"
)

// RunConfigureForm shows an interactive form with all configuration sections.
// Returns applied=true if the user completed the form, false if cancelled.
func RunConfigureForm(cfg *config.Config) (applied bool, err error) {
	// Print current config summary
	overview := tabs.NewOverviewTab(cfg)
	fmt.Println(overview.View())

	// Create tab instances
	sslTab := tabs.NewSSLTab(cfg)
	dbTab := tabs.NewDatabaseTab(cfg)
	smtpTab := tabs.NewSMTPTab(cfg)
	ssoTab := tabs.NewSSOTab(cfg)
	storageTab := tabs.NewStorageTab(cfg)
	telegramTab := tabs.NewTelegramTab(cfg)
	aiTab := tabs.NewAITab(cfg)

	// Collect all groups
	var groups []*huh.Group
	groups = append(groups, sslTab.Groups()...)
	groups = append(groups, dbTab.Groups()...)
	groups = append(groups, smtpTab.Groups()...)
	groups = append(groups, ssoTab.Groups()...)
	groups = append(groups, storageTab.Groups()...)
	groups = append(groups, telegramTab.Groups()...)
	groups = append(groups, aiTab.Groups()...)

	// Run the form
	form := huh.NewForm(groups...)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, fmt.Errorf("running configure form: %w", err)
	}

	// Apply all changes back to config
	sslTab.Apply(cfg)
	dbTab.Apply(cfg)
	smtpTab.Apply(cfg)
	ssoTab.Apply(cfg)
	storageTab.Apply(cfg)
	telegramTab.Apply(cfg)
	aiTab.Apply(cfg)

	return true, nil
}
```

- [ ] **Step 2: Delete tab.go**

```bash
rm internal/tui/panel/tab.go
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/tui/panel/...`
Expected: Build error — `configure.go` still references `panel.New()` and `tea.NewProgram`. That's expected; Task 9 fixes it.

---

### Task 9: Update configure.go — complete post-apply flow

**Files:**
- Modify: `internal/cli/configure.go`

- [ ] **Step 1: Rewrite runConfigure function**

Replace the entire `runConfigure` function and update imports. Keep `newConfigureCmd`, `newConfigureExportCmd`, and `runConfigureExport` unchanged.

New imports:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/gradient8/launchpad/internal/tui/panel"
	"github.com/spf13/cobra"
)
```

New `runConfigure`:

```go
func runConfigure(dir string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	secPath := filepath.Join(dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return fmt.Errorf("loading secrets: %w", err)
	}

	runtimePath := filepath.Join(dir, "generated", ".runtime.yaml")
	runtime, _ := template.LoadRuntime(runtimePath)

	applied, err := panel.RunConfigureForm(cfg)
	if err != nil {
		return err
	}
	if !applied {
		fmt.Println("Configuration cancelled.")
		return nil
	}

	// Save config
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Println("✓ Configuration saved.")

	// Re-render templates
	outputDir := filepath.Join(dir, "generated")
	derived := template.ComputeDerived(cfg, sec)
	renderCtx := &template.RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}
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

Remove the `tea` import (no longer needed). Remove `newConfigureCmd`'s body change — actually `newConfigureCmd` is fine as-is since it just calls `runConfigure(".")`.

- [ ] **Step 2: Verify full project compiles**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && go build ./...`
Expected: No errors

- [ ] **Step 3: Tidy modules**

Run: `go mod tidy`
Expected: No errors. Removes any unused dependencies (e.g., `bubbletea` if no longer imported elsewhere).

- [ ] **Step 4: Run existing tests**

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/tui/panel/panel.go internal/cli/configure.go
git rm internal/tui/panel/tab.go
git commit -m "feat(configure): replace placeholder panel with working huh.Form

Replaces bubbletea Model + TabBar with a single huh.Form containing
all configuration sections. After form completion, saves config,
re-renders templates, and restarts all services."
```

---

## Chunk 3: Build + Manual Test

### Task 10: Build binary and test on server

- [ ] **Step 1: Cross-compile for Linux**

Run: `cd /Users/kola/workspace/gradient8/github/launchpad-setup/new && GOOS=linux GOARCH=amd64 go build -o launchpad ./cmd/launchpad/`
Expected: `launchpad` binary created

- [ ] **Step 2: Upload to test server**

```bash
ssh root@10.233.201.133 "rm -f /usr/local/bin/launchpad"
scp launchpad root@10.233.201.133:/usr/local/bin/launchpad
```

- [ ] **Step 3: Run configure on test server**

```bash
ssh root@10.233.201.133 "cd /opt/launchpad && launchpad configure"
```

Expected: Shows config summary, then sequentially displays SSL → Database → SMTP → SSO → Storage → Telegram → AI forms. After completing all sections, prints save/render/restart success messages.

- [ ] **Step 4: Verify services are healthy after restart**

```bash
ssh root@10.233.201.133 "cd /opt/launchpad/generated && docker compose ps"
```

Expected: All services running and healthy.
