# Project Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the Go project from `new/` to the repository root, archive legacy bash artifacts into `old/`, producing a standard Go project layout.

**Architecture:** File-only restructuring — move directories, update .gitignore and deploy script paths. Zero Go code changes.

**Tech Stack:** Go 1.24, bash, git

**Spec:** `docs/superpowers/specs/2026-03-23-project-restructure-design.md`

---

### Task 1: Archive legacy directories into old/

**Files:**
- Move: `deploy/` → `old/deploy/`
- Move: `tests/` → `old/tests/`

- [ ] **Step 1: Move deploy/ into old/**

```bash
mv deploy/ old/deploy/
```

- [ ] **Step 2: Move tests/ into old/**

```bash
mv tests/ old/tests/
```

- [ ] **Step 3: Verify old/ structure**

```bash
ls old/
```

Expected: `deploy/  docker-compose.override.yml  gateway/  helm/  launchpad/  register.sh  tests/`

---

### Task 2: Move import files into Go embed directory

**Files:**
- Move: `files/*.tar.gz`, `files/*.yaml` → `new/internal/importfiles/files/`
- Delete: `files/` directory

- [ ] **Step 1: Copy import files into embed directory**

```bash
cp files/ani-code-main.tar.gz files/nodejs-helloworld-main.tar.gz files/test-openclaw-main.tar.gz files/nodejs-helloworld.yaml files/test-openclaw.yaml new/internal/importfiles/files/
```

- [ ] **Step 2: Verify files landed correctly**

```bash
ls -la new/internal/importfiles/files/
```

Expected: 5 files (3 tar.gz + 2 yaml) plus `.gitkeep`

- [ ] **Step 3: Remove files/ directory**

```bash
rm -rf files/
```

---

### Task 3: Promote new/ contents to root

**Files:**
- Move: `new/cmd/` → `cmd/`
- Move: `new/internal/` → `internal/`
- Move: `new/go.mod` → `go.mod`
- Move: `new/go.sum` → `go.sum`
- Move: `new/Makefile` → `Makefile`
- Move: `new/scripts/` → `scripts/`
- Merge: `new/docs/superpowers/` into `docs/superpowers/`
- Skip: `new/bin/`, `new/launchpad` (compiled binaries)
- Skip: `new/README.md` (minimal, root README already exists)
- Skip: `new/.gitignore` (will be handled in Task 4)

- [ ] **Step 1: Move Go source directories**

```bash
mv new/cmd/ cmd/
mv new/internal/ internal/
```

- [ ] **Step 2: Move Go module files and Makefile**

```bash
mv new/go.mod go.mod
mv new/go.sum go.sum
mv new/Makefile Makefile
```

- [ ] **Step 3: Move scripts directory**

```bash
mv new/scripts/ scripts/
```

- [ ] **Step 4: Merge new/docs into existing docs**

```bash
cp -r new/docs/superpowers/plans/* docs/superpowers/plans/
cp -r new/docs/superpowers/specs/* docs/superpowers/specs/
```

- [ ] **Step 5: Delete remaining new/ directory**

```bash
rm -rf new/
```

- [ ] **Step 6: Verify root structure**

```bash
ls -la
```

Expected top-level: `cmd/`, `internal/`, `scripts/`, `docs/`, `old/`, `go.mod`, `go.sum`, `Makefile`, `README.md`, `README.zh.md`, `.gitignore`

---

### Task 4: Update deploy script paths

**Files:**
- Modify: `scripts/deploy.sh`

After promoting to root, `deploy.sh` is at `scripts/deploy.sh` and the project root is one level up. The path references need updating.

- [ ] **Step 1: Update deploy.sh**

Replace the Configuration section with:

```bash
# Configuration
REMOTE_HOST="${REMOTE_HOST:-root@10.233.201.133}"
REMOTE_PATH="${REMOTE_PATH:-/usr/local/bin/launchpad}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
EMBED_DIR="$PROJECT_DIR/internal/importfiles/files"
OUT="$PROJECT_DIR/bin/launchpad-linux"
```

Changes:
- Remove `REPO_ROOT` (no longer needed — project dir IS repo root)
- Remove `FILES_DIR` (files are already in embed dir permanently)

Also update the "Embed template files" section — files are already embedded, no copy needed. Remove the embed and cleanup steps:

```bash
# Build
echo "=> Building linux/amd64..."
cd "$PROJECT_DIR"
GOOS=linux GOARCH=amd64 go build -o "$OUT" ./cmd/launchpad/

echo "=> Binary: $OUT ($(du -h "$OUT" | awk '{print $1}'))"

# Deploy
if [[ "${1:-}" == "--local" ]]; then
    echo "=> Local build only, skipping deploy."
else
    echo "=> Deploying to $REMOTE_HOST:$REMOTE_PATH..."
    scp -o ConnectTimeout=15 "$OUT" "$REMOTE_HOST:$REMOTE_PATH"
    echo "=> Done."
fi
```

---

### Task 5: Update .gitignore

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Replace .gitignore contents**

```
old/
bin/
.superpowers/
internal/importfiles/files/*.tar.gz
internal/importfiles/files/*.yaml
```

Note: The last two lines match the pattern from `new/.gitignore` — the tar.gz and yaml files are large binary assets that should not be committed to git. They are copied into the embed directory at build time or kept locally.

- [ ] **Step 2: Verify git status**

```bash
git status
```

Expected: Shows moved/deleted/new files. The import files (tar.gz, yaml) should NOT appear as new untracked files.

---

### Task 6: Verify build and tests

- [ ] **Step 1: Build the binary**

```bash
go build ./cmd/launchpad
```

Expected: Succeeds with no errors. (Note: if importfiles are gitignored and not present, build still succeeds — the embed uses `.gitkeep` as fallback)

- [ ] **Step 2: Run Go tests**

```bash
go test ./...
```

Expected: All tests pass.

- [ ] **Step 3: Verify deploy script syntax**

```bash
bash -n scripts/deploy.sh
```

Expected: No syntax errors.

---

### Task 7: Commit

- [ ] **Step 1: Stage all changes**

```bash
git add -A
```

- [ ] **Step 2: Review staged changes**

```bash
git status
```

Verify: No sensitive files (`.secrets`, `.env`) are staged. `old/` and `bin/` should not appear.

- [ ] **Step 3: Commit**

```bash
git commit -m "refactor: promote Go project to root, archive legacy bash scripts to old/"
```
