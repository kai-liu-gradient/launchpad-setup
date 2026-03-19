# AniLaunchpad Go TUI Migration — Design Spec

**Date:** 2026-03-19
**Status:** Draft
**Location:** `new/` directory (separate from existing shell implementation)

## Motivation

1. **Single binary distribution** — embed templates via `embed.FS`, eliminate external file dependencies; users download one binary instead of cloning a repo
2. **Robustness** — Go's type system, structured error handling, and testability replace fragile shell string manipulation
3. **Better rendering logic** — typed Config structs + Go templates replace `envsubst` with variable sprawl
4. **Mature TUI ecosystem** — Bubbletea is the de facto standard for complex terminal UIs in Go

## Migration Strategy

**Phased approach, TUI first (Approach B):**

- **Phase 1:** Migrate wizard interaction + config generation + template rendering + secrets to Go
- **Deployment execution** (k3s install, docker compose, helm, kubectl) stays as `os/exec` calls — these are inherently CLI-tool orchestration
- **Future phases:** Gradually replace `os/exec` calls with native Go implementations (e.g., Docker SDK) as needed

## CLI Command Structure

Subcommand style using Cobra:

```
launchpad install                  # Express install (domain + email, all defaults)
launchpad install --custom         # Full wizard
launchpad install --config FILE    # From config file
launchpad install --resume         # Resume failed deployment

launchpad configure                # Interactive config panel (Tab dashboard)
launchpad configure export FILE    # Export current config

launchpad status                   # Service status overview
launchpad upgrade                  # Upgrade image versions
launchpad restart [service]        # Restart service (all if omitted)
launchpad uninstall                # Uninstall services (keep config)
launchpad uninstall --all          # Full uninstall (K3s, config, everything)

launchpad setup-certs              # Standalone: regenerate certificates
launchpad setup-db                 # Standalone: initialize database
launchpad import-templates         # Standalone: import templates
```

## TUI Design

### Install — Express Mode

Only 2 required fields: domain and admin email. Everything else uses smart defaults (self-signed SSL, built-in K3s, built-in PostgreSQL + Redis). A "Custom →" escape hatch is available to enter full wizard mode.

Implementation: `charmbracelet/huh` Form with a single Group containing 2 Input fields + 1 Confirm.

### Install — Custom Mode

6-step wizard:

1. **Basic** — Domain, admin email, image registry
2. **SSL** — Mode selection (selfsigned/letsencrypt/custom), mode-specific sub-fields expand on selection
3. **Database** — Mode selection (builtin/external), external expands connection string inputs
4. **Kubernetes** — Mode selection (builtin/external), external expands kubeconfig + context
5. **Advanced** — Collapsible optional sections (SMTP, SSO, Storage, Telegram, AI, Performance)
6. **Review** — Configuration summary, confirm deploy

Implementation: `huh.Form` with multiple `huh.Group`s. Step 5 uses `huh.Select` to pick a category, then expands a sub-form.

Rationale for Express/Custom split: infrastructure decisions (SSL mode, DB mode, K8s mode) are hard to change post-install, so Custom mode surfaces these upfront. Service-level config (SMTP, SSO, etc.) can be added later via `configure`.

### Deploy Progress — Style I (Sequential List)

Simple sequential list with progress bar:

```
Deploying AniLaunchpad...

✓ Installing K3s                          32s
✓ Generating SSL certificates              2s
⠋ Installing Ingress-Nginx...
○ Configuring CoreDNS
○ ...

████████████░░░░░░░░  Step 3/14 · 34s
```

Custom Bubbletea Model with `bubbles/spinner` and `bubbles/progress`. The `engine` package sends `StepEvent` messages through a channel; the TUI Model receives them via `tea.Cmd`.

Step list is dynamically generated based on config (same logic as current shell — skip K3s steps if external, skip Kyverno if not selfsigned, etc.).

### Configure Panel

Tab-based dashboard launched via `launchpad configure`:

- Top: Tab bar (Overview, SSL, DB, SMTP, SSO, Storage, Telegram, AI)
- Center: Current tab content with editable fields (using `huh` forms)
- Bottom: Status bar showing pending changes count
- `a` key: Apply all pending changes (save config → re-render templates → restart affected services)
- Batch apply — users edit multiple settings, then apply once

