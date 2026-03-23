# README Rewrite: Shell → Go CLI

**Date:** 2026-03-23
**Status:** Approved

## Summary

Rewrite README.md to reflect the Go-based `launchpad` CLI, replacing all references to the old bash `./deploy/setup.sh` approach. Delete README.zh.md (English only going forward).

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | English only | User preference; delete README.zh.md |
| Audience | Operators + developers | Deployment guide first, dev info second |
| Architecture diagrams | One simplified mermaid diagram | Old README had 3 detailed diagrams; one request-routing overview is enough |
| Shell pitfalls section | Delete entirely | These were bash-specific; Go binary has none of these issues |
| Module dependency map | Delete entirely | Bash module table; irrelevant for Go |
| Deployment flow diagram | Delete entirely | Detailed phase flowchart was for bash orchestration |
| Config format | YAML example | Go CLI uses `.setup.yaml`, not bash `.conf` files |
| Length target | ~250-300 lines | Concise but complete |

## README Structure

### 1. Header
- Project name: `# AniLaunchpad`
- One-line description
- No language switcher (single language)

### 2. Features
Bullet list based on actual Go CLI capabilities:
- Express install (default) and Custom install (full TUI wizard)
- Non-interactive mode (`--config FILE`)
- Reconfigure running installations (`configure`)
- Runtime settings editor with hot-reload (`settings`)
- 3 SSL modes: Let's Encrypt, self-signed CA, custom certificates
- Built-in or external database (PostgreSQL + Redis)
- Built-in K3s or external Kubernetes cluster
- Template auto-import (embedded in binary)
- Upgrade, restart, uninstall lifecycle commands
- Telegram bot configuration
- Unified TUI progress display

### 3. Prerequisites
Simplified table — Go binary handles most things internally:
- Linux (any modern distro)
- Docker 20.10+
- Docker Compose v2
- helm 3.x (only for builtin K3s mode)

Note: jq, envsubst, openssl are NOT needed — the Go binary handles these internally.

### 4. Quick Start
Three usage patterns:
```
# Express install (recommended for first install)
sudo launchpad install

# Full custom wizard
sudo launchpad install --custom

# Non-interactive with config file
sudo launchpad install --config setup.yaml
```

### 5. CLI Reference
Complete table of all subcommands with flags:

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `launchpad install` | Install AniLaunchpad | `--custom`, `--config FILE`, `--resume`, `--dir` |
| `launchpad configure` | Reconfigure running installation | |
| `launchpad configure export` | Export current config to file | `--output/-o FILE` |
| `launchpad settings` | Edit runtime settings (hot-reload TUI) | |
| `launchpad status` | Show service status | `--dir` |
| `launchpad upgrade` | Upgrade to new version | `--version`, `--dir` |
| `launchpad restart [service]` | Restart services | `--dir` |
| `launchpad uninstall` | Remove services and data | `--all` (includes K3s), `--dir` |
| `launchpad setup-certs` | Regenerate SSL certificates | `--dir` |
| `launchpad setup-db` | Initialize databases | `--dir` |
| `launchpad import-templates` | Re-import templates to Gitea + API | `--dir` |
| `launchpad setup-telegram` | Configure Telegram bot | `--token`, `--username`, `--dir` |

### 6. Configuration Reference
YAML example showing the Config struct fields:
```yaml
domain: example.com
subdomain: corp
admin_email: admin@example.com
images:
  registry: swr.ap-southeast-1.myhuaweicloud.com/ghisha
  default_version: "2.0.3"
ssl:
  mode: selfsigned  # letsencrypt | selfsigned | custom
database:
  mode: builtin     # builtin | external
kubernetes:
  mode: builtin     # builtin | external
  kubeconfig: /etc/rancher/k3s/k3s.yaml  # optional, for external mode
  storage_class: local-path
performance:
  api_replicas: 1
  router_replicas: 1
  db_conn_limit: 100
```

With brief descriptions of optional sections (smtp, sso, storage, telegram, ai, stripe, experimental).

### 7. Architecture Overview
One simplified mermaid diagram showing request routing:
- Browser → Nginx (TLS termination) → services (UI, API, Router, Gateway, Gitea)
- Router → K8s Ingress → user pods
- Data layer: PostgreSQL + Redis

Brief text explaining the two deployment modes:
- Mode 1: Self-signed + built-in K3s (single machine, dev/test)
- Mode 2: Public cert + external K8s cluster (production)

No separate mermaid diagrams for each mode.

### 8. Project Structure
Updated Go project layout:
```
launchpad-setup/
├── cmd/
│   ├── launchpad/          # CLI entry point
│   └── gen-versions/       # Code generator for version constants
├── internal/
│   ├── cli/                # Cobra command definitions
│   ├── config/             # Config struct, validation, I/O
│   ├── engine/             # Deployment orchestration
│   ├── secrets/            # Secret generation and management
│   ├── settings/           # Runtime settings editor
│   ├── template/           # Embedded config templates + rendering
│   ├── importfiles/        # Embedded template repos for Gitea import
│   └── tui/                # Terminal UI components
│       ├── components/     # Shared TUI building blocks
│       ├── wizard/         # Install wizard (express + custom)
│       ├── panel/          # Configure panel
│       ├── settings/       # Settings editor tabs
│       └── progress/       # Deploy progress display
├── scripts/
│   └── deploy.sh           # Build + deploy helper
├── go.mod
├── go.sum
├── Makefile
└── docs/
```

### 9. Development
- Build: `make build`
- Build for Linux: `make build-linux`
- Test: `make test` or `go test ./...`
- Deploy to server: `./scripts/deploy.sh`
- Lint: `make lint`

### 10. Troubleshooting
Simplified table with common issues updated for Go CLI:

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| "connection refused" on port 6802 | API not healthy yet | `launchpad install --resume` |
| Pods can't reach `*.domain` | CoreDNS or dnsmasq not configured | Check CoreDNS configmap and dnsmasq |
| Git push fails during template import | SSL cert not trusted / Gitea not ready | Check Gitea health; self-signed mode uses GIT_SSL_NO_VERIFY |
| Cluster registration fails | API not reachable via public URL | Check DNS records; retry with `--resume` |

### 11. License
`Copyright (c) Gradient8. All rights reserved.`

## Files Changed

| File | Action |
|------|--------|
| `README.md` | Complete rewrite |
| `README.zh.md` | Delete |

## Out of Scope

- Generating new mermaid diagrams beyond the single routing overview
- Writing separate docs/ files for architecture details
- Translating the new README to Chinese
