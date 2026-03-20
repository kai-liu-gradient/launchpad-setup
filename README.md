# AniLaunchpad Setup

[中文文档](README.zh.md)

One-click deployment toolkit for the **AniLaunchpad** full-stack platform — an interactive wizard that provisions infrastructure, configures services, and brings everything online in a single run.

## Features

- **Interactive Wizard** — 6-page terminal UI with arrow-key navigation and back/forward flow
- **Non-Interactive Mode** — supply a config file for fully automated deployments (`--config FILE`)
- **Bilingual** — English and Chinese (select at startup)
- **Flexible Infrastructure** — built-in K3s _or_ external Kubernetes cluster
- **3 SSL Modes** — Let's Encrypt (wildcard via DNS), self-signed CA, or bring-your-own certificates
- **Built-in _or_ External Database** — auto-provisions PostgreSQL + Redis, or connect to existing instances
- **Unified Progress Display** — live progress bar with per-step spinners (TTY) or simple progress (non-TTY)
- **Resume & Retry** — `--resume` picks up where a failed deploy left off
- **Template Import** — automatically pushes Git repos and imports YAML templates to Gitea + API
- **Clean Uninstall** — `--uninstall-all` removes services, K3s, cron, config, and DNS entries

## Prerequisites

| Dependency | Minimum Version | Notes |
|---|---|---|
| **Linux** | Any modern distro | macOS/Windows not supported |
| **Docker** | 20.10+ | Must be running (`docker info`) |
| **Docker Compose** | v2 | Plugin format (`docker compose`) |
| **jq** | 1.6+ | Used by cluster registration |
| **envsubst** | any | Part of `gettext-base` package |
| **openssl** | 1.1+ | Secret/certificate generation |
| **helm** | 3.x | Only required for `K8S_MODE=builtin` |
| **curl** | any | Let's Encrypt + cluster registration |

## Quick Start

```bash
# Interactive mode (recommended for first install)
sudo ./deploy/setup.sh

# Non-interactive mode with config file
sudo ./deploy/setup.sh --config deploy/presets/test-server.conf
```

The wizard walks through: domain setup, SSL mode, database mode, Kubernetes mode, and advanced options (SMTP, SSO, billing, AI, storage, performance tuning).

---

## Architecture

### Request Routing Overview

Two distinct traffic paths flow through the system:

```mermaid
graph TD
    subgraph "External Traffic"
        Browser["Browser"]
    end

    subgraph "Nginx (TLS Termination — ports 80/443)"
        NG["nginx:alpine"]
    end

    subgraph "Docker Compose Services"
        UI["UI :6801"]
        API["API :6802 / :6804 admin"]
        Router["Router :6580"]
        GW["Gateway :6555"]
        Gitea["Gitea :3000 / :2222 SSH"]
        Cron["Cron"]
        Backup["Backup Worker"]
    end

    subgraph "Data Layer"
        PG["PostgreSQL :5432<br/>6 schemas + gitea"]
        Redis["Redis :6379"]
    end

    subgraph "Kubernetes Cluster"
        Ingress["Ingress-Nginx :30080"]
        CoreDNS["CoreDNS"]
        Kyverno["Kyverno<br/>(selfsigned only)"]
        Pods["User Project Pods"]
    end

    Browser -->|"https://launchpad.sub.domain"| NG
    Browser -->|"https://project.domain"| NG

    NG -->|"/dashboard routes"| UI
    NG -->|"/api/* routes"| API
    NG -->|"*.domain (wildcard)"| Router
    NG -->|"/gatewayproxy/*"| GW
    NG -->|"gitea subdomain"| Gitea

    UI --> API
    API --> PG
    API --> Redis
    GW --> PG
    Router -->|"query route rules"| API
    Router -->|"proxy to pod"| Ingress
    Ingress --> Pods
    CoreDNS -.->|"resolve *.domain"| Ingress
    Kyverno -.->|"inject CA cert"| Pods
```

**Path A — Dashboard Access** (`https://launchpad.sub.domain`):
Browser → Nginx → UI (frontend) → API (backend) → manages projects, deployments, users

**Path B — User Service Access** (`https://myproject.domain`):
Browser → Nginx → Router → queries API for routing rules → proxies to K8s Ingress-Nginx (:30080) → user's Pod

### Deployment Modes

The system supports two fundamentally different deployment topologies:

