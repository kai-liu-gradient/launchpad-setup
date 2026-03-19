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

**Implementation:**

1. Add `deploy_log()` function to `common.sh`:
   - Appends a timestamped line to `${DEPLOY_DIR}/generated/deploy.log`
   - Guards with `mkdir -p` on the directory (safe for `--resume` after manual cleanup)
   - Uses ISO 8601 timestamps with timezone: `date '+%Y-%m-%dT%H:%M:%S%z'`

2. Call sites in `deploy.sh`:
   - `deploy_services()` start: `[START]` or `[RESUME]` line with config summary
   - `_complete_step`: `[OK] <key> — <elapsed>s`
   - `_fail_with_cursor`: `[FAIL] <key> — <elapsed>s — <reason>` + partial timing summary (before calling `deploy_fail` which exits)
   - `deploy_services()` end: `[END]` line with total elapsed and status

### B. Phase Timing

Track start/end time for each deployment step and display a summary table after deployment completes (or on failure).

**Data structures — all declared at file scope (before `deploy_services`):**

```bash
declare -A _ds_start_times   # step key → epoch seconds
declare -A _ds_key_by_idx    # step index → step key (reverse lookup)
_ds_elapsed=()               # step index → elapsed seconds (parallel to _ds_steps)
_deploy_start_time=0         # overall deploy start
```

These are file-scope globals so both `deploy_services()` inner functions and `deploy_fail()` can access them.

**Timing integration:**

- `_add_step`: populate `_ds_key_by_idx[idx]=key` for reverse lookup
- When a step transitions to "active" (inside `_update_step`): if status is `"active"` and no start time recorded yet, record `$(date +%s)` in `_ds_start_times[$key]`
- `_complete_step`: compute `elapsed = $(date +%s) - _ds_start_times[$key]`, store in `_ds_elapsed[$idx]`, call `deploy_log "[OK] $key — ${elapsed}s"`
- `_fail_with_cursor`: compute elapsed for the failed step, call `deploy_log "[FAIL] ..."`, print partial timing summary, then call `deploy_fail`

**Summary table output:**

Printed at two points:
1. **On success:** after `deploy_services()` completes, before `show_result()`
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

**Helper function:** `_format_duration()` converts seconds to human-readable format:
- < 60s: `42s`
- 60s–3599s: `2m18s`
- >= 3600s: `1h5m`

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `deploy/scripts/lib/common.sh` | Add `deploy_log()` and `_format_duration()` functions | ~15 |
| `deploy/scripts/lib/deploy.sh` | Add timing arrays (file scope), update `_add_step`/`_complete_step`/`_fail_with_cursor`/`deploy_fail`, add `_print_timing_summary()` | ~60 |

Total: ~75 lines of new code, no existing behavior changed.

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
