# AniLaunchpad One-Click Deployment Script Design

## Overview

A modular, interactive Bash deployment script that automates the full deployment of the AniLaunchpad platform on a single Linux server. Supports both Chinese and English, with sensible defaults for quick setup and optional advanced configuration for power users.

## Target Users

Primary: Operations engineers with basic Linux and Docker experience, unfamiliar with Launchpad internals.
Stretch goal: Non-technical users who can follow prompts.

## Architecture

### Single-Server Full Stack

All services run on one machine. Nginx handles TLS termination and reverse proxying for three domains:

```
User Browser
    │
    ▼
┌─────────────────────────────────────────────────┐
│  Nginx (Docker container, :443/:80)              │
│  ├─ launchpad.{sub}.{domain}  → API/UI/Admin    │
│  ├─ launchpad-gitea.{sub}.{domain} → Gitea      │
│  └─ *.{domain} → Router → k3s/K8s Pods         │
└────────────────┬────────────────────────────────┘
                 │ Docker Network
    ┌────────────┼────────────────────┐
    ▼            ▼                    ▼
┌────────┐ ┌─────────┐ ┌──────────────────┐
│Launchpad│ │ Gateway │ │   Infrastructure │
│ API/UI  │ │  :6555  │ │ PostgreSQL Redis │
│ Router  │ │         │ │ Gitea   k3s      │
│ Cron    │ │         │ │                  │
└────────┘ └─────────┘ └──────────────────┘
```

### Key Decisions

1. **Single-layer Nginx** — TLS termination + reverse proxy in one Docker container, replacing the previous two-layer setup (external VM Nginx + internal Docker Nginx).
2. **Three server blocks**:
   - `launchpad.{sub}.{domain}:443` — Merged outer+inner config (Terminal WS, SSE, Socket.io, Gateway proxy, Admin, UI)
   - `launchpad-gitea.{sub}.{domain}:443` — Proxy to Gitea container `:3000`
   - `*.{domain}:443` — Proxy to Router `:6580` (WebSocket support)
3. **HTTP→HTTPS redirect** — Port 80 returns 301 to 443.
4. **k3s** — Disable built-in Traefik to avoid port conflicts; used only as container runtime for projects.

## Interactive Flow

### Language Selection

```
═══════════════════════════════════════════
  AniLaunchpad Setup Script v1.0
═══════════════════════════════════════════

  Language / 语言选择:
    1) English
    2) 中文
  Please select / 请选择 [1]:
```

### Step 1: Domain Configuration

```
[1/5] Basic Configuration
  Main domain (e.g. company.com):
  Subdomain prefix [corp]:

  The following domains will be generated:
    ✓ launchpad.corp.company.com      (Dashboard)
    ✓ launchpad-gitea.corp.company.com (Gitea)
    ✓ *.company.com                    (Project access)
  Confirm? [Y/n]:
```

### Step 2: SSL Certificate

```
[2/5] SSL Certificate
  Certificate method:
    1) Let's Encrypt auto-apply (recommended)
    2) Provide certificate files
  Select [1]:

  # If 1:
  DNS provider:
    1) Cloudflare
    2) Alibaba Cloud DNS
    3) Azure DNS
    4) Other (manual DNS verification)
  Select [1]:
  Cloudflare API Token:
```

- Main site + Gitea: standard certificate or same wildcard
- `*.{domain}` wildcard: **must use DNS-01 challenge**
- Tool: `acme.sh` (zero-dependency, built-in DNS provider plugins)

### Step 3: Database

```
[3/5] Database
  Database mode:
    1) Built-in PostgreSQL + Redis (recommended)
    2) Use external database
  Select [1]:

  # If 1 → auto-generate passwords, zero config
  # If 2 → prompt for connection strings
```

### Step 3.5: Kubernetes

```
[3.5/5] Kubernetes Cluster
  K8s mode:
    1) Built-in k3s (recommended, auto-install on this machine)
    2) Use existing K8s cluster
  Select [1]:

  # If 2:
  kubeconfig file path [~/.kube/config]:
  K8s Context name (leave empty for current):
  Ingress backend domain:
    (*.company.com subdomains need to resolve to this address)
    e.g. k8s-ingress.company.com
  Storage Class [standard]:
  ✓ Cluster connection verified: 3 nodes, v1.28.2
```

