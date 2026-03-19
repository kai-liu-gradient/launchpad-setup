# Deploy Observability: Phase Logging & Timing

**Date:** 2026-03-19
**Status:** Draft
**Scope:** `deploy/scripts/lib/common.sh`, `deploy/scripts/lib/deploy.sh`

## Problem

The current deployment process has no persistent logging. All output goes to the terminal (or `/dev/tty`) and is lost after the session ends. When a deployment fails or runs slowly, there's no way to analyze what happened after the fact. The `log_info` and `log_ok` functions are silent in non-VERBOSE mode, and failure only dumps the last 20 lines of a single container's logs.

## Solution

Two complementary features:

### A. Deploy Log File

**Path:** `deploy/generated/deploy.log`

Every deployment appends structured log lines to a persistent file. The log captures phase transitions with timestamps, enabling post-hoc analysis.

**Log format:**

```
[2026-03-19T14:32:01+08:00] [START] deploy v1.0 — K8S_MODE=builtin SSL_MODE=selfsigned DB_MODE=builtin
[2026-03-19T14:32:01+08:00] [PHASE] ssl — started
[2026-03-19T14:32:13+08:00] [OK]    ssl — 12s
[2026-03-19T14:32:13+08:00] [PHASE] ingress — started
[2026-03-19T14:32:58+08:00] [OK]    ingress — 45s
[2026-03-19T14:34:14+08:00] [FAIL]  api — 90s — timeout waiting for healthy
[2026-03-19T14:34:14+08:00] [END]   deploy — FAILED at phase: api — total: 2m13s
```

**Implementation:**

1. Add `deploy_log()` function to `common.sh`:
   - Appends a timestamped line to `${DEPLOY_DIR}/generated/deploy.log`
   - Creates the file on first write (directory already exists at this point)
   - Uses ISO 8601 timestamps with timezone

2. Call sites in `deploy.sh`:
   - `deploy_services()` start: `[START]` line with config summary (K8S_MODE, SSL_MODE, DB_MODE)
   - `_update_step` when status becomes "active": `[PHASE] <key> — started`
   - `_complete_step`: `[OK] <key> — <elapsed>s`
   - `deploy_fail`: `[FAIL] <key> — <elapsed>s — <reason>`
   - `deploy_services()` end: `[END]` line with total elapsed and status

### B. Phase Timing

Track start/end time for each deployment step and display a summary table after deployment completes.

**Data structures:**

```bash
declare -A _ds_start_times   # step key → epoch seconds
_ds_elapsed=()               # step index → elapsed seconds (parallel to _ds_steps)
_deploy_start_time=0         # overall deploy start
```

**Timing integration:**

- When a step transitions to "active", record `$(date +%s)` in `_ds_start_times`
- When `_complete_step` fires, compute `elapsed = $(date +%s) - start` and store in `_ds_elapsed`
- On failure, also compute elapsed for the failed step

**Summary table output (after deploy_services, before show_result):**

TTY mode:
```
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Deploy timing:
    ssl              12s  ✓
    ingress          45s  ✓
    coredns           8s  ✓
    postgres          5s  ✓
    api              38s  ✓
    nginx             3s  ✓
    templates        22s  ✓
    cluster           5s  ✓
    ────────────────────────
    Total:         2m18s
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Non-TTY mode: same content, plain text (no color codes).

**Helper function:** `_format_duration()` converts seconds to human-readable format:
- < 60s: `42s`
- >= 60s: `2m18s`

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `deploy/scripts/lib/common.sh` | Add `deploy_log()` function | ~10 |
| `deploy/scripts/lib/deploy.sh` | Add timing arrays, update `_add_step`/`_complete_step`/`deploy_fail`, add summary output | ~40 |

## Non-Goals

- Command output capture (piping each phase's stdout/stderr to log file) — deferred, higher complexity
- Configuration audit/summary at deploy start — separate concern
- Log rotation — deploy.log is append-only, manual cleanup acceptable for now

## Testing

- Run `./deploy/setup.sh --config deploy/presets/test-server.conf` and verify:
  1. `deploy/generated/deploy.log` exists with correct format
  2. Each phase has `[PHASE]` and `[OK]`/`[FAIL]` entries
  3. Timing summary displays after deployment
  4. `--resume` appends to existing log (doesn't overwrite)
- Run `./deploy/setup.sh --resume` after a failure and verify log continuity