Implementation: Custom Bubbletea Model managing tab state + dirty tracking. Each tab is a `huh.Form`. Apply triggers `engine.ApplyChanges()` which computes config diff, identifies affected services, and restarts them (reusing the progress component).

## Configuration Model

```go
type Config struct {
    Domain       string       `yaml:"domain" validate:"required,fqdn"`
    Subdomain    string       `yaml:"subdomain"`                        // prefix, default "launchpad"
    AdminEmail   string       `yaml:"admin_email" validate:"required,email"`
    Images       ImageConfig  `yaml:"images"`
    SSL          SSLConfig    `yaml:"ssl"`
    Database     DBConfig     `yaml:"database"`
    Kubernetes   K8sConfig    `yaml:"kubernetes"`
    SMTP         SMTPConfig   `yaml:"smtp,omitempty"`
    SSO          SSOConfig    `yaml:"sso,omitempty"`
    Storage      StorageConfig `yaml:"storage"`
    Telegram     TelegramConfig `yaml:"telegram,omitempty"`
    AI           AIConfig     `yaml:"ai,omitempty"`
    Stripe       StripeConfig `yaml:"stripe,omitempty"`
    Performance  PerfConfig   `yaml:"performance"`
}

type ImageConfig struct {
    Registry       string `yaml:"registry"`        // default: swr.ap-southeast-1.myhuaweicloud.com/ghisha
    DefaultVersion string `yaml:"default_version"` // default from embedded versions.conf
    API            string `yaml:"api,omitempty"`    // per-service overrides
    UI             string `yaml:"ui,omitempty"`
    Router         string `yaml:"router,omitempty"`
    Gateway        string `yaml:"gateway,omitempty"`
    Gitea          string `yaml:"gitea,omitempty"`
}

type SSLConfig struct {
    Mode            string `yaml:"mode"`              // "selfsigned" | "letsencrypt" | "custom"
    DNSProvider     string `yaml:"dns_provider,omitempty"`     // cloudflare | aliyun | azure | manual
    DNSAPIToken     string `yaml:"dns_api_token,omitempty"`
    CertPath        string `yaml:"cert_path,omitempty"`        // custom: fullchain
    KeyPath         string `yaml:"key_path,omitempty"`         // custom: private key
    WildcardCertPath string `yaml:"wildcard_cert_path,omitempty"` // custom: wildcard cert
    WildcardKeyPath  string `yaml:"wildcard_key_path,omitempty"`  // custom: wildcard key
}

type DBConfig struct {
    Mode string            `yaml:"mode"`  // "builtin" | "external"
    URLs map[string]string `yaml:"urls,omitempty"` // main, monitoring, events, billing, stats, gateway, gitea
}

type K8sConfig struct {
    Mode           string `yaml:"mode"`          // "builtin" | "external"
    Kubeconfig     string `yaml:"kubeconfig,omitempty"`
    Context        string `yaml:"context,omitempty"`
    StorageClass   string `yaml:"storage_class,omitempty"`
    DefaultBackend string `yaml:"default_backend,omitempty"`
}

type SMTPConfig struct {
    Host     string `yaml:"host,omitempty"`
    Port     int    `yaml:"port,omitempty"`
    User     string `yaml:"user,omitempty"`
    Password string `yaml:"password,omitempty"`
    From     string `yaml:"from,omitempty"`
}

type SSOConfig struct {
    EntraTenantID    string `yaml:"entra_tenant_id,omitempty"`
    EntraClientID    string `yaml:"entra_client_id,omitempty"`
    EntraSecret      string `yaml:"entra_secret,omitempty"`
}

type StorageConfig struct {
    Mode      string `yaml:"mode"`       // "local" | "s3" | "azure"
    S3Bucket  string `yaml:"s3_bucket,omitempty"`
    S3Region  string `yaml:"s3_region,omitempty"`
    S3Key     string `yaml:"s3_key,omitempty"`
    S3Secret  string `yaml:"s3_secret,omitempty"`
    AzureConn string `yaml:"azure_connection_string,omitempty"`
    AzureContainer string `yaml:"azure_container,omitempty"`
}

type TelegramConfig struct {
    BotToken    string `yaml:"bot_token,omitempty"`
    BotUsername string `yaml:"bot_username,omitempty"`
    ChatID      string `yaml:"chat_id,omitempty"`
}

type AIConfig struct {
    CRS2Endpoint string `yaml:"crs2_endpoint,omitempty"`
    CRS2Token    string `yaml:"crs2_token,omitempty"`
}

type StripeConfig struct {
    SecretKey      string `yaml:"secret_key,omitempty"`
    WebhookSecret  string `yaml:"webhook_secret,omitempty"`
    PublishableKey string `yaml:"publishable_key,omitempty"`
}

type PerfConfig struct {
    APIReplicas int `yaml:"api_replicas"` // default 1
    DBConnLimit int `yaml:"db_conn_limit"` // default 100
}
```