#### Mode 1: Self-Signed + Built-in K3s (Single Machine)

Best for: development, testing, internal/air-gapped environments.

```mermaid
graph LR
    subgraph "Single Server"
        subgraph "Docker Compose"
            Nginx2["Nginx<br/>self-signed wildcard cert"]
            Services2["API / UI / Router<br/>Gateway / Gitea"]
            DB2["PostgreSQL + Redis"]
        end
        subgraph "K3s (built-in)"
            Ingress2["Ingress-Nginx<br/>NodePort 30080"]
            CoreDNS2["CoreDNS<br/>custom *.domain → host IP"]
            Kyverno2["Kyverno<br/>CA cert injection"]
            dnsmasq2["dnsmasq<br/>Docker DNS → host IP"]
        end
    end
```

Key characteristics:
- K3s installed automatically (Traefik disabled, replaced by Ingress-Nginx)
- Self-signed CA generated → wildcard cert for `*.domain`
- **CoreDNS custom config**: pods resolve `*.domain` → Ingress ClusterIP
- **dnsmasq**: Docker containers resolve `*.domain` → host IP (for Gateway → Nginx → pods chain)
- **Kyverno ClusterPolicy**: injects CA cert into all pods via initContainer (merges with system CA bundle)
- `NODE_EXTRA_CA_CERTS` set on Gateway container for Node.js TLS trust
- `NODE_TLS_REJECT_UNAUTHORIZED=0` set on API + Gateway

#### Mode 2: Public Certificate + External Cluster

Best for: production, multi-node, public-facing deployments.

```mermaid
graph LR
    subgraph "Deploy Server"
        subgraph "Docker Compose"
            Nginx3["Nginx<br/>Let's Encrypt / custom cert"]
            Services3["API / UI / Router<br/>Gateway / Gitea"]
            DB3["PostgreSQL + Redis<br/>(or external)"]
        end
    end
    subgraph "External K8s Cluster"
        ExtIngress["Existing Ingress Controller"]
        ExtPods["User Project Pods"]
    end
    Services3 -->|"kubeconfig"| ExtIngress
    ExtIngress --> ExtPods
```

Key characteristics:
- Let's Encrypt wildcard cert via `acme.sh` + DNS API (Cloudflare, Aliyun, Azure)
- Or bring your own certificates (`SSL_MODE=custom`)
- External K8s cluster connected via kubeconfig file
- No Kyverno, no CoreDNS custom config, no dnsmasq needed
- Cluster has its own Ingress Controller; Router proxies to its domain
- Database can be external too (`DB_MODE=external`)

### Project Structure

```
launchpad-setup/
├── deploy/
│   ├── setup.sh                 # Entry point — CLI argument parsing + orchestration
│   ├── versions.conf            # Image versions (edit to upgrade)
│   ├── presets/                  # Pre-built config files for non-interactive deploy
│   ├── scripts/
│   │   ├── lang/
│   │   │   ├── en.sh            # English messages
│   │   │   └── zh.sh            # Chinese messages
│   │   └── lib/
│   │       ├── common.sh        # UI: logging, progress bars, prompts, validation
│   │       ├── i18n.sh          # Language selection
│   │       ├── detect.sh        # Environment checks + IP detection
│   │       ├── interact.sh      # 6-page wizard + save_config()
│   │       ├── secrets.sh       # Cryptographic secret generation
│   │       ├── render.sh        # Template rendering (envsubst + sed)
│   │       ├── certs.sh         # SSL: Let's Encrypt / self-signed / custom
│   │       ├── k3s.sh           # K3s install + cluster registration
│   │       ├── k8s-components.sh# Ingress-Nginx, CoreDNS, Kyverno
│   │       ├── database.sh      # PostgreSQL schema + user creation
│   │       └── deploy.sh        # Deployment orchestration + lifecycle
│   ├── templates/               # Config templates (envsubst placeholders)
│   │   ├── env.template         # → generated/launchpad/.env
│   │   ├── env.gateway.template # → generated/gateway/.env
│   │   ├── nginx.conf.template  # → generated/nginx/nginx.conf
│   │   ├── settings.yml.template# → generated/launchpad/config/settings.yml
│   │   ├── coredns-custom.yaml.template
│   │   ├── kyverno-inject-ca.yaml
│   │   ├── kyverno-sync-ca.yaml
│   │   └── values-builtin.yml   # Helm values for ingress-nginx
│   └── generated/               # Runtime output (gitignored)
│       ├── .setup.conf          # Saved configuration
│       ├── .secrets             # Generated secrets (chmod 600)
│       ├── docker-compose.yml   # Rendered compose file
│       ├── launchpad/.env       # App environment
│       ├── gateway/.env         # Gateway environment
│       └── nginx/               # Nginx config + SSL certs
├── files/                       # Template repos + YAML definitions for import
│   ├── *.tar.gz                 # Git repo archives → pushed to Gitea
│   └── *.yaml                   # Template definitions → imported via admin API
├── tests/
│   ├── e2e.sh                   # E2E test runner
│   ├── cases/                   # 10-step test scenarios
│   └── lib/                     # Test utilities
└── docs/                        # Technical documentation
```

