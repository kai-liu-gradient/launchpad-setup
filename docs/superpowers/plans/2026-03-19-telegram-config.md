# Telegram Configuration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `TELEGRAM_BOT_USERNAME` to the launchpad env and provide `--setup-telegram` CLI command for standalone Telegram configuration on deployed environments.

**Architecture:** Adds one new env var to the launchpad template, updates the wizard to ask for it, and adds a new CLI action that re-renders affected templates and restarts api + gateway.

**Tech Stack:** Bash (pure shell, no new dependencies)

**Spec:** `docs/superpowers/specs/2026-03-19-telegram-config-design.md`

---

## Chunk 1: Implementation

### Task 1: Add TELEGRAM_BOT_USERNAME to env.template

**Files:**
- Modify: `deploy/templates/env.template:66-67`

- [ ] **Step 1: Add Telegram section after Gateway block**

In `deploy/templates/env.template`, after line 66 (`ANI_CODE_RELEASE_URL=...`) and before the blank line + `# Email`, add:

```
# Telegram Bot
TELEGRAM_BOT_USERNAME=${TELEGRAM_BOT_USERNAME}
```

- [ ] **Step 2: Verify template syntax**

Run: `grep -n 'TELEGRAM' deploy/templates/env.template`
Expected: Shows the new `TELEGRAM_BOT_USERNAME` line.

---

### Task 2: Update interact.sh wizard and save_config

**Files:**
- Modify: `deploy/scripts/lib/interact.sh:264-266` (Module 6 input)
- Modify: `deploy/scripts/lib/interact.sh:435` (save_config)

- [ ] **Step 1: Add Username input before Token in Module 6**

In `deploy/scripts/lib/interact.sh`, replace line 265:

```bash
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
```

with:

```bash
        TELEGRAM_BOT_USERNAME=$(ask_default "Telegram Bot Username" "${TELEGRAM_BOT_USERNAME:-}")
        TELEGRAM_BOT_TOKEN=$(ask_default "Telegram Bot Token" "${TELEGRAM_BOT_TOKEN:-}")
```

- [ ] **Step 2: Add TELEGRAM_BOT_USERNAME to save_config**

In `deploy/scripts/lib/interact.sh`, after line 435 (`TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"`), add:

```bash
TELEGRAM_BOT_USERNAME="${TELEGRAM_BOT_USERNAME:-}"
```

- [ ] **Step 3: Verify changes**

Run: `grep -n 'TELEGRAM' deploy/scripts/lib/interact.sh`
Expected: Shows both `TELEGRAM_BOT_USERNAME` and `TELEGRAM_BOT_TOKEN` in Module 6 and in save_config.

---

### Task 3: Export variable in render.sh

**Files:**
- Modify: `deploy/scripts/lib/render.sh:26-27`

- [ ] **Step 1: Add export after GATEWAY_PUBLIC_URL**

In `deploy/scripts/lib/render.sh`, after line 26 (`export GATEWAY_PUBLIC_URL=...`), add:

```bash
    export TELEGRAM_BOT_USERNAME="${TELEGRAM_BOT_USERNAME:-}"
```

- [ ] **Step 2: Verify**

Run: `grep -n 'TELEGRAM' deploy/scripts/lib/render.sh`
Expected: Shows the new export line.

---

### Task 4: Add --setup-telegram CLI command to setup.sh

**Files:**
- Modify: `deploy/setup.sh:29` (CLI parsing)
- Modify: `deploy/setup.sh:170-175` (new action handler)
- Modify: `deploy/setup.sh:198` (help text)

- [ ] **Step 1: Add CLI argument parsing**

In `deploy/setup.sh`, after line 29 (`--setup-db)`), add:

```bash
        --setup-telegram)   ACTION="setup-telegram";   shift ;;
```

- [ ] **Step 2: Add action handler**

In `deploy/setup.sh`, after the `setup-db)` block (after line 175), add:

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

- [ ] **Step 3: Add help text**

In the help section, after line 198 (`--setup-db`), add:

```
        echo "  --setup-telegram    Configure Telegram bot settings"
```

- [ ] **Step 4: Verify CLI parsing**

Run: `grep -n 'telegram\|TELEGRAM' deploy/setup.sh`
Expected: Shows the new CLI arg, action handler, and help text.

---

### Task 5: Final verification and commit

- [ ] **Step 1: Verify all TELEGRAM references are consistent**

Run: `grep -rn 'TELEGRAM' deploy/`
Expected: `TELEGRAM_BOT_TOKEN` in env.gateway.template, interact.sh, setup.sh. `TELEGRAM_BOT_USERNAME` in env.template, interact.sh, render.sh, setup.sh. `TELEGRAM_MODE` in env.gateway.template.

- [ ] **Step 2: Syntax check setup.sh**

Run: `bash -n deploy/setup.sh && echo "OK"`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add deploy/templates/env.template deploy/scripts/lib/interact.sh deploy/scripts/lib/render.sh deploy/setup.sh
git commit -m "feat: add TELEGRAM_BOT_USERNAME config and --setup-telegram CLI command"
```