### Step 4: Advanced Configuration

```
[4/5] Advanced Configuration
  Enter advanced configuration? [y/N]:

  # If y, show category menu:
  Select modules to configure (comma-separated, Enter to skip all):
    1) Admin account       - Initial admin email and password
    2) Email (SMTP)        - Email sending configuration
    3) Payment integration - Stripe payment keys
    4) Enterprise SSO      - Microsoft Entra ID
    5) AI services         - CRS2 / PayGO credential config
    6) Notifications       - Telegram Bot
    7) Performance tuning  - API replicas, connection pool, timeouts
    8) Configure all
  Select [Enter to skip]: 1,2

  ── Admin Account ──
  Admin email [admin@company.com]:
  Admin password (leave empty to auto-generate):
  ✓ Configured

  ── Email Service ──
  SMTP server [smtp.gmail.com]:
  SMTP port [587]:
  SMTP username:
  SMTP password:
  Sender address [noreply@company.com]:
  ✓ Configured

  Skipped: Payment, SSO, AI services, Notifications, Performance
  (Reconfigure later with ./deploy/setup.sh --reconfigure)
```

Each advanced module has its own linear Q&A with defaults. Unconfigured modules use safe defaults (features disabled but won't break the system).

### Step 5: Confirm and Deploy

```
[5/5] Deployment Summary
  ┌──────────────────────────────┐
  │ Domain:    company.com       │
  │ Dashboard: launchpad.corp... │
  │ SSL:       Let's Encrypt     │
  │ Database:  Built-in          │
  │ K8s:       Built-in k3s     │
  └──────────────────────────────┘
  Start deployment? [Y/n]:
```

## Directory Structure

```
launchpad-setup/
├── launchpad/                  # Existing: base config reference (untouched)
├── gateway/                    # Existing: base config reference (untouched)
├── helm/                       # Existing: K8s Helm config (untouched)
├── register.sh                 # Existing: cluster registration reference (untouched)
├── README.md
│
└── deploy/                     # NEW: one-click deployment
    ├── setup.sh                # Main entry point
    ├── scripts/
    │   ├── lib/
    │   │   ├── common.sh       # Common functions: colors, logging, validation
    │   │   ├── i18n.sh         # Language loading
    │   │   ├── detect.sh       # Environment pre-checks: OS, Docker, ports, disk, memory
    │   │   ├── interact.sh     # Interactive parameter collection (basic + advanced)
    │   │   ├── secrets.sh      # Key/password generation (JWT, ZT RSA, DB passwords)
    │   │   ├── render.sh       # Template rendering: .env + nginx.conf
    │   │   ├── certs.sh        # SSL certificates: Let's Encrypt (acme.sh) / user-provided
    │   │   ├── k3s.sh          # k3s install + disable Traefik / external K8s validation
    │   │   ├── database.sh     # PostgreSQL init (5 schemas + users)
    │   │   └── deploy.sh       # docker compose up + health checks + result output
    │   └── lang/
    │       ├── en.sh           # English language pack
    │       └── zh.sh           # Chinese language pack
    ├── templates/
    │   ├── env.template        # Launchpad .env template
    │   ├── env.gateway.template # Gateway .env template
    │   ├── nginx.conf.template # Merged single-layer Nginx config
    │   └── docker-compose.yml.template  # Full orchestration (Nginx/Gitea/PG/Redis + all services)
    └── generated/              # Runtime generated (gitignored)
        ├── launchpad/.env
        ├── gateway/.env
        ├── nginx/
        │   ├── nginx.conf
        │   └── certs/          # SSL certificates
        ├── docker-compose.yml
        └── .setup.conf         # Persisted user config (reload on re-run)
```

## Template Rendering

### Method

Use `envsubst` — zero dependencies, templates use `${VARIABLE}` placeholders.

```bash
render_template() {
    local template="$1"
    local output="$2"
    envsubst < "$template" > "$output"
}
```

### Template Variables

**Basic (required):**

| Variable | Example | Source |
|----------|---------|-------|
| `DOMAIN` | `company.com` | User input |
| `SUBDOMAIN` | `corp` | User input (default: `corp`) |
| `LAUNCHPAD_DOMAIN` | `launchpad.corp.company.com` | Derived |
| `GITEA_DOMAIN` | `launchpad-gitea.corp.company.com` | Derived |
| `WILDCARD_DOMAIN` | `company.com` | Same as DOMAIN |

**Auto-generated (secrets.sh):**

| Variable | Method |
|----------|--------|
| `JWT_SECRET` | `openssl rand -hex 32` |
| `SESSION_SECRET` | `openssl rand -hex 32` |
| `ENCRYPTION_KEY` | `openssl rand -hex 16` |
| `ZT_PRIVATE_KEY` | `openssl genrsa 2048 \| base64` |
| `ZT_PUBLIC_KEY` | Derived from private key |
| `DB_PASSWORD_MAIN` | `openssl rand -base64 18` |
| `DB_PASSWORD_MONITORING` | `openssl rand -base64 18` |
| `DB_PASSWORD_EVENTS` | `openssl rand -base64 18` |
| `DB_PASSWORD_BILLING` | `openssl rand -base64 18` |
| `DB_PASSWORD_STATS` | `openssl rand -base64 18` |
| `GITEA_DB_PASSWORD` | `openssl rand -base64 18` |
| `REDIS_PASSWORD` | `openssl rand -base64 18` |

**Mode-dependent:**

| Variable | Built-in | External |
|----------|----------|----------|
| `DB_HOST` | `postgres` (Docker service) | User-provided IP |
| `DEFAULT_BACKEND` | `localhost` | User-provided ingress domain |

## K8s/k3s Integration

### Built-in k3s Mode

```
Install k3s ──→ Deploy Launchpad ──→ Wait for API ──→ Download register.sh ──→ Execute
    │                                                       │
    ├─ curl get.k3s.io                                      └─ curl -fsSL
    ├─ --disable traefik                                       https://{LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh
    ├─ --tls-san <local-ip>
    └─ export kubeconfig
```

### External K8s Mode

```
Deploy Launchpad ──→ Wait for API ──→ Download register.sh ──→ Execute with --skip-k3s-check
```

### register.sh Download

The register script is always downloaded from the deployed Launchpad instance to ensure version consistency with the running API image:

```bash
register_cluster() {
    wait_for_api

    curl -fsSL "https://${LAUNCHPAD_DOMAIN}/admin/adminapi/k3s/register.sh" \
        -o /tmp/register.sh
    chmod +x /tmp/register.sh

    if [ "$K8S_MODE" = "builtin" ]; then
        LAUNCHPAD_REGISTRATION_URL="http://localhost:6802/admin/adminapi/k3s/register" \
        CLUSTER_NAME="local-k3s" \
            /tmp/register.sh
    else
        LAUNCHPAD_REGISTRATION_URL="http://localhost:6802/admin/adminapi/k3s/register" \
        KUBECONFIG="$K8S_KUBECONFIG_PATH" \
        CLUSTER_NAME="$K8S_CLUSTER_NAME" \
            /tmp/register.sh --skip-k3s-check
    fi
}
```

## Nginx Configuration

Single-layer Nginx replacing the previous two-layer setup (external VM + internal Docker). The template merges both layers.

### Server Blocks

**Block 1: `launchpad.{sub}.{domain}` (Dashboard + API + Gateway)**

- TLS termination with configured certificates
- `/api/v1/terminal/*` → WebSocket proxy to API (3600s timeout)
- `/api/v1/headless-oauth/*/output` → SSE proxy, buffering off (660s timeout)
- `/api/v1/auth/oauth-signup/*/output` → SSE proxy, buffering off (660s timeout)
- `/api/socket.io/` → WebSocket proxy to API (86400s timeout)
- `/gatewayproxy/` → Proxy to Gateway `:6555` (300s timeout)
- `/admin/` → Proxy to Admin `:6804` (rate limited)
- `/api/` → Proxy to API `:6802` (rate limited, 100MB upload)
- `/` → Proxy to UI `:6801`

**Block 2: `launchpad-gitea.{sub}.{domain}` (Gitea)**

- TLS termination
- `/` → Proxy to Gitea `:3000`

**Block 3: `*.{domain}` (Project subdomains)**

- TLS termination with wildcard certificate
- `/` → Proxy to Router `:6580` (WebSocket support, 86400s timeout)

**Block 4: HTTP redirect**

- Port 80, all server names → 301 to HTTPS

## Docker Compose Services

```yaml
services:
  # ---- Infrastructure (conditional) ----
  postgres:          # Built-in mode only, port 5432
  redis:             # Built-in mode only, port 6379

  # ---- Core services ----
  api:               # Launchpad API (:6802)
  ui:                # Launchpad UI (:6801)
  router:            # Project routing (:6580)
  cron:              # Scheduled jobs
  backup-worker:     # Backup queue worker

  # ---- Additional services ----
  gateway:           # ani-code Gateway (:6555)
  gitea:             # Gitea (:3000, SSH :2222)

  # ---- Entry point ----
  nginx:             # TLS + reverse proxy (:80, :443)
```

## Deployment Execution Order

```bash
deploy_services() {
    # Phase 1: Infrastructure
    docker compose up -d postgres redis
    wait_for_healthy postgres 30
    wait_for_healthy redis 15

    # Phase 2: Database initialization (first deploy, built-in mode)
    init_postgres_schemas    # 5 schemas + users + permissions
    init_gitea_database      # gitea database

    # Phase 3: Application services
    docker compose up -d api ui router cron backup-worker gateway gitea
    wait_for_healthy api 60
    wait_for_healthy ui 30
    wait_for_healthy router 30
    wait_for_healthy gitea 45

    # Phase 4: Nginx entry point (last, after all backends ready)
    docker compose up -d nginx
    wait_for_healthy nginx 15

    # Phase 5: Cluster registration
    register_cluster
}
```

### Health Check Mechanism

Each service has a Docker healthcheck defined. The script polls `docker inspect` for health status with a per-service timeout. On failure, the last 20 lines of logs are displayed.

## Final Output

```
═══════════════════════════════════════════════════
  ✓ AniLaunchpad deployment complete!
═══════════════════════════════════════════════════

  Access URLs:
    Dashboard:  https://launchpad.corp.company.com
    Gitea:      https://launchpad-gitea.corp.company.com
    Admin:      https://launchpad.corp.company.com/admin

  Admin credentials:
    Email:      admin@company.com
    Password:   <auto-generated>

  K8s cluster:
    Mode:       Built-in k3s
    Status:     ✓ Registered

  DNS reminder:
    Ensure these DNS records point to this server (x.x.x.x):
    ├─ launchpad.corp.company.com       → x.x.x.x
    ├─ launchpad-gitea.corp.company.com → x.x.x.x
    └─ *.company.com                     → x.x.x.x

  Config files: deploy/generated/
  View logs:    docker compose -f deploy/generated/docker-compose.yml logs -f
  Reconfigure:  ./deploy/setup.sh --reconfigure

═══════════════════════════════════════════════════
```

## CLI Options

```bash
./deploy/setup.sh                  # Fresh install (interactive)
./deploy/setup.sh --reconfigure    # Re-enter configuration (loads previous .setup.conf)
./deploy/setup.sh --status         # Show service status
./deploy/setup.sh --upgrade        # Update image versions
./deploy/setup.sh --uninstall      # Uninstall (with confirmation)
```

## Bilingual Support

Language packs in `scripts/lang/{en,zh}.sh`. All user-facing strings reference variables:

```bash
# lang/en.sh
MSG_WELCOME="AniLaunchpad Setup Script"
MSG_DOMAIN_PROMPT="Enter your main domain (e.g. company.com)"

# lang/zh.sh
MSG_WELCOME="AniLaunchpad 一键部署脚本"
MSG_DOMAIN_PROMPT="请输入您的主域名 (例: company.com)"
```

Selected at startup, loaded via `source "lang/${LANG_CHOICE}.sh"`.

## Configuration Persistence

User inputs are saved to `generated/.setup.conf` after successful deployment. On re-run (`--reconfigure`), the script loads previous values as defaults, allowing users to modify only what changed.

## Idempotency

Re-running the script detects existing state:
- Running containers → prompt to upgrade or skip
- Existing certificates → skip renewal unless expiring
- Existing database schemas → skip initialization
- Existing k3s → skip installation