### Module Dependency Map

| Module | Responsibility | Depends On |
|---|---|---|
| `common.sh` | Logging, progress bars, styled UI, validation | (none — loaded first) |
| `i18n.sh` | Language selection prompt | `common.sh` |
| `detect.sh` | OS/Docker/tool checks, IP detection | `common.sh` |
| `interact.sh` | 6-page wizard, `save_config()` | `common.sh`, `detect.sh` |
| `secrets.sh` | Generate/save/load cryptographic secrets | `common.sh` |
| `render.sh` | Template → generated config rendering | `common.sh`, `detect.sh`, `secrets.sh` |
| `certs.sh` | SSL certificate management (3 modes) | `common.sh`, `detect.sh` |
| `k3s.sh` | K3s install, external K8s validation, cluster registration | `common.sh`, `detect.sh` |
| `k8s-components.sh` | Ingress-Nginx, CoreDNS, Kyverno, CA distribution | `common.sh`, `detect.sh` |
| `database.sh` | PostgreSQL schema/user creation | `common.sh`, `secrets.sh` |
| `deploy.sh` | Orchestration, health checks, lifecycle (start/stop/upgrade) | All of the above |

---

## Deployment Flow

```mermaid
flowchart TD
    Start(["./setup.sh"]) --> ParseArgs["Parse CLI arguments"]
    ParseArgs --> Mode{"--config file?"}
    Mode -->|Yes| LoadConfig["Load config file<br/>(non-interactive)"]
    Mode -->|No| Wizard["Interactive Wizard<br/>6 pages"]

    LoadConfig --> CheckEnv["check_environment()<br/>Docker, Compose, jq, envsubst, helm"]
    Wizard --> CheckEnv

    CheckEnv --> Secrets["generate_secrets()<br/>JWT, encryption keys, RSA,<br/>DB passwords, Redis password"]
    Secrets --> Render["render_templates()<br/>.env, docker-compose.yml,<br/>nginx.conf, settings.yml"]
    Render --> SaveConfig["save_config() + save_secrets()"]

    SaveConfig --> Deploy["deploy_services()<br/>Unified progress display"]

    Deploy --> K3S{"K3s needed?<br/>(builtin && !running)"}
    K3S -->|Yes| InstK3S["install_k3s()"]
    K3S -->|No| SSL
    InstK3S --> SSL["setup_certificates()"]

    SSL --> Builtin{"K8S_MODE=builtin?"}
    Builtin -->|Yes| Ingress["install_ingress_nginx()"]
    Builtin -->|No| Infra
    Ingress --> CoreDNS["configure_coredns()"]
    CoreDNS --> SelfSigned{"SSL_MODE=selfsigned?"}
    SelfSigned -->|Yes| KyvernoStep["install_kyverno()<br/>setup_cert_distribution()"]
    SelfSigned -->|No| Infra
    KyvernoStep --> Infra

    Infra["Start PostgreSQL + Redis"] --> DBInit["init_database()<br/>6 schemas + Prisma push"]
    DBInit --> GiteaStart["Start Gitea<br/>bootstrap_gitea()"]
    GiteaStart --> AppServices["Start API, UI, Router,<br/>Gateway, Cron, Backup"]
    AppServices --> NginxStart["Start Nginx<br/>setup_hosts()"]
    NginxStart --> Templates["import_templates()<br/>Git repos + YAML"]
    Templates --> Register["register_cluster()"]
    Register --> Done(["show_result()"])
```

### Phase Details

