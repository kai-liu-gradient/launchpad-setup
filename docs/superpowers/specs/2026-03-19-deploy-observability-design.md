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
[2026-03-19T14:32:01+08:00] [START]  deploy v1.0 — K8S_MODE=builtin SSL_MODE=selfsigned DB_MODE=builtin
[2026-03-19T14:32:01+08:00] [PHASE]  ssl — started
[2026-03-19T14:32:13+08:00] [OK]     ssl — 12s
[2026-03-19T14:32:13+08:00] [PHASE]  ingress — started
[2026-03-19T14:32:58+08:00] [OK]     ingress — 45s
[2026-03-19T14:34:14+08:00] [FAIL]   api — 90s — timeout waiting for healthy
[2026-03-19T14:34:14+08:00] [END]    deploy — FAILED at phase: api — total: 2m13s
```

For `--resume` runs, the first line uses `[RESUME]` instead of `[START]`:
```
[2026-03-19T14:40:01+08:00] [RESUME] deploy v1.0 — K8S_MODE=builtin SSL_MODE=selfsigned DB_MODE=builtin
```

**Resume detection:** `setup.sh` already sets `ACTION="resume"` for `--resume`. `deploy_services()` checks `${ACTION:-}` to decide `[START]` vs `[RESUME]`.

**Implementation:**

1. Add `deploy_log()` function to `common.sh`:
   - Appends a timestamped line to `${DEPLOY_DIR}/generated/deploy.log`
   - Guards with `mkdir -p` on the directory (safe for `--resume` after manual cleanup)
   - Uses ISO 8601 timestamps with timezone: `date '+%Y-%m-%dT%H:%M:%S%z'`

2. Call sites in `deploy.sh`:
   - `deploy_services()` start: `[START]` or `[RESUME]` line with config summary
   - `_update_step` when status becomes "active": emit `[PHASE] <key> — started` via `deploy_log`
   - `_complete_step`: `[OK] <key> — <elapsed>s`
   - `_fail_with_cursor`: `[FAIL] <key> — <elapsed>s — <reason>` + partial timing summary (before calling `deploy_fail` which exits)
   - `deploy_services()` end: `[END]` line with total elapsed and status

### B. Phase Timing

Track start/end time for each deployment step and display a summary table after deployment completes (or on failure).

**Data structures — all at file scope (before `deploy_services`):**

```bash
# Move existing _ds_step_map to file scope alongside new timing arrays.
# This enables deploy_fail() and _print_timing_summary() to access them.
declare -A _ds_step_map       # step key → array index (EXISTING, moved from inside deploy_services)
declare -A _ds_start_times    # step key → epoch seconds (NEW)
declare -A _ds_key_by_idx     # step index → step key, reverse of _ds_step_map (NEW)
_ds_elapsed=()                # step index → elapsed seconds, parallel to _ds_steps (NEW)
_deploy_start_time=0          # overall deploy start epoch (NEW)
```

Moving `_ds_step_map` out of `deploy_services()` to file scope ensures all timing-related data is globally accessible. The existing `_ds_steps` array (index → `status:pct:label`) and `_ds_idx` counter also move to file scope for consistency.

**Timing integration:**

- `_add_step`: populate `_ds_key_by_idx[$idx]=$key` for reverse lookup; initialize `_ds_elapsed[$idx]=0`
- `_update_step`: when status arg is `"active"` and `_ds_start_times[$key]` is unset, record `$(date +%s)` and emit `deploy_log "[PHASE] $key — started"`
- `_complete_step` (both TTY and non-TTY paths): look up key via `_ds_key_by_idx[$idx]`, compute `elapsed = $(date +%s) - _ds_start_times[$key]`, store in `_ds_elapsed[$idx]`, call `deploy_log "[OK] $key — ${elapsed}s"`. The non-TTY `_complete_step` is redefined to include timing logic (currently it only calls `progress_update`).
- `_fail_with_cursor`: look up current step's key, compute elapsed, call `deploy_log "[FAIL] ..."`, call `_print_timing_summary`, then call `deploy_fail`. **All failure paths MUST go through `_fail_with_cursor`**, never call `deploy_fail` directly from step code.

**Summary table output:**

New function `_print_timing_summary()` iterates `_ds_steps` and `_ds_elapsed`, printing a table. Called at two points:
1. **On success:** at end of `deploy_services()`, before returning
2. **On failure:** inside `_fail_with_cursor`, before calling `deploy_fail` (which exits)

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

**Helper function:** `_format_duration()` in `common.sh` — converts seconds to human-readable format:
- < 60s: `42s`
- 60s–3599s: `2m18s`
- >= 3600s: `1h5m`

Placed in `common.sh` because it's a general utility (could be useful for other timing displays in the future).

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `deploy/scripts/lib/common.sh` | Add `deploy_log()` and `_format_duration()` functions | ~20 |
| `deploy/scripts/lib/deploy.sh` | Move `_ds_step_map`/`_ds_steps`/`_ds_idx` to file scope; add timing arrays; update `_add_step`/`_update_step`/`_complete_step` (both TTY and non-TTY)/`_fail_with_cursor`; add `_print_timing_summary()` | ~70 |

Total: ~90 lines of new/modified code.

## Non-Goals

- Command output capture (piping each phase's stdout/stderr to log file) — deferred, higher complexity
- Configuration audit/summary at deploy start — separate concern
- Log rotation — deploy.log is append-only, manual cleanup acceptable for now

## Testing

- Run `./deploy/setup.sh --config deploy/presets/test-server.conf` and verify:
  1. `deploy/generated/deploy.log` exists with correct format
  2. Each phase has `[PHASE]` and `[OK]`/`[FAIL]` entries
  3. Timing summary displays after deployment completes
  4. `--resume` appends to existing log with `[RESUME]` marker
- Run `./deploy/setup.sh --resume` after a failure and verify:
  1. Log continuity (appended, not overwritten)
  2. Partial timing summary on failure includes completed steps
- Verify non-TTY mode (pipe through `cat`): summary table displays without ANSI codes