`DefaultConfig(domain, email)` returns a fully populated Config with all defaults for Express mode. Validation uses `go-playground/validator`. Default image versions are embedded from `versions.conf` at compile time.

Saved as `.setup.yaml` (structured YAML, replacing the shell-sourceable `.setup.conf`).

## Template Engine

### Embedding

All templates embedded via `embed.FS` in `internal/template/files/`. Templates migrated from `envsubst` (`${VAR}`) to Go `text/template` (`{{ .Field }}`).

Benefits over envsubst:
- Type safety — misspelled field names are compile-time errors
- Conditional logic — `{{ if eq .SSL.Mode "selfsigned" }}` replaces shell if/else string concatenation
- Testable — pass a Config struct, assert rendered output

### docker-compose.yml Rendering

The current `render_compose()` is the most complex rendering operation (~330 lines of procedural shell with conditional assembly). In Go, this becomes a single `docker-compose.yml.tmpl` using Go template conditionals:

```
{{ if eq .Config.Database.Mode "builtin" }}
  postgres:
    image: {{ .Derived.ImageRefs.postgres }}
    ...
{{ end }}
{{ if eq .Config.SSL.Mode "selfsigned" }}
    volumes:
      - {{ .Derived.CACertPath }}:/usr/local/share/ca-certificates/launchpad-ca.crt:ro
{{ end }}
{{ if eq .Config.Kubernetes.Mode "builtin" }}
    extra_hosts:
      - "{{ .Derived.ExternalDomain }}:{{ .Derived.HostIP }}"
{{ end }}
```

This replaces the shell's sed-based post-processing (extra_hosts injection, CA cert mounting) with declarative template logic. The template will be larger than other templates but is a single source of truth.

### Render Context

```go
type RenderContext struct {
    Config  *config.Config
    Secrets *secrets.Secrets
    Derived *DerivedValues
    Runtime *RuntimeValues    // values generated during deployment (Gitea token, etc.)
}

type DerivedValues struct {
    // URLs
    DashboardURL   string   // https://launchpad.{subdomain}.{domain}
    GiteaURL       string   // https://launchpad-gitea.{subdomain}.{domain}
    APIURL         string   // https://launchpad.{subdomain}.{domain}:6804
    ExternalDomain string   // {subdomain}.{domain}
    PublicURL      string
    CORSOrigins    string
    InternalAPIURL string
    InternalAdminURL string

    // Database
    DBConnStrings  map[string]string  // per-schema connection strings
    GiteaDBHost    string             // parsed from Gitea DB URL
    GiteaDBPort    string
    GiteaDBName    string
    GiteaDBUser    string

    // Images
    ImageRefs      map[string]string  // per-service full image references

    // Network
    HostIP         string             // dynamically detected host IP
    DefaultBackend string             // computed for builtin K8s mode
    NodeTLSReject  string             // "0" if selfsigned, "" otherwise

    // Paths
    CACertPath     string             // CA cert path for volume mounts
    RouterLocalURL string
    GiteaGitSSH    string
    EntraRedirectURI string
    ANICodeReleaseURL string
}

// RuntimeValues holds state generated during deployment that must
// survive re-renders (e.g., Gitea bootstrap creates a token).
type RuntimeValues struct {
    GiteaAccessToken string `yaml:"gitea_access_token,omitempty"`
    GiteaUser        string `yaml:"gitea_user,omitempty"`
}
```

