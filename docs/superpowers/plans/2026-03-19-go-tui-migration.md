# Go TUI Migration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the shell-based AniLaunchpad setup system to a Go binary using Bubbletea TUI, with embedded templates and single-binary distribution.

**Architecture:** Cobra CLI → Bubbletea TUI (huh forms) → Config/Secrets structs → Go text/template rendering with embed.FS → Deploy engine orchestrating external tools via os/exec. Express install (2 questions) and Custom wizard (6 steps) for installation; Tab-based configure panel for post-install changes.

**Tech Stack:** Go 1.22+, Cobra, Bubbletea, Huh, Lipgloss, Bubbles, go-playground/validator, gopkg.in/yaml.v3

**Spec:** `docs/superpowers/specs/2026-03-19-go-tui-migration-design.md`

---

## File Structure

```
new/
├── cmd/
│   ├── launchpad/main.go                    # CLI entry point
│   └── gen-versions/main.go                 # go:generate tool for versions.conf
├── internal/
│   ├── cli/
│   │   ├── root.go                          # Root cobra command + global flags
│   │   ├── install.go                       # install subcommand
│   │   ├── configure.go                     # configure subcommand
│   │   ├── status.go                        # status subcommand
│   │   ├── upgrade.go                       # upgrade subcommand
│   │   ├── restart.go                       # restart subcommand
│   │   ├── uninstall.go                     # uninstall subcommand
│   │   └── helpers.go                       # setup-certs, setup-db, import-templates
│   ├── config/
│   │   ├── config.go                        # Config struct + DefaultConfig()
│   │   ├── config_test.go                   # Config unit tests
│   │   ├── validate.go                      # Validation rules
│   │   ├── validate_test.go                 # Validation tests
│   │   ├── diff.go                          # Config diff computation
│   │   ├── diff_test.go                     # Diff tests
│   │   ├── io.go                            # Load/Save YAML
│   │   ├── io_test.go                       # I/O tests
│   │   └── versions_gen.go                  # Generated from versions.conf
│   ├── secrets/
│   │   ├── secrets.go                       # Secrets struct + Generate()
│   │   ├── secrets_test.go                  # Secrets tests
│   │   ├── io.go                            # Load/Save with chmod 600
│   │   └── io_test.go                       # I/O tests
│   ├── template/
│   │   ├── embed.go                         # embed.FS declaration
│   │   ├── derived.go                       # DerivedValues + ComputeDerived()
│   │   ├── derived_test.go                  # Derived values tests
│   │   ├── runtime.go                       # RuntimeValues struct + I/O
│   │   ├── render.go                        # RenderAll() + per-template render
│   │   ├── render_test.go                   # Render tests
│   │   └── files/                           # Embedded template files
│   │       ├── env.tmpl
│   │       ├── env.gateway.tmpl
│   │       ├── nginx.conf.tmpl
│   │       ├── settings.yml.tmpl
│   │       ├── docker-compose.yml.tmpl
│   │       ├── coredns-custom.yaml.tmpl
│   │       ├── kyverno-inject-ca.yaml.tmpl
│   │       ├── kyverno-sync-ca.yaml.tmpl
│   │       ├── values-builtin.yml
│   │       └── crontab
│   ├── engine/
│   │   ├── engine.go                        # Engine struct + Deploy()
│   │   ├── engine_test.go                   # Engine tests (mocked executor)
│   │   ├── exec.go                          # RunWithTimeout, DockerExec, DockerRun
│   │   ├── exec_test.go                     # Exec tests
│   │   ├── steps.go                         # Step definitions + buildStepList()
│   │   ├── preflight.go                     # Pre-flight checks
│   │   ├── preflight_test.go                # Pre-flight tests
│   │   ├── health.go                        # Health check utilities
│   │   ├── reconfigure.go                   # ApplyChanges() logic
│   │   └── reconfigure_test.go              # Reconfigure tests
│   └── tui/
│       ├── wizard/
│       │   ├── wizard.go                    # Main wizard model (express/custom switch)
│       │   ├── express.go                   # Express form (domain + email)
│       │   └── custom.go                    # Custom 6-step form
│       ├── progress/
│       │   ├── progress.go                  # Deploy progress model
│       │   └── step.go                      # Step rendering
│       ├── panel/
│       │   ├── panel.go                     # Configure panel model (tab management)
│       │   ├── tab.go                       # Tab component
│       │   └── tabs/                        # Per-tab forms
│       │       ├── overview.go
│       │       ├── ssl.go
│       │       ├── database.go
│       │       ├── smtp.go
│       │       ├── sso.go
│       │       ├── storage.go
│       │       ├── telegram.go
│       │       └── ai.go
│       └── components/
│           └── styles.go                    # Shared lipgloss styles
├── go.mod
├── go.sum
└── Makefile
```

---

## Chunk 1: Foundation — Project Scaffolding + Config + Secrets

### Task 1: Initialize Go module and dependencies

**Files:**
- Create: `new/go.mod`
- Create: `new/cmd/launchpad/main.go`
- Create: `new/Makefile`

- [ ] **Step 1: Create Go module**

```bash
mkdir -p new/cmd/launchpad
cd new && go mod init github.com/gradient8/launchpad
```

- [ ] **Step 2: Add dependencies**

```bash
cd new
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/huh@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get gopkg.in/yaml.v3@latest
go get github.com/go-playground/validator/v10@latest
```

- [ ] **Step 3: Write minimal main.go**

```go
// new/cmd/launchpad/main.go
package main

import (
	"fmt"
	"os"

	"github.com/gradient8/launchpad/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Write root CLI command stub**

```go
// new/internal/cli/root.go
package cli

import "github.com/spf13/cobra"

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "launchpad",
		Short:   "AniLaunchpad deployment tool",
		Version: version,
	}
	return cmd
}

func Execute(version string) error {
	return newRootCmd(version).Execute()
}
```

- [ ] **Step 5: Write Makefile**

```makefile
# new/Makefile
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")

.PHONY: build test lint

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/launchpad ./cmd/launchpad

test:
	go test ./... -v