#### Phase 1: Environment Detection

`check_environment()` verifies all prerequisites are installed and running. For `K8S_MODE=builtin`, it also installs `dnsmasq` if missing.

#### Phase 2: Configuration

**Interactive mode** presents a 6-page wizard:

| Page | Collects | Key Variables |
|---|---|---|
| 1 - Basic | Domain, subdomain, image registry, versions | `DOMAIN`, `SUBDOMAIN`, `IMAGE_REGISTRY`, `IMAGE_VERSION_*` |
| 2 - SSL | Certificate mode, DNS provider, API token | `SSL_MODE`, `DNS_PROVIDER`, `DNS_API_TOKEN` |
| 3 - Database | Built-in or external, connection URLs | `DB_MODE`, `DATABASE_URL`, `REDIS_HOST` |
| 4 - Kubernetes | Built-in K3s or external cluster | `K8S_MODE`, `K8S_KUBECONFIG_PATH`, `STORAGE_CLASS` |
| 5 - Advanced | Admin, SMTP, Stripe, SSO, AI, notifications, storage, performance | `ADMIN_EMAIL`, `SMTP_*`, `API_REPLICAS`, etc. |
| 6 - Summary | Review all settings, confirm deploy | (read-only) |

**Non-interactive mode** reads all variables from a config file (see `deploy/presets/test-server.conf` for example).

#### Phase 3: Secret Generation & Template Rendering

**Secrets generated** (stored in `generated/.secrets`, chmod 600):
- 3 JWT/session secrets (hex-64)
- 3 encryption keys (hex-32)
- 1 RSA keypair (base64-encoded, for Zero Trust)
- 8 database passwords (base64-24)
- 1 Redis password
- 2 gateway secrets
- 1 admin password (if not user-provided)

**Templates rendered** (via `envsubst` + `sed` placeholder replacement):
- `docker-compose.yml` — conditionally includes PostgreSQL/Redis, sets image versions, injects extra_hosts for builtin mode
- `.env` files — all environment variables for API, Gateway
- `nginx.conf` — reverse proxy rules, TLS cert paths
- `settings.yml` — application settings

#### Phase 4: Unified Deployment

`deploy_services()` builds a dynamic step list at runtime based on the deployment mode. Steps are conditionally included:

| Step | Condition | Timeout | What It Does |
|---|---|---|---|
| K3s Install | builtin && not running | 60s | `curl get.k3s.io`, wait for node ready |
| SSL Certificates | always | — | Generate/fetch certs per `SSL_MODE` |
| Ingress-Nginx | builtin | 120s | Helm install, wait for controller pod |
| CoreDNS Config | builtin | — | Custom DNS + dnsmasq setup |
| Kyverno | builtin && selfsigned | 120s | Helm install, wait for admission controller |
| CA Distribution | builtin && selfsigned | — | ConfigMap + ClusterPolicy for CA injection |
| Infrastructure | builtin DB | 30s | Start PostgreSQL + Redis, health check |
| Database Init | always | — | Create 6 schemas + Gitea DB, Prisma push (6 schemas) |
| Gitea | always | 90s | Start, health check, create admin + token + org |
| API | always | 90s | Start, health check |
| UI | always | 60s | Start, health check |
| Router | always | 60s | Start, health check |
| Nginx | always | 30s | Start, update `/etc/hosts`, wait 5s for HTTPS |
| Template Import | always | — | Push Git repos to Gitea, import YAML via admin API |
| Cluster Registration | always | — | Download + run `register.sh` |

#### Phase 5: Completion

`show_result()` displays:
- Access URLs (dashboard, Gitea, admin)
- Admin credentials
- K8s cluster status
- DNS configuration reminder (which domains to point to server IP)

---

## CLI Reference

### Install & Configure

| Command | Description |
|---|---|
| `./deploy/setup.sh` | Fresh install with interactive wizard |
| `./deploy/setup.sh --config FILE` | Non-interactive install using config file |
| `./deploy/setup.sh --reconfigure` | Re-enter configuration wizard (preserves existing config as defaults) |

### Manage

| Command | Description |
|---|---|
| `./deploy/setup.sh --status` | Show Docker Compose service status |
| `./deploy/setup.sh --restart <service>` | Restart a service (`api\|ui\|router\|gateway\|nginx\|gitea\|all`) |
| `./deploy/setup.sh --upgrade` | Pull latest images (per `versions.conf`) and restart |