`ComputeDerived(cfg, secrets)` encapsulates all URL/connection-string assembly logic currently spread across `render_templates()` in shell.

`RenderAll(ctx, outputDir)` iterates embedded `.tmpl` files, executes each with the RenderContext, writes to output directory. Before re-rendering `.env`, it loads existing `RuntimeValues` from the current `.env` to preserve runtime-generated state (Gitea token, etc.).

### Template Files

```
internal/template/files/
├── env.tmpl
├── env.gateway.tmpl
├── nginx.conf.tmpl
├── settings.yml.tmpl
├── docker-compose.yml.tmpl        # largest template, heavy conditionals
├── coredns-custom.yaml.tmpl
├── kyverno-inject-ca.yaml.tmpl
├── kyverno-sync-ca.yaml.tmpl
├── values-builtin.yml
└── crontab
```

## Deploy Engine

### Architecture

`engine.Engine` orchestrates deployment by executing steps sequentially. Each step is a function that calls external tools via `os/exec`. Progress events are sent to the TUI through a channel.

```go
type Engine struct {
    config  *config.Config
    secrets *secrets.Secrets
    output  string
    notify  chan<- StepEvent
}

type StepEvent struct {
    Step   string
    Status StepStatus  // running | done | failed
    Detail string
    Err    error
}
```

Step list is built dynamically based on config (builtin vs external K8s/DB, SSL mode).

### exec Wrapper

```go
// Run a host command with timeout
func RunWithTimeout(ctx context.Context, name string, timeout time.Duration,
    cmd string, args ...string) error

// Run a command inside a running Docker container
func DockerExec(ctx context.Context, service string, cmd ...string) (string, error)

// Run a one-off container command (docker compose run --rm)
func DockerRun(ctx context.Context, service string, cmd ...string) (string, error)
```

Wraps `os/exec.CommandContext` with timeout, stdout/stderr capture, and structured error reporting. `DockerExec` and `DockerRun` are needed for Gitea bootstrap (container-exec commands) and Prisma migrations (one-off container runs).

### Gitea Bootstrap Flow

After the Gitea container is healthy, `bootstrapGitea()` performs:

1. `DockerExec(gitea, "gitea", "admin", "user", "create", ...)` — create admin user
2. HTTP POST to `localhost:3000/api/v1/users/{admin}/tokens` — generate access token
3. HTTP POST to `localhost:3000/api/v1/orgs` — create `launchpad` organization
4. Store token in `RuntimeValues.GiteaAccessToken` and persist to `.secrets`

This is a multi-step process that involves container-exec + HTTP API calls + mutating runtime state. The engine handles this as a single Step with sub-operations.

### Database Migrations (Prisma)

Database schema initialization runs 6 Prisma migrations via one-off container:

```go
func (e *Engine) initDatabase(ctx context.Context) error {
    schemas := []string{"main", "monitoring", "events", "billing", "stats", "gateway"}
    for _, schema := range schemas {
        err := DockerRun(ctx, "api",
            "npx", "prisma", "db", "push", "--schema", "prisma/"+schema+".prisma")
        if err != nil {
            return fmt.Errorf("schema %s: %w", schema, err)
        }
        e.notifyDetail(fmt.Sprintf("Schema %s initialized", schema))
    }
    return nil
}
```

### Template Import

`importTemplates()` handles:
1. Extract `.tar.gz` files from embedded `files/` assets
2. For each repo: create in Gitea via API → `git push` via HTTPS
3. For each `.yaml`: validate via admin API → import as official template

Both the template archives and import logic are part of the embedded binary.

### Reconfigure Flow

`ApplyChanges(old, new)` computes config diff → identifies affected services → re-renders templates → restarts affected services. Reuses the progress TUI component.

## Secrets Management

