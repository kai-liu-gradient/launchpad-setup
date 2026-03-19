# Telegram Configuration Design

**Date:** 2026-03-19
**Status:** Approved

## Goal

Add `TELEGRAM_BOT_USERNAME` to the launchpad env template and provide a standalone `--setup-telegram` CLI command so operators can configure Telegram on an already-deployed environment with minimal disruption (only api + gateway restart).

## Current State

- `env.gateway.template` already has `TELEGRAM_BOT_TOKEN`, `GATEWAY_PUBLIC_URL`, `TELEGRAM_MODE=polling`
- `interact.sh` wizard Module 6 (Notifications) asks for `TELEGRAM_BOT_TOKEN`
- `render.sh` exports `GATEWAY_PUBLIC_URL` (derived from domain)
- `env.template` (launchpad/.env) is **missing** `TELEGRAM_BOT_USERNAME`
- No standalone CLI command exists for Telegram configuration

## Changes

### 1. `deploy/templates/env.template`

Add after the Gateway section:

```
# Telegram Bot
TELEGRAM_BOT_USERNAME=${TELEGRAM_BOT_USERNAME}
```

### 2. `deploy/scripts/lib/interact.sh`

**Module 6 (Notifications):** Change input order — ask Username before Token:

```bash
TELEGRAM_BOT_USERNAME=$(ask_default "Telegram Bot Username" "${TELEGRAM_BOT_USERNAME:-}")
TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
```

**`save_config()`:** Add persistence line:

```bash
TELEGRAM_BOT_USERNAME="${TELEGRAM_BOT_USERNAME:-}"
```

### 3. `deploy/scripts/lib/render.sh`

Export the variable for envsubst (after line 26, near `GATEWAY_PUBLIC_URL`):

```bash
export TELEGRAM_BOT_USERNAME="${TELEGRAM_BOT_USERNAME:-}"
```

### 4. `deploy/setup.sh`

**CLI argument parsing:** Add `--setup-telegram`.

**Action handler:**

```bash
setup-telegram)
    source "${DEPLOY_DIR}/scripts/lib/detect.sh"
    source "${DEPLOY_DIR}/scripts/lib/interact.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    source "${DEPLOY_DIR}/scripts/lib/render.sh"
    source "${DEPLOY_DIR}/scripts/lib/deploy.sh"
    load_saved_config
    TELEGRAM_BOT_USERNAME=$(ask_default "Telegram Bot Username" "${TELEGRAM_BOT_USERNAME:-}")
    TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
    render_templates
    save_config
    restart_service "api"
    restart_service "gateway"
    ;;
```

Note: `detect.sh` is required because `render_templates()` calls `detect_internal_ip()` when `K8S_MODE=builtin`. `interact.sh` is required because `save_config()` is defined there. `ask_default()` is in `common.sh` (always loaded). `secrets.sh` is sourced explicitly to match existing `--setup-*` patterns (though `load_saved_config` also sources it).

**Help text:** Add under "Setup Phases":

```
  --setup-telegram    Configure Telegram bot settings
```

### 5. Language files

No changes needed — existing `MSG_ADV_NOTIFY_DESC="Telegram Bot"` is sufficient.

## Execution Flow for `--setup-telegram`

1. `load_saved_config()` — load existing `.setup.conf` + `.secrets`
2. Interactive prompts for Username then Token (shows current values as defaults)
3. `render_templates()` — re-render all .env files with updated values
4. `save_config()` — persist new Telegram values to `.setup.conf`
5. `restart_service "api"` + `restart_service "gateway"` — restart only affected services

## Files Modified

| File | Change |
|------|--------|
| `deploy/templates/env.template` | Add `TELEGRAM_BOT_USERNAME` |
| `deploy/scripts/lib/interact.sh` | Add Username input, reorder, add to save_config |
| `deploy/scripts/lib/render.sh` | Export `TELEGRAM_BOT_USERNAME` |
| `deploy/setup.sh` | Add `--setup-telegram` CLI arg + action + help text |

## Impact

- Existing deployments: no impact (variable defaults to empty)
- Only api + gateway services are restarted when using `--setup-telegram`
- All other services remain unaffected