### Granular Operations

| Command | Description |
|---|---|
| `./deploy/setup.sh --resume` | Resume full deploy from existing config (after failure) |
| `./deploy/setup.sh --setup-k3s` | Install/reconfigure K3s only |
| `./deploy/setup.sh --setup-certs` | Regenerate SSL certificates only |
| `./deploy/setup.sh --setup-db` | Run database initialization only |
| `./deploy/setup.sh --import-templates` | Re-import templates to Gitea + API |

### Teardown

| Command | Description |
|---|---|
| `./deploy/setup.sh --uninstall` | Stop services, remove containers and **all data** |
| `./deploy/setup.sh --uninstall-all` | Complete removal: services + K3s + cron + config + DNS |

---

## Important Notes & Known Pitfalls

### Shell Scripting

| Pitfall | Impact | Solution |
|---|---|---|
| `local` keyword in subshells | `local` only works inside functions. Using it in `( ... )` subshells fails silently with `set -e`, exit code 128 | Never use `local` in `( ... )` subshells |
| EXIT trap inherited by subshells | The `/dev/tty` cursor-restore trap fires in subshells. Via SSH (no tty), the redirect error becomes the exit code | Add `trap - EXIT` at the start of subshell blocks |
| `grep` + `pipefail` | `grep pattern \| tail \| cut` — if grep matches nothing, exit 1 propagates and kills the script | Append `\|\| true` to grep pipelines that might match nothing |
| `envsubst` only sees exported variables | `source .setup.conf` sets shell vars but doesn't export them. `envsubst` is an external process | Explicitly `export` every variable referenced in templates |

### Network & DNS

| Pitfall | Impact | Solution |
|---|---|---|
| Docker containers can't resolve `*.domain` | Gateway needs to reach Nginx:443 for wildcard TLS, but Docker DNS doesn't know about custom domains | Install dnsmasq with `address=/DOMAIN/HOST_IP` |
| CoreDNS loop detection | Default CoreDNS config uses `/etc/resolv.conf` which may point to CoreDNS itself, causing a loop | Patch CoreDNS: comment out `loop` plugin, use public DNS forwarders |
| `KUBECONFIG` not set when K3s step is skipped | If K3s is already running, `install_k3s()` is skipped, leaving KUBECONFIG unset. Helm/kubectl default to `localhost:8080` | Export KUBECONFIG at the top of `deploy_services()` before any conditional logic |

### SSL / TLS

| Pitfall | Impact | Solution |
|---|---|---|
| Self-signed CA not trusted by curl/wget in K8s pods | Kyverno injects `NODE_EXTRA_CA_CERTS` (Node.js only). System tools (curl, wget, git) use `/etc/ssl/certs/ca-certificates.crt` | Use initContainer to merge CA into system bundle, mount as volume |
| `DEFAULT_BACKEND` must not contain protocol prefix | Router code adds `http://` internally. If value already has `http://`, result is `http://http://host:port` | Strip `http://`/`https://` prefix in `render.sh` |

### API & Services

| Pitfall | Impact | Solution |
|---|---|---|
| Template import uses admin port (6804), not API port (6802) | Port 6802 requires JWT auth (no user registered yet). Port 6804 is unauthenticated admin API | Use `localhost:6804/api/templates/yaml/create` |
| API container has `wget`, not `curl` | The API Docker image doesn't include curl | Use `docker compose exec -T api wget -q -O-` for API container HTTP calls |
| Health endpoint is `/health`, not `/api/health` | `/api/` prefix is only for API routes | Check `http://localhost:6802/health` |
| Empty wget response treated as success | `wget` errors go to stderr; with `2>/dev/null \|\| echo ""`, the variable is empty, and `grep "error"` returns false | Use `2>&1 \|\| echo "WGET_FAILED"` and check for `"success":true` explicitly |
| Git commit needs committer identity | `git commit --author=` sets author, but committer identity is also required. Remote servers have no global git config | Set `GIT_AUTHOR_NAME/EMAIL` and `GIT_COMMITTER_NAME/EMAIL` env vars |

### Configuration