```go
type Secrets struct {
    JWTSecret                     string `yaml:"jwt_secret"`          // access token signing
    JWTRefreshSecret              string `yaml:"jwt_refresh_secret"`  // refresh token signing
    SessionSecret                 string `yaml:"session_secret"`
    EncryptionKey                 string `yaml:"encryption_key"`
    ClaudeCredentialsEncryptionKey string `yaml:"claude_credentials_encryption_key"`
    SSHKeyEncryptionSecret        string `yaml:"ssh_key_encryption_secret"`
    RSAPublicKey                  string `yaml:"rsa_public_key"`
    RSAPrivateKey                 string `yaml:"rsa_private_key"`
    DBPasswordMain                string `yaml:"db_password_main"`
    DBPasswordMonitoring          string `yaml:"db_password_monitoring"`
    DBPasswordEvents              string `yaml:"db_password_events"`
    DBPasswordBilling             string `yaml:"db_password_billing"`
    DBPasswordStats               string `yaml:"db_password_stats"`
    DBPasswordGateway             string `yaml:"db_password_gateway"`
    DBPasswordGitea               string `yaml:"db_password_gitea"`
    PostgresSuperuserPassword     string `yaml:"postgres_superuser_password"`
    RedisPassword                 string `yaml:"redis_password"`
    GatewayAPIKey                 string `yaml:"gateway_api_key"`
    InternalSecret                string `yaml:"internal_secret"`
    AdminPassword                 string `yaml:"admin_password"`
}
```

`GiteaAccessToken` is stored only in `RuntimeValues` (not in Secrets) because it is generated at runtime during Gitea bootstrap, not during initial secret generation. The Secrets file holds generation-time secrets; RuntimeValues holds deployment-time state. `RuntimeValues` is persisted to `.runtime.yaml` alongside `.secrets`.

Each field has a specific consumer and is not interchangeable (shell name mapping: `RSAPublicKey` → `ZT_PUBLIC_KEY`, `RSAPrivateKey` → `ZT_PRIVATE_KEY`). Generated using Go `crypto/rand` + `crypto/rsa` (native, no `openssl` dependency). Saved to `.secrets` with `chmod 600`.

## Pre-flight Checks

Before deployment, the engine validates required external tools:

| Tool | Required | Notes |
|------|----------|-------|
| Docker + Docker Compose v2 | Always | Core runtime |
| helm | Builtin K8s only | Installs ingress-nginx |
| jq | No (removed) | Replaced by Go native JSON parsing |
| dnsmasq | Builtin K8s only | Docker→K8s DNS routing |
| kubectl | Builtin K8s only | Shipped with K3s |

`envsubst` is no longer needed (replaced by Go template engine). Pre-flight check is a dedicated engine step that runs before any deployment.

## Host-Level Side Effects

The deployment modifies host state beyond the install directory:

- **`/etc/hosts`**: Adds entries for `*.{subdomain}.{domain}` pointing to the host IP. Removed on `uninstall --all`.
- **dnsmasq**: Configures Docker-to-K8s DNS routing for builtin mode. Adds config to `/etc/dnsmasq.d/` or equivalent.
- **K3s**: Installs as a system service (builtin mode only). Removed on `uninstall --all`.

These are tracked in the engine steps and have corresponding cleanup in the uninstall flow.

## Resume Strategy

`launchpad install --resume` recovers from failed deployments using **implicit idempotency** (same approach as current shell):

Each step checks if its work is already done before executing:
- `installK3s`: checks if `k3s` binary exists and service is running
- `setupCerts`: checks if cert files exist and are valid
- `installIngress`: checks if ingress-nginx helm release exists
- `initDatabase`: checks if schemas exist (`SELECT schema_name ...`)
- `bootstrapGitea`: checks if admin user and org exist via API
- `importTemplates`: checks if repos exist in Gitea

No checkpoint file needed. The engine runs the full step list; completed steps are no-ops that report `done` immediately.

## Failure Handling

On step failure:
1. Engine sends `StepEvent{Status: Failed, Err: err}` to TUI
2. TUI displays the failed step with error message
3. Suggests: `launchpad install --resume` to retry, or `docker compose logs <service>` for diagnostics
4. Writes full deployment log to `generated/deploy.log`