lint:
	go vet ./...

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/launchpad-linux-amd64 ./cmd/launchpad
```

- [ ] **Step 6: Verify build**

Run: `cd new && make build`
Expected: Binary at `new/bin/launchpad`

Run: `./bin/launchpad --version`
Expected: Prints version string

- [ ] **Step 7: Commit**

```bash
git add new/
git commit -m "feat(new): initialize Go module with Cobra CLI skeleton"
```

---

### Task 2: Config struct + defaults + validation

**Files:**
- Create: `new/internal/config/config.go`
- Create: `new/internal/config/config_test.go`
- Create: `new/internal/config/validate.go`
- Create: `new/internal/config/validate_test.go`

- [ ] **Step 1: Write config struct tests**

```go
// new/internal/config/config_test.go
package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")

	if cfg.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "example.com")
	}
	if cfg.Subdomain != "launchpad" {
		t.Errorf("Subdomain = %q, want %q", cfg.Subdomain, "launchpad")
	}
	if cfg.AdminEmail != "admin@example.com" {
		t.Errorf("AdminEmail = %q, want %q", cfg.AdminEmail, "admin@example.com")
	}
	if cfg.SSL.Mode != "selfsigned" {
		t.Errorf("SSL.Mode = %q, want %q", cfg.SSL.Mode, "selfsigned")
	}
	if cfg.Database.Mode != "builtin" {
		t.Errorf("Database.Mode = %q, want %q", cfg.Database.Mode, "builtin")
	}
	if cfg.Kubernetes.Mode != "builtin" {
		t.Errorf("Kubernetes.Mode = %q, want %q", cfg.Kubernetes.Mode, "builtin")
	}
	if cfg.Storage.Mode != "local" {
		t.Errorf("Storage.Mode = %q, want %q", cfg.Storage.Mode, "local")
	}
	if cfg.Performance.APIReplicas != 1 {
		t.Errorf("Performance.APIReplicas = %d, want 1", cfg.Performance.APIReplicas)
	}
	if cfg.Performance.DBConnLimit != 100 {
		t.Errorf("Performance.DBConnLimit = %d, want 100", cfg.Performance.DBConnLimit)
	}
	if cfg.Images.Registry == "" {
		t.Error("Images.Registry should not be empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd new && go test ./internal/config/ -v -run TestDefaultConfig`
Expected: FAIL — package/types not defined

- [ ] **Step 3: Write config.go with all structs and DefaultConfig**

Write `new/internal/config/config.go` containing all structs from the spec: `Config`, `ImageConfig`, `SSLConfig`, `DBConfig`, `K8sConfig`, `SMTPConfig`, `SSOConfig`, `StorageConfig`, `TelegramConfig`, `AIConfig`, `StripeConfig`, `PerfConfig`. Include `DefaultConfig(domain, email string) *Config` returning all defaults (selfsigned, builtin, local, etc.). Use exact struct definitions from spec lines 103-198.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd new && go test ./internal/config/ -v -run TestDefaultConfig`
Expected: PASS

- [ ] **Step 5: Write validation tests**

```go
// new/internal/config/validate_test.go
package config

import "testing"

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	if err := Validate(cfg); err != nil {
		t.Errorf("valid config should not error: %v", err)
	}
}

func TestValidate_MissingDomain(t *testing.T) {
	cfg := DefaultConfig("", "admin@example.com")
	if err := Validate(cfg); err == nil {
		t.Error("empty domain should fail validation")
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	cfg := DefaultConfig("example.com", "not-an-email")
	if err := Validate(cfg); err == nil {
		t.Error("invalid email should fail validation")
	}
}

func TestValidate_InvalidSSLMode(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.SSL.Mode = "invalid"
	if err := Validate(cfg); err == nil {
		t.Error("invalid SSL mode should fail validation")
	}
}

func TestValidate_ExternalDBRequiresURLs(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.Database.Mode = "external"
	cfg.Database.URLs = nil
	if err := Validate(cfg); err == nil {
		t.Error("external DB without URLs should fail validation")
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `cd new && go test ./internal/config/ -v -run TestValidate`
Expected: FAIL — `Validate` not defined

- [ ] **Step 7: Write validate.go**

Write `new/internal/config/validate.go` with `func Validate(cfg *Config) error` using `go-playground/validator` for struct tag validation plus custom rules: SSL mode must be one of `selfsigned|letsencrypt|custom`; DB/K8s mode must be `builtin|external`; external DB requires non-empty URLs map; external K8s requires kubeconfig path; letsencrypt requires dns_provider.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd new && go test ./internal/config/ -v`
Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add new/internal/config/
git commit -m "feat(new): add Config struct with defaults and validation"
```

---

### Task 3: Config I/O (YAML load/save)

**Files:**
- Create: `new/internal/config/io.go`
- Create: `new/internal/config/io_test.go`

- [ ] **Step 1: Write I/O tests**

```go
// new/internal/config/io_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".setup.yaml")

	original := DefaultConfig("example.com", "admin@example.com")
	original.SSL.Mode = "letsencrypt"
	original.SSL.DNSProvider = "cloudflare"

	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Domain != original.Domain {
		t.Errorf("Domain = %q, want %q", loaded.Domain, original.Domain)
	}
	if loaded.SSL.Mode != "letsencrypt" {
		t.Errorf("SSL.Mode = %q, want %q", loaded.SSL.Mode, "letsencrypt")
	}
	if loaded.SSL.DNSProvider != "cloudflare" {
		t.Errorf("SSL.DNSProvider = %q, want %q", loaded.SSL.DNSProvider, "cloudflare")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/config/ -v -run TestSave`
Expected: FAIL

- [ ] **Step 3: Write io.go**

Write `new/internal/config/io.go` with `func Save(cfg *Config, path string) error` (marshal to YAML, write with 0644 perms) and `func Load(path string) (*Config, error)` (read file, unmarshal YAML).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/config/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add new/internal/config/io.go new/internal/config/io_test.go
git commit -m "feat(new): add Config YAML save/load"
```

---

### Task 4: Config diff computation

**Files:**
- Create: `new/internal/config/diff.go`
- Create: `new/internal/config/diff_test.go`

- [ ] **Step 1: Write diff tests**

```go
// new/internal/config/diff_test.go
package config

import "testing"

func TestDiff_NoChanges(t *testing.T) {
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
	d := Diff(a, b)
	if len(d.Changes) != 0 {
		t.Errorf("expected no changes, got %d", len(d.Changes))
	}
}

func TestDiff_SSLModeChanged(t *testing.T) {
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
	b.SSL.Mode = "letsencrypt"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Error("expected changes for SSL mode")
	}
	services := d.AffectedServices()
	// SSL change affects nginx and possibly all services
	found := false
	for _, s := range services {
		if s == "nginx" {
			found = true
		}
	}
	if !found {
		t.Error("SSL change should affect nginx")
	}
}

func TestDiff_SMTPAdded(t *testing.T) {
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
	b.SMTP.Host = "smtp.example.com"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Error("expected changes for SMTP")
	}
	services := d.AffectedServices()
	found := false
	for _, s := range services {
		if s == "api" {
			found = true
		}
	}
	if !found {
		t.Error("SMTP change should affect api")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/config/ -v -run TestDiff`
Expected: FAIL

- [ ] **Step 3: Write diff.go**

Write `new/internal/config/diff.go` with:
- `type ConfigDiff struct { Changes []Change }` where `Change` has `Field string` and `Old, New interface{}`
- `func Diff(old, new *Config) *ConfigDiff` — compares fields using reflect or manual comparison
- `func (d *ConfigDiff) AffectedServices() []string` — maps changed fields to service names:
  - SSL changes → nginx, api, ui, router, gateway
  - SMTP changes → api
  - SSO changes → api
  - Storage changes → api
  - Telegram changes → api
  - AI changes → api
  - Stripe changes → api
  - Performance changes → api
  - Database changes → all services
  - Images changes → affected service by name

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/config/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add new/internal/config/diff.go new/internal/config/diff_test.go
git commit -m "feat(new): add Config diff computation with affected services"
```

---

### Task 5: Secrets generation + I/O

**Files:**
- Create: `new/internal/secrets/secrets.go`
- Create: `new/internal/secrets/secrets_test.go`
- Create: `new/internal/secrets/io.go`
- Create: `new/internal/secrets/io_test.go`

- [ ] **Step 1: Write secrets generation tests**

```go
// new/internal/secrets/secrets_test.go
package secrets

import "testing"

func TestGenerate(t *testing.T) {
	s, err := Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Check hex secrets are correct length (32 bytes = 64 hex chars)
	if len(s.JWTSecret) != 64 {
		t.Errorf("JWTSecret length = %d, want 64", len(s.JWTSecret))
	}
	if len(s.JWTRefreshSecret) != 64 {
		t.Errorf("JWTRefreshSecret length = %d, want 64", len(s.JWTRefreshSecret))
	}
	if len(s.SessionSecret) != 64 {
		t.Errorf("SessionSecret length = %d, want 64", len(s.SessionSecret))
	}
	// Check encryption keys (16 bytes = 32 hex chars)
	if len(s.EncryptionKey) != 32 {
		t.Errorf("EncryptionKey length = %d, want 32", len(s.EncryptionKey))
	}
	// Check RSA keys are populated
	if s.RSAPublicKey == "" {
		t.Error("RSAPublicKey should not be empty")
	}
	if s.RSAPrivateKey == "" {
		t.Error("RSAPrivateKey should not be empty")
	}
	// Check DB passwords (24 chars base64-safe)
	if len(s.DBPasswordMain) != 24 {
		t.Errorf("DBPasswordMain length = %d, want 24", len(s.DBPasswordMain))
	}
	if s.DBPasswordMain == s.DBPasswordMonitoring {
		t.Error("DB passwords should be unique")
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	s1, _ := Generate()
	s2, _ := Generate()
	if s1.JWTSecret == s2.JWTSecret {
		t.Error("two Generate calls should produce different JWTSecret")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/secrets/ -v -run TestGenerate`
Expected: FAIL

- [ ] **Step 3: Write secrets.go**

Write `new/internal/secrets/secrets.go` with:
- `Secrets` struct matching spec (all 21 named fields)
- `func Generate() (*Secrets, error)` using `crypto/rand` for hex/base64 secrets, `crypto/rsa` for RSA keypair (2048-bit, base64-encoded)
- Helper: `randHex(n int) string` — n random bytes as hex
- Helper: `randBase64Safe(n int) string` — n random bytes as URL-safe base64 (trimmed)
- Helper: `generateRSAKeyPair() (pub, priv string, err error)` — PEM → base64

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/secrets/ -v`
Expected: All PASS

- [ ] **Step 5: Write I/O tests**

```go
// new/internal/secrets/io_test.go
package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".secrets")

	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file permissions
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file perms = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.JWTSecret != original.JWTSecret {
		t.Error("JWTSecret mismatch after save/load")
	}
	if loaded.RSAPublicKey != original.RSAPublicKey {
		t.Error("RSAPublicKey mismatch after save/load")
	}
}
```

- [ ] **Step 6: Write io.go**

Write `new/internal/secrets/io.go` with `func Save(s *Secrets, path string) error` (YAML marshal, write with 0600 perms) and `func Load(path string) (*Secrets, error)`.

- [ ] **Step 7: Run all tests**

Run: `cd new && go test ./internal/secrets/ -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add new/internal/secrets/
git commit -m "feat(new): add Secrets generation and I/O with crypto/rand"
```

---

## Chunk 2: Template Engine

### Task 6: DerivedValues computation

**Files:**
- Create: `new/internal/template/derived.go`
- Create: `new/internal/template/derived_test.go`

- [ ] **Step 1: Write derived values tests**

```go
// new/internal/template/derived_test.go
package template

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func testConfig() *config.Config {
	return config.DefaultConfig("example.com", "admin@example.com")
}

func testSecrets() *secrets.Secrets {
	s, _ := secrets.Generate()
	return s
}

func TestComputeDerived_URLs(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	d := ComputeDerived(cfg, sec)

	if d.DashboardURL != "https://launchpad.example.com" {
		t.Errorf("DashboardURL = %q, want %q", d.DashboardURL, "https://launchpad.example.com")
	}
	if d.GiteaURL != "https://launchpad-gitea.example.com" {
		t.Errorf("GiteaURL = %q, want %q", d.GiteaURL, "https://launchpad-gitea.example.com")
	}
	if d.APIURL != "https://launchpad.example.com:6804" {
		t.Errorf("APIURL = %q, want %q", d.APIURL, "https://launchpad.example.com:6804")
	}
	if d.ExternalDomain != "example.com" {
		t.Errorf("ExternalDomain = %q, want %q", d.ExternalDomain, "example.com")
	}
}

func TestComputeDerived_SelfSignedTLS(t *testing.T) {
	cfg := testConfig()
	cfg.SSL.Mode = "selfsigned"
	d := ComputeDerived(cfg, testSecrets())
	if d.NodeTLSReject != "0" {
		t.Errorf("NodeTLSReject = %q, want %q for selfsigned", d.NodeTLSReject, "0")
	}
}

func TestComputeDerived_LetsEncryptTLS(t *testing.T) {
	cfg := testConfig()
	cfg.SSL.Mode = "letsencrypt"
	d := ComputeDerived(cfg, testSecrets())
	if d.NodeTLSReject != "" {
		t.Errorf("NodeTLSReject = %q, want empty for letsencrypt", d.NodeTLSReject)
	}
}

func TestComputeDerived_BuiltinDBConnStrings(t *testing.T) {
	cfg := testConfig()
	cfg.Database.Mode = "builtin"
	sec := testSecrets()
	d := ComputeDerived(cfg, sec)

	schemas := []string{"main", "monitoring", "events", "billing", "stats", "gateway"}
	for _, schema := range schemas {
		if _, ok := d.DBConnStrings[schema]; !ok {
			t.Errorf("missing DBConnString for schema %q", schema)
		}
	}
}

func TestComputeDerived_ImageRefs(t *testing.T) {
	cfg := testConfig()
	d := ComputeDerived(cfg, testSecrets())

	services := []string{"api", "ui", "router", "gateway", "gitea"}
	for _, svc := range services {
		if _, ok := d.ImageRefs[svc]; !ok {
			t.Errorf("missing ImageRef for %q", svc)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/template/ -v -run TestComputeDerived`
Expected: FAIL

- [ ] **Step 3: Write derived.go**

Write `new/internal/template/derived.go` with `DerivedValues` struct (all fields from spec lines 248-280) and `func ComputeDerived(cfg *config.Config, sec *secrets.Secrets) *DerivedValues`. Logic:
- URLs: concatenate `https://{subdomain}.{domain}`, `https://{subdomain}-gitea.{domain}`, etc. Note: if subdomain is "launchpad", it produces `launchpad.example.com` (no double prefix)
- DB connection strings: for builtin, construct `postgresql://{user}:{pass}@localhost:5432/{db}`; for external, use URLs from config
- Image refs: `{registry}/{image}:{version}`, with per-service overrides
- NodeTLSReject: "0" for selfsigned, "" otherwise
- HostIP: placeholder "127.0.0.1" for now (dynamically detected at deploy time)

Reference `deploy/scripts/lib/render.sh` for the exact derivation logic.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/template/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add new/internal/template/derived.go new/internal/template/derived_test.go
git commit -m "feat(new): add DerivedValues computation for template rendering"
```

---

### Task 7: RuntimeValues + embed.FS setup

**Files:**
- Create: `new/internal/template/runtime.go`
- Create: `new/internal/template/embed.go`
- Create: `new/internal/template/files/` (placeholder template files)

- [ ] **Step 1: Write RuntimeValues**

```go
// new/internal/template/runtime.go
package template

import (
	"os"

	"gopkg.in/yaml.v3"
)

type RuntimeValues struct {
	GiteaAccessToken string `yaml:"gitea_access_token,omitempty"`
	GiteaUser        string `yaml:"gitea_user,omitempty"`
}

func LoadRuntime(path string) (*RuntimeValues, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RuntimeValues{}, nil
		}
		return nil, err
	}
	var rv RuntimeValues
	if err := yaml.Unmarshal(data, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (rv *RuntimeValues) Save(path string) error {
	data, err := yaml.Marshal(rv)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
```

- [ ] **Step 2: Write embed.go**

```go
// new/internal/template/embed.go
package template

import "embed"

//go:embed all:files
var Templates embed.FS
```

- [ ] **Step 3: Create placeholder template files**

Create `new/internal/template/files/` directory with a minimal `env.tmpl`:

```
# AniLaunchpad Environment Configuration
# Generated by launchpad — do not edit manually

DOMAIN={{ .Config.Domain }}
SUBDOMAIN={{ .Config.Subdomain }}
ADMIN_EMAIL={{ .Config.AdminEmail }}
PUBLIC_URL={{ .Derived.PublicURL }}

# SSL
SSL_MODE={{ .Config.SSL.Mode }}
NODE_TLS_REJECT_UNAUTHORIZED={{ .Derived.NodeTLSReject }}
```

Other template files will be migrated from shell envsubst to Go template syntax in a later task. For now, create stubs with comments so embedding works.

- [ ] **Step 4: Write RuntimeValues round-trip test**

```go
// add to new/internal/template/derived_test.go or a new runtime_test.go
func TestRuntimeValues_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".runtime.yaml")

	rv := &RuntimeValues{GiteaAccessToken: "token-123", GiteaUser: "admin"}
	if err := rv.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadRuntime(path)
	if err != nil {
		t.Fatalf("LoadRuntime failed: %v", err)
	}
	if loaded.GiteaAccessToken != "token-123" {
		t.Error("GiteaAccessToken mismatch")
	}
}

func TestRuntimeValues_LoadMissing(t *testing.T) {
	rv, err := LoadRuntime("/nonexistent")
	if err != nil {
		t.Error("missing file should return empty RuntimeValues, not error")
	}
	if rv.GiteaAccessToken != "" {
		t.Error("missing file should return empty token")
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd new && go test ./internal/template/ -v -run TestRuntime`
Expected: All PASS

- [ ] **Step 6: Verify build compiles with embed**

Run: `cd new && go build ./internal/template/`
Expected: Compiles without error

- [ ] **Step 5: Commit**

```bash
git add new/internal/template/runtime.go new/internal/template/embed.go new/internal/template/files/
git commit -m "feat(new): add RuntimeValues, embed.FS setup, and template stubs"
```

---

### Task 8: Template renderer

**Files:**
- Create: `new/internal/template/render.go`
- Create: `new/internal/template/render_test.go`

- [ ] **Step 1: Write render tests**

```go
// new/internal/template/render_test.go
package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAll(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	derived := ComputeDerived(cfg, sec)
	runtime := &RuntimeValues{}

	ctx := &RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Check .env was created
	envPath := filepath.Join(outDir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DOMAIN=example.com") {
		t.Error(".env should contain DOMAIN=example.com")
	}
	if !strings.Contains(content, "SSL_MODE=selfsigned") {
		t.Error(".env should contain SSL_MODE=selfsigned")
	}
}

func TestRenderAll_PreservesRuntime(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	derived := ComputeDerived(cfg, sec)
	runtime := &RuntimeValues{
		GiteaAccessToken: "test-token-123",
		GiteaUser:        "admin",
	}

	ctx := &RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	envPath := filepath.Join(outDir, ".env")
	data, _ := os.ReadFile(envPath)
	content := string(data)
	if !strings.Contains(content, "test-token-123") {
		t.Error(".env should contain the runtime Gitea token")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/template/ -v -run TestRender`
Expected: FAIL

- [ ] **Step 3: Write render.go**

Write `new/internal/template/render.go` with:
- `type RenderContext struct { Config, Secrets, Derived, Runtime }`
- `func RenderAll(ctx *RenderContext, outDir string) error` — walks `Templates` embed.FS, for each `.tmpl` file: parse as `text/template`, execute with ctx, write to `outDir` stripping `.tmpl` suffix. Map template output names: `env.tmpl` → `.env`, `env.gateway.tmpl` → `.env.gateway`, `docker-compose.yml.tmpl` → `docker-compose.yml`, etc.
- Non-template files (like `values-builtin.yml`, `crontab`) are copied as-is.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/template/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add new/internal/template/render.go new/internal/template/render_test.go
git commit -m "feat(new): add template renderer with embed.FS and RenderContext"
```

---

## Chunk 3: Deploy Engine

### Task 9: Exec wrapper + pre-flight checks

**Files:**
- Create: `new/internal/engine/exec.go`
- Create: `new/internal/engine/exec_test.go`
- Create: `new/internal/engine/preflight.go`
- Create: `new/internal/engine/preflight_test.go`

- [ ] **Step 1: Write exec wrapper tests**

```go
// new/internal/engine/exec_test.go
package engine

import (
	"context"
	"testing"
	"time"
)

func TestRunWithTimeout_Success(t *testing.T) {
	err := RunWithTimeout(context.Background(), "echo-test", 5*time.Second, "echo", "hello")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
}

func TestRunWithTimeout_Timeout(t *testing.T) {
	err := RunWithTimeout(context.Background(), "sleep-test", 100*time.Millisecond, "sleep", "10")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestRunWithTimeout_BadCommand(t *testing.T) {
	err := RunWithTimeout(context.Background(), "bad-cmd", 5*time.Second, "nonexistent-command-xyz")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/engine/ -v -run TestRunWithTimeout`
Expected: FAIL

- [ ] **Step 3: Write exec.go**

Write `new/internal/engine/exec.go` with:
- `func RunWithTimeout(ctx context.Context, name string, timeout time.Duration, cmd string, args ...string) error`
- `func RunWithOutput(ctx context.Context, name string, timeout time.Duration, cmd string, args ...string) (string, error)` — returns stdout
- `func DockerExec(ctx context.Context, composeDir, service string, cmd ...string) (string, error)` — runs `docker compose exec {service} {cmd...}`
- `func DockerRun(ctx context.Context, composeDir, service string, cmd ...string) (string, error)` — runs `docker compose run --rm {service} {cmd...}`

All functions wrap `exec.CommandContext` with timeout, capture stdout/stderr, return structured errors.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/engine/ -v -run TestRunWithTimeout`
Expected: All PASS

- [ ] **Step 5: Write pre-flight tests**

```go
// new/internal/engine/preflight_test.go
package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
)

func TestPreflightChecks_BuiltinRequiresHelm(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "builtin"
	checks := RequiredTools(cfg)

	found := false
	for _, c := range checks {
		if c.Name == "helm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("builtin K8s should require helm")
	}
}

func TestPreflightChecks_ExternalSkipsHelm(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "external"
	checks := RequiredTools(cfg)

	for _, c := range checks {
		if c.Name == "helm" {
			t.Error("external K8s should not require helm")
		}
	}
}
```

- [ ] **Step 6: Write preflight.go**

Write `new/internal/engine/preflight.go` with:
- `type ToolCheck struct { Name string; Required bool; CheckCmd string; CheckArgs []string }`
- `func RequiredTools(cfg *config.Config) []ToolCheck` — returns list based on config
- `func RunPreflightChecks(cfg *config.Config) error` — executes each check, returns first failure with helpful message

- [ ] **Step 7: Run all tests**

Run: `cd new && go test ./internal/engine/ -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add new/internal/engine/
git commit -m "feat(new): add exec wrapper and pre-flight checks"
```

---

### Task 10: Engine orchestration + step system

**Files:**
- Create: `new/internal/engine/engine.go`
- Create: `new/internal/engine/engine_test.go`
- Create: `new/internal/engine/steps.go`
- Create: `new/internal/engine/health.go`

- [ ] **Step 1: Write engine tests with mocked executor**

```go
// new/internal/engine/engine_test.go
package engine

import (
	"context"
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func TestBuildStepList_BuiltinFull(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	sec, _ := secrets.Generate()

	e := New(cfg, sec, "/tmp/test-output", nil)
	steps := e.BuildStepList()

	// Builtin + selfsigned should have all steps
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}

	expected := []string{
		"Pre-flight checks",
		"Installing K3s",
		"Generating SSL certificates",
		"Installing Ingress-Nginx",
		"Configuring CoreDNS",
		"Setting up Kyverno + CA distribution",
	}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected step: %q", exp)
		}
	}
}

func TestBuildStepList_ExternalSkipsK3s(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "external"
	cfg.SSL.Mode = "letsencrypt"
	sec, _ := secrets.Generate()

	e := New(cfg, sec, "/tmp/test-output", nil)
	steps := e.BuildStepList()

	for _, s := range steps {
		if s.Name == "Installing K3s" {
			t.Error("external K8s should not have K3s step")
		}
		if s.Name == "Setting up Kyverno + CA distribution" {
			t.Error("letsencrypt should not have Kyverno step")
		}
	}
}

func TestDeploy_SendsEvents(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	sec, _ := secrets.Generate()
	events := make(chan StepEvent, 100)

	e := New(cfg, sec, t.TempDir(), events)
	// We can't actually deploy in tests, but verify the engine initializes
	steps := e.BuildStepList()
	if len(steps) == 0 {
		t.Error("step list should not be empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/engine/ -v -run TestBuildStepList`
Expected: FAIL

- [ ] **Step 3: Write engine.go**

Write `new/internal/engine/engine.go` with:
- `type StepStatus int` constants: `Pending`, `Running`, `Done`, `Failed`
- `type StepEvent struct { Step string; Status StepStatus; Detail string; Err error }`
- `type Step struct { Name string; Fn func(context.Context) error }`
- `type Engine struct { config, secrets, output, notify }`
- `func New(cfg, sec, output, notify) *Engine`
- `func (e *Engine) Deploy(ctx context.Context) error` — iterates steps, sends events, stops on error
- `func (e *Engine) notify(event StepEvent)` — nil-safe channel send

- [ ] **Step 4: Write steps.go**

Write `new/internal/engine/steps.go` with:
- `func (e *Engine) BuildStepList() []Step` — dynamic step list based on config (see spec lines 332 and 449-458 for logic)
- Stub functions for each step: `preflightCheck`, `installK3s`, `setupCerts`, `installIngress`, `configureCoreDNS`, `setupKyverno`, `startPostgres`, `startRedis`, `initDatabase`, `startGitea`, `bootstrapGitea`, `startServices`, `waitAPIHealth`, `waitUIHealth`, `waitRouterHealth`, `startNginx`, `importTemplates`, `registerCluster`, `showResult`
- Each stub logs what it would do and returns nil (real implementations in Chunk 7)
- Include `setupHosts` (manages /etc/hosts entries) and `configureDnsmasq` (Docker→K8s DNS routing) as named stubs

- [ ] **Step 5: Write health.go**

Write `new/internal/engine/health.go` with:
- `func WaitForHealth(ctx context.Context, service, url string, timeout time.Duration) error` — polls URL until 200 or timeout
- `func WaitForDocker(ctx context.Context, composeDir, service string, timeout time.Duration) error` — polls `docker compose ps` until healthy

- [ ] **Step 6: Run all tests**

Run: `cd new && go test ./internal/engine/ -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add new/internal/engine/
git commit -m "feat(new): add deploy engine with dynamic step list and event system"
```

---

### Task 11: Reconfigure logic

**Files:**
- Create: `new/internal/engine/reconfigure.go`
- Create: `new/internal/engine/reconfigure_test.go`

- [ ] **Step 1: Write reconfigure tests**

```go
// new/internal/engine/reconfigure_test.go
package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func TestApplyChanges_BuildsRestartList(t *testing.T) {
	old := config.DefaultConfig("example.com", "admin@example.com")
	new_ := config.DefaultConfig("example.com", "admin@example.com")
	new_.SMTP.Host = "smtp.example.com"

	sec, _ := secrets.Generate()
	events := make(chan StepEvent, 100)
	e := New(old, sec, t.TempDir(), events)

	steps := e.BuildReconfigureSteps(old, new_)
	if len(steps) == 0 {
		t.Error("SMTP change should produce reconfigure steps")
	}

	// Should include re-render and api restart
	hasRender := false
	hasRestart := false
	for _, s := range steps {
		if s.Name == "Re-rendering templates" {
			hasRender = true
		}
		if s.Name == "Restarting api" {
			hasRestart = true
		}
	}
	if !hasRender {
		t.Error("should include re-render step")
	}
	if !hasRestart {
		t.Error("should include api restart step")
	}
}
```

- [ ] **Step 2: Run tests, verify fail, write implementation, verify pass**

Write `new/internal/engine/reconfigure.go` with `func (e *Engine) BuildReconfigureSteps(old, new *config.Config) []Step` — uses `config.Diff` to determine changes, builds step list: re-render templates → restart affected services.

- [ ] **Step 3: Run tests**

Run: `cd new && go test ./internal/engine/ -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add new/internal/engine/reconfigure.go new/internal/engine/reconfigure_test.go
git commit -m "feat(new): add reconfigure logic with diff-based service restart"
```

---

## Chunk 4: TUI Components

### Task 12: Shared styles

**Files:**
- Create: `new/internal/tui/components/styles.go`

- [ ] **Step 1: Write shared lipgloss styles**

```go
// new/internal/tui/components/styles.go
package components

import "github.com/charmbracelet/lipgloss"

var (
	// Brand colors
	PrimaryColor   = lipgloss.Color("#6C5CE7")
	SuccessColor   = lipgloss.Color("#00B894")
	WarningColor   = lipgloss.Color("#FDCB6E")
	ErrorColor     = lipgloss.Color("#E74C3C")
	MutedColor     = lipgloss.Color("#555555")
	SubtitleColor  = lipgloss.Color("#888888")

	// Text styles
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8C291"))
	SubtitleStyle = lipgloss.NewStyle().Foreground(SubtitleColor)
	SuccessStyle  = lipgloss.NewStyle().Foreground(SuccessColor)
	ErrorStyle    = lipgloss.NewStyle().Foreground(ErrorColor)
	MutedStyle    = lipgloss.NewStyle().Foreground(MutedColor)

	// Component styles
	StepDone    = SuccessStyle.Render("✓")
	StepRunning = lipgloss.NewStyle().Foreground(WarningColor).Render("⠋")
	StepPending = MutedStyle.Render("○")
	StepFailed  = ErrorStyle.Render("✗")
)
```

- [ ] **Step 2: Verify compilation**

Run: `cd new && go build ./internal/tui/components/`
Expected: Compiles

- [ ] **Step 3: Commit**

```bash
git add new/internal/tui/components/
git commit -m "feat(new): add shared TUI styles with lipgloss"
```

---

### Task 13: Deploy progress TUI

**Files:**
- Create: `new/internal/tui/progress/progress.go`
- Create: `new/internal/tui/progress/step.go`

- [ ] **Step 1: Write step.go**

```go
// new/internal/tui/progress/step.go
package progress

import (
	"fmt"
	"time"

	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type StepView struct {
	Name     string
	Status   engine.StepStatus
	Duration time.Duration
	Detail   string
}

func (s StepView) Render() string {
	icon := components.StepPending
	switch s.Status {
	case engine.Done:
		icon = components.StepDone
	case engine.Running:
		icon = components.StepRunning
	case engine.Failed:
		icon = components.StepFailed
	}

	line := fmt.Sprintf("%s %s", icon, s.Name)
	if s.Status == engine.Done {
		line += components.MutedStyle.Render(fmt.Sprintf(" %s", s.Duration.Round(time.Second)))
	}
	if s.Status == engine.Running && s.Detail != "" {
		line += components.MutedStyle.Render(fmt.Sprintf(" %s", s.Detail))
	}
	if s.Status == engine.Failed {
		line += components.ErrorStyle.Render(fmt.Sprintf(" %s", s.Detail))
	}
	return line
}
```

- [ ] **Step 2: Write progress.go — the Bubbletea Model**

Write `new/internal/tui/progress/progress.go` with:
- `type Model struct` containing: steps []StepView, current int, spinner spinner.Model, progress progress.Model, startTime time.Time, elapsed time.Duration, done bool, err error
- `func New(stepNames []string) Model` — initializes with pending steps
- `type StepEventMsg engine.StepEvent` — message type for TUI
- `type TickMsg time.Time` — for elapsed timer
- `func (m Model) Init() tea.Cmd` — start spinner + timer
- `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)` — handle StepEventMsg (update step status), TickMsg (update elapsed), KeyMsg (q to quit on failure)
- `func (m Model) View() string` — render title, step list, progress bar, elapsed time. Format matches Style I from spec.
- `func WaitForEvents(ch <-chan engine.StepEvent) tea.Cmd` — converts channel events to tea.Msg

- [ ] **Step 3: Verify compilation**

Run: `cd new && go build ./internal/tui/progress/`
Expected: Compiles

- [ ] **Step 4: Commit**

```bash
git add new/internal/tui/progress/
git commit -m "feat(new): add deploy progress TUI with sequential list display"
```

---

### Task 14: Install wizard — Express mode

**Files:**
- Create: `new/internal/tui/wizard/wizard.go`
- Create: `new/internal/tui/wizard/express.go`

- [ ] **Step 1: Write express.go**

Write `new/internal/tui/wizard/express.go` with:
- `func NewExpressForm() *huh.Form` — creates a huh.Form with one Group:
  - `huh.NewInput().Title("Domain name").Placeholder("example.com").Validate(domainValidator)`
  - `huh.NewInput().Title("Admin email").Placeholder("admin@example.com").Validate(emailValidator)`
  - `huh.NewConfirm().Title("Deploy with defaults?").Description("Self-signed SSL, built-in K3s + PostgreSQL + Redis")`
- `func domainValidator(s string) error` — validates FQDN format
- `func emailValidator(s string) error` — validates email format
- Returns `*config.Config` via `ExpressResult()` after form completes

- [ ] **Step 2: Write wizard.go — the mode switcher**

Write `new/internal/tui/wizard/wizard.go` with:
- `type Mode int` — `Express`, `Custom`
- `type Model struct` containing: mode Mode, expressForm *huh.Form, customForm *huh.Form, result *config.Config, switchToCustom bool, done bool
- `func New(mode Mode) Model`
- `func (m Model) Init() tea.Cmd`
- `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)` — delegates to express or custom form; if express user picks "Custom →", switches mode
- `func (m Model) View() string` — renders brand header + current form
- `func (m Model) Result() *config.Config` — returns the collected config

- [ ] **Step 3: Verify compilation**

Run: `cd new && go build ./internal/tui/wizard/`
Expected: Compiles

- [ ] **Step 4: Commit**

```bash
git add new/internal/tui/wizard/
git commit -m "feat(new): add Express install wizard with huh forms"
```

---

### Task 15: Install wizard — Custom mode (Groups 1-4)

**Files:**
- Create: `new/internal/tui/wizard/custom.go`

- [ ] **Step 1: Write custom.go with Groups 1-4**

Write `new/internal/tui/wizard/custom.go` with `func NewCustomForm() *huh.Form` starting with 4 groups:

**Group 1 — Basic:** Domain input (with validation), Subdomain input (default "launchpad"), Admin email input (with validation), Image registry select (Default / Custom)

**Group 2 — SSL:** Mode select: selfsigned / letsencrypt / custom. Conditional fields (DNS provider + token for LE, cert paths for custom) — use `huh.WithHideFunc` to show/hide based on mode selection

**Group 3 — Database:** Mode select: builtin / external. Conditional: 7 connection URL inputs for external

**Group 4 — Kubernetes:** Mode select: builtin / external. Conditional: kubeconfig path + context for external

- [ ] **Step 2: Verify compilation**

Run: `cd new && go build ./internal/tui/wizard/`
Expected: Compiles

- [ ] **Step 3: Commit**

```bash
git add new/internal/tui/wizard/custom.go
git commit -m "feat(new): add Custom wizard Groups 1-4 (Basic, SSL, DB, K8s)"
```

---

### Task 15b: Install wizard — Custom mode (Groups 5-6 + tests)

**Files:**
- Modify: `new/internal/tui/wizard/custom.go`
- Create: `new/internal/tui/wizard/wizard_test.go`

- [ ] **Step 1: Add Group 5 (Advanced) and Group 6 (Review)**

**Group 5 — Advanced:** SMTP fields (host, port, user, password, from), SSO fields (Entra tenant, client, secret), Storage select (local / s3 / azure) + conditional sub-fields, Telegram fields (bot token, username, chat ID), AI fields (CRS2 endpoint, token), Stripe fields (secret key, webhook secret, publishable key), Performance (API replicas, DB connection limit)

**Group 6 — Review:** Summary display of all config values, Final confirm

- [ ] **Step 2: Write wizard TUI tests**

```go
// new/internal/tui/wizard/wizard_test.go
package wizard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestExpressWizard_InitialState(t *testing.T) {
	m := New(Express)
	if m.mode != Express {
		t.Errorf("mode = %v, want Express", m.mode)
	}
	if m.done {
		t.Error("wizard should not be done initially")
	}
}

func TestCustomWizard_InitialState(t *testing.T) {
	m := New(Custom)
	if m.mode != Custom {
		t.Errorf("mode = %v, want Custom", m.mode)
	}
}

func TestWizard_ViewRendersWithoutPanic(t *testing.T) {
	m := New(Express)
	m.Init()
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd new && go test ./internal/tui/wizard/ -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add new/internal/tui/wizard/
git commit -m "feat(new): complete Custom wizard with Advanced + Review groups and tests"
```

---

## Chunk 5: CLI Integration

### Task 16: Configure panel TUI

**Files:**
- Create: `new/internal/tui/panel/panel.go`
- Create: `new/internal/tui/panel/tab.go`
- Create: `new/internal/tui/panel/tabs/overview.go`
- Create: `new/internal/tui/panel/tabs/ssl.go`
- Create: `new/internal/tui/panel/tabs/database.go`
- Create: `new/internal/tui/panel/tabs/smtp.go`
- Create: `new/internal/tui/panel/tabs/sso.go`
- Create: `new/internal/tui/panel/tabs/storage.go`
- Create: `new/internal/tui/panel/tabs/telegram.go`
- Create: `new/internal/tui/panel/tabs/ai.go`

- [ ] **Step 1: Write tab.go**

Write `new/internal/tui/panel/tab.go` with:
- `type Tab struct { Name string; Form *huh.Form; Dirty bool }`
- `type TabBar struct { tabs []Tab; active int }`
- `func (tb *TabBar) Render() string` — renders horizontal tab bar with active highlight
- `func (tb *TabBar) Next()`, `func (tb *TabBar) Prev()` — cycle tabs

- [ ] **Step 2: Write overview tab**

Write `new/internal/tui/panel/tabs/overview.go` — read-only overview showing: domain, status, SSL mode, DB mode, K8s mode, SMTP status, SSO status, Storage mode. Each field shows current value and `[change]` hint.

- [ ] **Step 3: Write remaining tabs**

Write each tab file (`ssl.go`, `database.go`, `smtp.go`, `sso.go`, `storage.go`, `telegram.go`, `ai.go`) as `func New{Tab}Form(cfg *config.Config) *huh.Form` — returns editable huh.Form populated with current config values.

- [ ] **Step 4: Write panel.go — main model**

Write `new/internal/tui/panel/panel.go` with:
- `type Model struct` containing: tabBar TabBar, config *config.Config, originalConfig *config.Config, pendingChanges int, applying bool, progressModel progress.Model
- `func New(cfg *config.Config) Model`
- `func (m Model) Update(msg tea.Msg)` — handle left/right for tab switch, 'a' for apply, 'q' for quit. On apply: compute diff, build reconfigure steps, switch to progress view.
- `func (m Model) View() string` — render tab bar + current tab form + status bar showing pending changes count

- [ ] **Step 5: Verify compilation**

Run: `cd new && go build ./internal/tui/panel/`
Expected: Compiles

- [ ] **Step 6: Commit**

```bash
git add new/internal/tui/panel/
git commit -m "feat(new): add configure panel TUI with tab-based navigation"
```

---

### Task 17: CLI core commands (install + configure)

**Files:**
- Modify: `new/internal/cli/root.go`
- Create: `new/internal/cli/install.go`
- Create: `new/internal/cli/configure.go`

- [ ] **Step 1: Write install.go**

Write `new/internal/cli/install.go` with:
- `func newInstallCmd() *cobra.Command` with flags: `--custom`, `--config FILE`, `--resume`, `--dir DIR`
- Migration detection: if `.setup.conf` exists in dir, refuse to run with migration message
- Logic:
  1. If `--config`: load config from file, validate, skip wizard
  2. If `--resume`: load existing `.setup.yaml` from dir, skip wizard, go to deploy
  3. If `--custom`: launch wizard in Custom mode
  4. Default: launch wizard in Express mode
  5. After wizard: generate secrets → save config → render templates → deploy (engine.Deploy with progress TUI)

- [ ] **Step 2: Write configure.go**

Write `new/internal/cli/configure.go` with:
- `func newConfigureCmd() *cobra.Command` with subcommand `export`
- Main: load `.setup.yaml` → launch panel TUI
- Export: load config → write to specified file

- [ ] **Step 3: Update root.go to register install + configure**

- [ ] **Step 4: Verify build**

Run: `cd new && make build && ./bin/launchpad install --help`
Expected: Shows install flags

- [ ] **Step 5: Commit**

```bash
git add new/internal/cli/
git commit -m "feat(new): add install and configure CLI commands"
```

---

### Task 17b: CLI operational commands (status, upgrade, restart, uninstall, helpers)

**Files:**
- Create: `new/internal/cli/status.go`
- Create: `new/internal/cli/upgrade.go`
- Create: `new/internal/cli/restart.go`
- Create: `new/internal/cli/uninstall.go`
- Create: `new/internal/cli/helpers.go`
- Modify: `new/internal/cli/root.go`

- [ ] **Step 1: Write status.go**

`func newStatusCmd() *cobra.Command` — load config → `docker compose ps` → format status table

- [ ] **Step 2: Write upgrade.go**

`func newUpgradeCmd() *cobra.Command` with `--version` flag — update image versions → re-render → pull → up

- [ ] **Step 3: Write restart.go**

`func newRestartCmd() *cobra.Command` with optional `[service]` arg

- [ ] **Step 4: Write uninstall.go**

`func newUninstallCmd() *cobra.Command` with `--all` flag — basic: `docker compose down`; `--all`: + K3s removal + /etc/hosts cleanup + dnsmasq cleanup + install dir removal

- [ ] **Step 5: Write helpers.go**

`newSetupCertsCmd()`, `newSetupDBCmd()`, `newImportTemplatesCmd()` — thin wrappers around engine functions

- [ ] **Step 6: Register all commands in root.go**

```go
cmd.AddCommand(
    newInstallCmd(), newConfigureCmd(),
    newStatusCmd(), newUpgradeCmd(), newRestartCmd(),
    newUninstallCmd(), newSetupCertsCmd(), newSetupDBCmd(),
    newImportTemplatesCmd(),
)
```

- [ ] **Step 7: Verify all commands**

Run: `cd new && make build && ./bin/launchpad --help`
Expected: All subcommands listed

- [ ] **Step 8: Commit**

```bash
git add new/internal/cli/
git commit -m "feat(new): add status, upgrade, restart, uninstall, and helper CLI commands"
```

---

## Chunk 6: Template Migration + Versions

### Task 18: Versions generator

**Files:**
- Create: `new/cmd/gen-versions/main.go`

- [ ] **Step 1: Write gen-versions tool**

Write `new/cmd/gen-versions/main.go` — a small CLI that:
1. Reads `deploy/versions.conf` (shell `KEY=VALUE` format)
2. Generates `internal/config/versions_gen.go` with typed constants:

```go
// Code generated by gen-versions. DO NOT EDIT.
package config

const (
	DefaultImageRegistry = "swr.ap-southeast-1.myhuaweicloud.com/ghisha"
	DefaultImageVersion  = "2.0.3"
	DefaultAPIVersion    = "2.0.3"
	DefaultUIVersion     = "2.0.3"
	DefaultRouterVersion = "1.16.2"
	DefaultGatewayVersion = "1.18.7"
	DefaultGiteaVersion  = "1.25-rootless"
)
```

- [ ] **Step 2: Add go:generate directive**

Add to `new/internal/config/config.go`:
```go
//go:generate go run ../../cmd/gen-versions -input ../../../deploy/versions.conf -output versions_gen.go
```

- [ ] **Step 3: Run generate and verify**

Run: `cd new && go generate ./internal/config/`
Expected: `versions_gen.go` created with correct constants

- [ ] **Step 4: Update DefaultConfig to use generated constants**

Update `DefaultConfig()` in `config.go` to use `DefaultImageRegistry`, `DefaultImageVersion`, etc. from `versions_gen.go`.

- [ ] **Step 5: Run all tests**

Run: `cd new && go test ./... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add new/cmd/gen-versions/ new/internal/config/versions_gen.go new/internal/config/config.go
git commit -m "feat(new): add go:generate tool for versions.conf"
```

---

### Task 19: Migrate simple template files

**Files:**
- Modify: `new/internal/template/files/env.tmpl` (complete migration)
- Create: `new/internal/template/files/env.gateway.tmpl`
- Create: `new/internal/template/files/nginx.conf.tmpl`
- Create: `new/internal/template/files/settings.yml.tmpl`
- Create: `new/internal/template/files/coredns-custom.yaml.tmpl`
- Create: `new/internal/template/files/kyverno-inject-ca.yaml.tmpl`
- Create: `new/internal/template/files/kyverno-sync-ca.yaml.tmpl`
- Copy: `new/internal/template/files/values-builtin.yml`
- Copy: `new/internal/template/files/crontab`
- Create: `new/internal/template/testdata/` (golden files)

- [ ] **Step 1: Migrate env.tmpl**

Translate `deploy/templates/env.template` from `${VAR}` syntax to `{{ .Field }}` syntax. Map every variable to the corresponding Config/Secrets/Derived field. Reference `deploy/scripts/lib/render.sh` for the full variable list.

- [ ] **Step 2: Migrate env.gateway.tmpl, nginx.conf.tmpl, settings.yml.tmpl**

Translate each from `deploy/templates/`. nginx.conf has conditional blocks for SSL modes — use `{{ if eq .Config.SSL.Mode "selfsigned" }}`.

- [ ] **Step 3: Migrate K8s templates**

Translate `coredns-custom.yaml.template`, `kyverno-inject-ca.yaml`, `kyverno-sync-ca.yaml`.

- [ ] **Step 4: Copy static files**

Copy `values-builtin.yml` and `crontab` as-is.

- [ ] **Step 5: Create golden file tests**

Create `new/internal/template/testdata/` with expected output files. Write golden file test:

```go
func TestRenderAll_GoldenFile(t *testing.T) {
	ctx := fixtureRenderContext() // fixed config, fixed secrets (deterministic)
	outDir := t.TempDir()
	RenderAll(ctx, outDir)

	// Compare .env output against golden file
	got, _ := os.ReadFile(filepath.Join(outDir, ".env"))
	want, _ := os.ReadFile("testdata/env.golden")
	if string(got) != string(want) {
		t.Errorf(".env mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
```

- [ ] **Step 6: Run tests**

Run: `cd new && go test ./internal/template/ -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add new/internal/template/
git commit -m "feat(new): migrate simple templates from envsubst to Go text/template"
```

---

### Task 19b: Migrate docker-compose.yml template

**Files:**
- Create: `new/internal/template/files/docker-compose.yml.tmpl`
- Create: `new/internal/template/testdata/docker-compose-builtin.golden`
- Create: `new/internal/template/testdata/docker-compose-external.golden`

This is the most complex template (~330 lines of conditional shell logic).

- [ ] **Step 1: Write golden file tests first**

```go
func TestRenderCompose_BuiltinMode(t *testing.T) {
	ctx := fixtureRenderContext() // builtin K8s + builtin DB + selfsigned
	outDir := t.TempDir()
	RenderAll(ctx, outDir)

	got, _ := os.ReadFile(filepath.Join(outDir, "docker-compose.yml"))
	content := string(got)

	// Builtin DB should include postgres and redis services
	if !strings.Contains(content, "postgres:") {
		t.Error("builtin mode should include postgres service")
	}
	if !strings.Contains(content, "redis:") {
		t.Error("builtin mode should include redis service")
	}
	// Selfsigned should include CA cert volume mount
	if !strings.Contains(content, "ca-certificates") {
		t.Error("selfsigned mode should mount CA cert")
	}
	// Builtin K8s should include extra_hosts
	if !strings.Contains(content, "extra_hosts") {
		t.Error("builtin K8s should include extra_hosts")
	}
}

func TestRenderCompose_ExternalMode(t *testing.T) {
	ctx := fixtureRenderContext()
	ctx.Config.Database.Mode = "external"
	ctx.Config.Kubernetes.Mode = "external"
	ctx.Config.SSL.Mode = "letsencrypt"
	// recompute derived
	ctx.Derived = ComputeDerived(ctx.Config, ctx.Secrets)

	outDir := t.TempDir()
	RenderAll(ctx, outDir)

	got, _ := os.ReadFile(filepath.Join(outDir, "docker-compose.yml"))
	content := string(got)

	if strings.Contains(content, "postgres:") {
		t.Error("external DB should not include postgres service")
	}
	if strings.Contains(content, "extra_hosts") {
		t.Error("external K8s should not include extra_hosts")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd new && go test ./internal/template/ -v -run TestRenderCompose`
Expected: FAIL

- [ ] **Step 3: Write docker-compose.yml.tmpl**

Translate from `render_compose()` in `deploy/scripts/lib/render.sh`. Use Go template conditionals for: builtin vs external DB (postgres/redis services), builtin vs external K8s (extra_hosts), selfsigned SSL (CA cert volume mounts, NODE_TLS_REJECT env), all service definitions with environment variables.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd new && go test ./internal/template/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add new/internal/template/
git commit -m "feat(new): migrate docker-compose.yml template with full conditional logic"
```

---

## Chunk 7: Real Engine Step Implementations

### Task 20: Infrastructure steps (K3s, certs, ingress, CoreDNS, Kyverno, dnsmasq, /etc/hosts)

**Files:**
- Modify: `new/internal/engine/steps.go`
- Create: `new/internal/engine/hosts.go`
- Create: `new/internal/engine/dnsmasq.go`

- [ ] **Step 1: Implement installK3s**

Replace stub with real implementation: download K3s install script → `RunWithTimeout` with 120s → wait for k3s service to be ready. Include idempotency check: if `k3s` binary exists and service is running, return nil immediately.

Reference: `deploy/scripts/lib/k3s.sh` `install_k3s()`.

- [ ] **Step 2: Implement setupCerts**

Switch on `config.SSL.Mode`:
- `selfsigned`: generate CA + wildcard cert using `openssl` via `RunWithTimeout` (or native Go `crypto/x509` for CA generation — prefer Go native)
- `letsencrypt`: call certbot/acme via `RunWithTimeout`
- `custom`: copy cert files to nginx/certs/

Reference: `deploy/scripts/lib/certs.sh`.

- [ ] **Step 3: Implement installIngress**

`helm install ingress-nginx` via `RunWithTimeout` → wait for controller pod to be ready via `kubectl wait`.

Reference: `deploy/scripts/lib/k8s-components.sh` `install_ingress_nginx()`.

- [ ] **Step 4: Implement configureCoreDNS**

Apply rendered `coredns-custom.yaml` via `kubectl apply`. Wait for CoreDNS pods to restart.

Reference: `deploy/scripts/lib/k8s-components.sh` `configure_coredns()`.

- [ ] **Step 5: Implement setupKyverno**

Install Kyverno via `helm install` → apply rendered `kyverno-inject-ca.yaml` and `kyverno-sync-ca.yaml` via `kubectl apply`.

Reference: `deploy/scripts/lib/k8s-components.sh` `install_kyverno()`, `setup_cert_distribution()`.

- [ ] **Step 6: Implement setupHosts (hosts.go)**

Write `new/internal/engine/hosts.go` with:
- `func (e *Engine) setupHosts(ctx context.Context) error` — add entries to `/etc/hosts` for `{subdomain}.{domain}`, `launchpad-gitea.{subdomain}.{domain}` etc. pointing to host IP
- `func RemoveHosts(domain string) error` — remove previously added entries (used by uninstall)
- Idempotency: check if entries already exist before adding

Reference: `deploy/scripts/lib/deploy.sh` `setup_hosts()`.

- [ ] **Step 7: Implement configureDnsmasq (dnsmasq.go)**

Write `new/internal/engine/dnsmasq.go` with:
- `func (e *Engine) configureDnsmasq(ctx context.Context) error` — write config to `/etc/dnsmasq.d/launchpad.conf`, restart dnsmasq service
- `func RemoveDnsmasq() error` — cleanup (used by uninstall)

Reference: `deploy/scripts/lib/k3s.sh` dnsmasq configuration sections.

- [ ] **Step 8: Implement hostIP detection in derived.go**

Update `ComputeDerived()` to dynamically detect host IP (or accept it as a parameter). Use `net.InterfaceAddrs()` to find non-loopback IPv4 address.

- [ ] **Step 9: Commit**

```bash
git add new/internal/engine/
git commit -m "feat(new): implement infrastructure engine steps (K3s, certs, ingress, DNS, hosts)"
```

---

### Task 21: Data layer steps (PostgreSQL, Redis, database init)

**Files:**
- Modify: `new/internal/engine/steps.go`

- [ ] **Step 1: Implement startPostgres**

`docker compose up -d postgres` → `WaitForDocker(ctx, dir, "postgres", 30s)`. Idempotency: if container already running and healthy, skip.

- [ ] **Step 2: Implement startRedis**

`docker compose up -d redis` → `WaitForDocker(ctx, dir, "redis", 15s)`. Same idempotency.

- [ ] **Step 3: Implement initDatabase**

Run 6 Prisma migrations via `DockerRun(ctx, dir, "api", "npx", "prisma", "db", "push", "--schema", "prisma/{schema}.prisma")`. Also create Gitea database and user via `DockerExec(ctx, dir, "postgres", "psql", ...)`. Send detail events for each schema.

Idempotency: check if schemas exist before running (`SELECT schema_name FROM information_schema.schemata`).

Reference: `deploy/scripts/lib/database.sh` `init_database()`.

- [ ] **Step 4: Commit**

```bash
git add new/internal/engine/steps.go
git commit -m "feat(new): implement data layer engine steps (PG, Redis, DB init)"
```

---

### Task 22: Application steps (Gitea bootstrap, services, health checks, Nginx)

**Files:**
- Modify: `new/internal/engine/steps.go`
- Create: `new/internal/engine/gitea.go`

- [ ] **Step 1: Implement startGitea**

`docker compose up -d gitea` → `WaitForDocker(ctx, dir, "gitea", 90s)`.

- [ ] **Step 2: Implement bootstrapGitea (gitea.go)**

Write `new/internal/engine/gitea.go` with `func (e *Engine) bootstrapGitea(ctx context.Context) error`:
1. `DockerExec(gitea, "gitea", "admin", "user", "create", ...)` — create admin
2. HTTP POST to `localhost:3000/api/v1/users/{admin}/tokens` — generate token (use `net/http`)
3. HTTP POST to `localhost:3000/api/v1/orgs` — create `launchpad` org
4. Save token to RuntimeValues and persist to `.runtime.yaml`
5. Re-render `.env` with updated RuntimeValues

Idempotency: check if admin user exists via Gitea API before creating.

Reference: `deploy/scripts/lib/deploy.sh` `bootstrap_gitea()`.

- [ ] **Step 3: Implement startServices**

`docker compose up -d api ui router cron backup-worker gateway` — starts all application services.

- [ ] **Step 4: Implement health check steps**

`waitAPIHealth`: `WaitForHealth(ctx, "api", "http://localhost:6804/health", 90s)`
`waitUIHealth`: `WaitForHealth(ctx, "ui", "http://localhost:3001", 60s)`
`waitRouterHealth`: `WaitForHealth(ctx, "router", "http://localhost:8080", 60s)`

- [ ] **Step 5: Implement startNginx**

`docker compose up -d nginx` → `WaitForDocker(ctx, dir, "nginx", 30s)` → call `setupHosts()`.

- [ ] **Step 6: Commit**

```bash
git add new/internal/engine/
git commit -m "feat(new): implement application engine steps (Gitea, services, health, Nginx)"
```

---

### Task 23: Finalization steps (template import, cluster registration) + deploy log

**Files:**
- Create: `new/internal/engine/import.go`
- Create: `new/internal/engine/register.go`
- Create: `new/internal/engine/log.go`

- [ ] **Step 1: Implement importTemplates (import.go)**

Write `new/internal/engine/import.go` with `func (e *Engine) importTemplates(ctx context.Context) error`:
1. Extract `.tar.gz` files from embedded assets or `files/` directory
2. For each repo: create in Gitea via HTTP API → `git push` via HTTPS (with retry)
3. For each `.yaml`: validate via admin API → import as official template

Idempotency: check if repos already exist in Gitea before creating.

Reference: `deploy/scripts/lib/deploy.sh` `import_templates()`.

- [ ] **Step 2: Implement registerCluster (register.go)**

Write `new/internal/engine/register.go` — self-register the K8s cluster with the API.

Reference: `deploy/scripts/lib/deploy.sh` cluster registration section.

- [ ] **Step 3: Implement deploy log writing (log.go)**

Write `new/internal/engine/log.go` with:
- `type Logger struct { file *os.File }` — writes all exec output to `generated/deploy.log`
- Integrate with `RunWithTimeout`, `DockerExec`, `DockerRun` — tee stdout/stderr to log file
- On failure, the TUI shows: "Full log: generated/deploy.log"

- [ ] **Step 4: Implement showResult**

Print formatted output: URLs (dashboard, gitea, admin), credentials, K8s status, DNS setup instructions, next steps.

Reference: `deploy/scripts/lib/deploy.sh` `show_result()`.

- [ ] **Step 5: Commit**

```bash
git add new/internal/engine/
git commit -m "feat(new): implement template import, cluster registration, and deploy logging"
```

---

### Task 24: Progress TUI tests

**Files:**
- Create: `new/internal/tui/progress/progress_test.go`

- [ ] **Step 1: Write progress model tests**

```go
// new/internal/tui/progress/progress_test.go
package progress

import (
	"testing"

	"github.com/gradient8/launchpad/internal/engine"
)

func TestProgressModel_InitialState(t *testing.T) {
	m := New([]string{"Step A", "Step B", "Step C"})
	if len(m.steps) != 3 {
		t.Errorf("steps count = %d, want 3", len(m.steps))
	}
	for _, s := range m.steps {
		if s.Status != engine.Pending {
			t.Errorf("initial step status = %v, want Pending", s.Status)
		}
	}
}

func TestProgressModel_StepEventUpdates(t *testing.T) {
	m := New([]string{"Step A", "Step B"})
	m, _ = m.Update(StepEventMsg{Step: "Step A", Status: engine.Running})
	if m.(Model).steps[0].Status != engine.Running {
		t.Error("Step A should be Running")
	}
	m, _ = m.Update(StepEventMsg{Step: "Step A", Status: engine.Done})
	if m.(Model).steps[0].Status != engine.Done {
		t.Error("Step A should be Done")
	}
}

func TestProgressModel_ViewNotEmpty(t *testing.T) {
	m := New([]string{"Step A"})
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `cd new && go test ./internal/tui/progress/ -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add new/internal/tui/progress/progress_test.go
git commit -m "test(new): add progress TUI model tests"
```

---

### Task 25: End-to-end build verification

**Files:**
- No new files

- [ ] **Step 1: Run full test suite**

Run: `cd new && go test ./... -v`
Expected: All tests pass

- [ ] **Step 2: Build binary**

Run: `cd new && make build`
Expected: `bin/launchpad` binary created

- [ ] **Step 3: Verify CLI help**

Run: `cd new && ./bin/launchpad --help`
Expected: All subcommands listed (install, configure, status, upgrade, restart, uninstall, setup-certs, setup-db, import-templates)

Run: `./bin/launchpad install --help`
Expected: Shows install flags (--custom, --config, --resume, --dir)

- [ ] **Step 4: Verify --config FILE path (non-interactive install)**

Create a test config file and verify it loads:

```bash
cat > /tmp/test-config.yaml <<'EOF'
domain: test.example.com
admin_email: admin@test.example.com
ssl:
  mode: selfsigned
database:
  mode: builtin
kubernetes:
  mode: builtin
EOF
cd new && ./bin/launchpad install --config /tmp/test-config.yaml --dir /tmp/test-install
```

Expected: Loads config, validates, proceeds to deploy (will fail on pre-flight if Docker not available, but config loading should succeed)

- [ ] **Step 5: Build Linux binary**

Run: `cd new && make build-linux`
Expected: `bin/launchpad-linux-amd64` created

- [ ] **Step 6: Run lint**

Run: `cd new && make lint`
Expected: No issues

- [ ] **Step 7: Final commit**

```bash
git add -A new/
git commit -m "feat(new): complete Go TUI migration v1 — single binary with embedded templates"
```