| Pitfall | Impact | Solution |
|---|---|---|
| Image versions differ per service | Router (1.16.2) and Gateway (1.18.7) use different versions from API/UI (2.0.3) | Use per-service version variables: `IMAGE_VERSION_ROUTER`, `IMAGE_VERSION_GATEWAY`, etc. |
| `GITEA_ACCESS_TOKEN` is in `.env`, not `.secrets` | Token is generated at runtime by `bootstrap_gitea()`. The `--import-templates` handler must read it from `.env` | Read from `generated/launchpad/.env`, not `.secrets` |
| `kubectl wait` fails if no resources exist yet | It errors immediately with "no matching resources found" instead of waiting | Add a polling loop to wait for the resource to exist first |
| Duplicate YAML keys from multi-phase sed injection | If multiple render phases each insert `environment:` under the same service, YAML rejects the duplicate key | Append into existing blocks using anchor lines, never create new mapping keys |

---

## Troubleshooting

### Viewing Logs

```bash
# All services
docker compose -f deploy/generated/docker-compose.yml logs -f

# Specific service
docker compose -f deploy/generated/docker-compose.yml logs -f api

# K3s / kubectl
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get pods -A
kubectl logs -n <namespace> <pod-name>
```

### Common Issues

| Symptom | Likely Cause | Fix |
|---|---|---|
| Script dies silently (exit 128) | `local` in subshell or EXIT trap in SSH | Check for subshell usage patterns, add `trap - EXIT` |
| "connection refused" on port 6802 | API not healthy yet | `./setup.sh --resume` or check `docker compose logs api` |
| Gitea HTTPS not reachable | DNS not configured, or Nginx not ready | Wait for Nginx to start; check `/etc/hosts` or DNS records |
| Git push fails during template import | SSL cert not trusted / Gitea not ready | For selfsigned: `GIT_SSL_NO_VERIFY=1`; check Gitea health |
| Pods can't reach `*.domain` | CoreDNS or dnsmasq not configured | Check `kubectl get cm coredns-custom -n kube-system` and `/etc/dnsmasq.d/launchpad.conf` |
| `helm: command not found` | Helm not installed | Install Helm 3: `curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \| bash` |
| Cluster registration fails | API not reachable via public URL, or DNS not set | Check `curl -k https://LAUNCHPAD_DOMAIN/health`; retry with `--resume` |
| Database "role already exists" | Re-running init on existing DB | Safe to ignore — `IF NOT EXISTS` guards handle this |

### Recovery Commands

```bash
# Resume after failure (picks up where it left off)
./deploy/setup.sh --resume

# Re-import templates only
./deploy/setup.sh --import-templates

# Restart a misbehaving service
./deploy/setup.sh --restart api

# Full reinstall (WARNING: deletes all data)
./deploy/setup.sh --uninstall-all
./deploy/setup.sh
```

---

## Configuration File Reference

For non-interactive deployments, create a config file (see `deploy/presets/test-server.conf`):

```bash
# Required
DOMAIN="example.com"
SUBDOMAIN="corp"
SSL_MODE="selfsigned"           # letsencrypt | selfsigned | custom
DB_MODE="builtin"               # builtin | external
K8S_MODE="builtin"              # builtin | external
ADMIN_EMAIL="admin@example.com"

# Image versions
IMAGE_REGISTRY="swr.ap-southeast-1.myhuaweicloud.com/ghisha"
IMAGE_VERSION_API="2.0.3"
IMAGE_VERSION_UI="2.0.3"
IMAGE_VERSION_ROUTER="1.16.2"
IMAGE_VERSION_GATEWAY="1.18.7"
IMAGE_VERSION_GITEA="1.25-rootless"

# Optional: K8s
K8S_KUBECONFIG_PATH="/etc/rancher/k3s/k3s.yaml"
STORAGE_CLASS="local-path"

# Optional: Let's Encrypt
DNS_PROVIDER="cloudflare"       # cloudflare | aliyun | azure | manual
DNS_API_TOKEN="your-token"

# Optional: External database
DATABASE_URL="postgresql://user:pass@host:5432/db?schema=launchpad_main"
REDIS_HOST="redis.example.com"
REDIS_PORT="6379"
REDIS_PASSWORD="your-password"

# Optional: Advanced
ADMIN_PASSWORD="custom-password"
SMTP_HOST="smtp.gmail.com"
API_REPLICAS="2"
ROUTER_REPLICAS="2"
GATEWAY_REPLICAS="2"
```

---

## License

Copyright (c) Gradient8. All rights reserved.