## Data Flow

```
User input (TUI)
    → Config struct (validate + fill defaults)
    → Secrets.Generate() → .secrets file
    → Config.Save() → .setup.yaml
    → Pre-flight checks (Docker, helm, jq, dnsmasq)
    → RenderContext{Config, Secrets, DerivedValues}
    → template.RenderAll() → generated/ directory
    → Engine.Deploy() → os/exec external commands
        → Gitea bootstrap → RuntimeValues (token)
        → Re-render .env with RuntimeValues
    → StepEvent channel → TUI progress updates
```

## Runtime File Layout

```
/opt/launchpad/                   # Default install directory (configurable)
├── .setup.yaml                   # Config (non-sensitive)
├── .secrets                      # Secrets (chmod 600)
├── generated/                    # Rendered files
│   ├── docker-compose.yml
│   ├── .env
│   ├── nginx.conf
│   ├── settings.yml
│   └── k8s/
├── nginx/certs/                  # SSL certificates
├── launchpad/                    # API config
├── gateway/                      # Gateway config
└── heartbeat/                    # K3s heartbeat
```

## Project Structure

```
new/
├── cmd/launchpad/main.go
├── internal/
│   ├── cli/                      # Cobra commands
│   │   ├── root.go
│   │   ├── install.go
│   │   ├── configure.go
│   │   ├── status.go
│   │   ├── upgrade.go
│   │   ├── restart.go
│   │   ├── uninstall.go
│   │   └── helpers.go
│   ├── tui/
│   │   ├── wizard/               # Install wizard (express + custom)
│   │   ├── progress/             # Deploy progress display
│   │   ├── panel/                # Configure panel (tabs + forms)
│   │   └── components/           # Shared lipgloss styles
│   ├── config/                   # Config struct, validation, diff, I/O
│   ├── template/                 # embed.FS + render engine
│   │   └── files/                # Embedded template files
│   ├── engine/                   # Deploy orchestration + os/exec
│   └── secrets/                  # Secret generation + I/O
├── go.mod
├── Makefile
└── .goreleaser.yml               # Optional: cross-platform release
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/huh` | Form components |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | spinner, progress bar |
| `gopkg.in/yaml.v3` | Config serialization |
| `github.com/go-playground/validator/v10` | Config validation |

## Language

English only for v1. i18n can be added later via Go's `golang.org/x/text` or a message catalog pattern.

## Testing Strategy

| Layer | Approach |
|-------|----------|
| **Config validation** | Unit tests: valid/invalid configs, default population, diff computation |
| **Template rendering** | Unit tests: render each template with known Config, assert output matches expected |
| **DerivedValues** | Unit tests: compute derived values from sample configs, assert correctness |
| **Secrets generation** | Unit tests: verify format, length, uniqueness |
| **Engine steps** | Integration tests: mock `os/exec` via interface, verify step ordering and error propagation |
| **TUI components** | Use `bubbletea`'s test framework (`tea.Program.Send`) for wizard flow and key handling |
| **E2E** | Adapt existing SSH-based test cases (`tests/cases/`) to invoke the Go binary instead of `setup.sh` |

## Versions Configuration

Default image versions are embedded by converting `versions.conf` (shell `KEY=VALUE` format) to a Go source file at build time via `go generate`:

```go
//go:generate go run ./cmd/gen-versions -input ../../deploy/versions.conf -output internal/config/versions_gen.go
```

This generates a `versions_gen.go` containing typed constants. The `upgrade` command updates these versions in the saved `.setup.yaml` and re-renders templates.

## Migration from Shell Installations

Out of scope for v1 as automated tooling. For manual migration: operators can export their current config from `.setup.conf` + `.secrets` and recreate as `.setup.yaml` + new `.secrets` format. The v1 binary will refuse to run if it detects old-format `.setup.conf` in the install directory, with a message explaining the manual migration path.

## Out of Scope (for v1)

- Native Docker SDK integration (stays as `os/exec` calling `docker compose`)
- Native Helm SDK integration (stays as `os/exec` calling `helm`)
- Chinese language support
- Web-based UI alternative
- Automated migration tooling from shell-based installations
