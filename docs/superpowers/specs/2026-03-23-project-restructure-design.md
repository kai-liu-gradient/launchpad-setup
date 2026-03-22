# Project Restructure: Promote Go Project to Root

**Date:** 2026-03-23
**Status:** Approved

## Summary

Replace the current mixed project layout with a standard Go project structure by promoting the contents of `new/` to the repository root. Archive legacy bash-based deployment artifacts into `old/`. Preserve all docs.

## Motivation

The Go-based `launchpad` CLI (in `new/`) has been tested and deployed successfully on the remote server. It fully replaces the legacy bash deployment scripts (`deploy/`). The project structure should reflect this: Go code as the primary citizen, legacy artifacts archived.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Go code location | Root directory (`cmd/`, `internal/`) | Standard Go project layout; IDE/linter/CI work out of the box |
| Module path | Keep `github.com/gradient8/launchpad` | Zero code changes; deployment is via scp, not `go install` |
| Legacy `deploy/` | Move to `old/deploy/` | Archive for reference |
| Legacy `tests/` | Move to `old/tests/` | Bash e2e tests are for the old bash scheme |
| `files/` (tar.gz, yaml) | Move into `internal/importfiles/files/` | Embedded into the Go binary via `//go:embed` |
| `old/` directory | gitignored | Contains `generated/.secrets` and other sensitive/local files |
| `bin/` directory | gitignored | Compiled binaries should not be committed |
| README files | Preserve as-is | Content update deferred to a later task |

## Target Directory Structure

```
launchpad-setup/
├── cmd/
│   ├── launchpad/main.go
│   └── gen-versions/main.go
├── internal/
│   ├── cli/
│   ├── config/
│   ├── importfiles/
│   │   ├── files/            # tar.gz + yaml from root files/
│   │   └── importfiles.go
│   ├── secrets/
│   ├── settings/
│   ├── template/
│   │   └── files/            # embedded config templates
│   └── tui/
│       ├── components/
│       ├── panel/
│       ├── progress/
│       ├── settings/
│       └── wizard/
├── go.mod
├── go.sum
├── Makefile
├── bin/                      # gitignored
├── old/                      # gitignored
│   ├── deploy/               # moved from root deploy/
│   ├── tests/                # moved from root tests/
│   ├── launchpad/
│   ├── helm/
│   ├── gateway/
│   └── register.sh
├── docs/                     # unchanged
├── README.md                 # unchanged
├── README.zh.md              # unchanged
└── .gitignore
```

## File Movement Plan

### Step 1: Archive legacy directories into old/

```
deploy/  →  old/deploy/
tests/   →  old/tests/
```

### Step 2: Move import files

Copy all `.tar.gz` and `.yaml` files from `files/` into `internal/importfiles/files/`, then remove the `files/` directory:

```
files/ani-code-main.tar.gz        →  internal/importfiles/files/
files/nodejs-helloworld-main.tar.gz →  internal/importfiles/files/
files/nodejs-helloworld.yaml       →  internal/importfiles/files/
files/test-openclaw-main.tar.gz    →  internal/importfiles/files/
files/test-openclaw.yaml           →  internal/importfiles/files/
```

Then delete the now-empty `files/` directory.

### Step 3: Promote new/ to root

```
new/cmd/        →  cmd/
new/internal/   →  internal/
new/go.mod      →  go.mod
new/go.sum      →  go.sum (if exists)
new/Makefile    →  Makefile
```

Do NOT move `new/bin/` — compiled binaries are ephemeral.

Then delete the now-empty `new/` directory.

### Step 4: Update .gitignore

```
old/
bin/
.superpowers/
```

### Step 5: Verify

```bash
go build ./cmd/launchpad
go test ./...
```

Both must pass with zero code changes.

### Step 6: Commit

Single commit with all structural changes.

## Code Changes

**None.** All Go source files, import paths, and the module name remain identical. Only file locations change within the repository.

## .gitignore (final)

```
old/
bin/
.superpowers/
```

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Sensitive files in old/ committed | old/ stays gitignored |
| Compiled binaries committed | bin/ gitignored |
| Import paths break | Module path unchanged; internal/ relative path unchanged |
| Git history shows mass rename | Unavoidable but acceptable; `git log --follow` preserves per-file history |

## Out of Scope

- README content update (deferred)
- CI/CD pipeline changes (none exist currently)
- New Go tests for the restructured layout
