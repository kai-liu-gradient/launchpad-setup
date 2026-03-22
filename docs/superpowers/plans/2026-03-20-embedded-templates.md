# Embedded Templates Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Embed template files (.tar.gz and .yaml) into the launchpad binary so a single file contains everything needed for deployment.

**Architecture:** Use Go's `embed` package to compile template files into the binary at build time. A new `resolveTemplateFiles()` function merges embedded and external files into a temp directory. The existing `importTemplates` logic is unchanged — only its file source changes.

**Tech Stack:** Go `embed`, `io/fs`, `os` (temp dirs)

**Spec:** `docs/superpowers/specs/2026-03-20-embedded-templates-design.md`

---

## File Structure

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/importfiles/importfiles.go` | `//go:embed all:files` — embeds template files into binary |
| Create | `internal/importfiles/files/.gitkeep` | Keeps directory in git when empty (dev builds) |
| Create | `internal/engine/resolve.go` | `resolveTemplateFiles()` — merges embedded + external files |
| Create | `internal/engine/resolve_test.go` | Tests for resolve logic |
| Modify | `internal/engine/import.go:32-42` | Replace hardcoded files/ lookup with `resolveTemplateFiles()` call |
| Modify | `.gitignore` | Add `internal/importfiles/files/*.tar.gz` and `*.yaml` |

---

## Task 1: Create the importfiles embed package

**Files:**
- Create: `internal/importfiles/importfiles.go`
- Create: `internal/importfiles/files/.gitkeep`
- Modify: `.gitignore`

- [ ] **Step 1: Create the embed package**

Create `internal/importfiles/importfiles.go`:

```go
// Package importfiles embeds template git repos (.tar.gz) and YAML template
// definitions (.yaml) that are imported into Gitea during deployment.
// This is separate from internal/template which embeds deployment config
// templates (.tmpl files) used for rendering docker-compose, nginx, etc.
package importfiles

import "embed"

//go:embed all:files
var Files embed.FS
```

- [ ] **Step 2: Create the .gitkeep placeholder**

Create `internal/importfiles/files/.gitkeep` (empty file). This ensures `go:embed all:files` has a valid directory to embed even when no template files are present (development builds).

- [ ] **Step 3: Update .gitignore**

Append to `.gitignore`:

```
internal/importfiles/files/*.tar.gz
internal/importfiles/files/*.yaml
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: SUCCESS — the embed package compiles with only `.gitkeep` in the directory.

- [ ] **Step 5: Commit**

```bash
git add internal/importfiles/ .gitignore
git commit -m "feat: add importfiles embed package for template embedding"
```

---

## Task 2: Implement resolveTemplateFiles

**Files:**
- Create: `internal/engine/resolve.go`
- Create: `internal/engine/resolve_test.go`

- [ ] **Step 1: Write tests for resolveTemplateFiles**

Create `internal/engine/resolve_test.go`:

```go
package engine

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestResolveTemplateFiles_EmbeddedOnly(t *testing.T) {
	// Simulate embedded FS with two files under "files/" prefix
	embedded := fstest.MapFS{
		"files/hello.tar.gz": &fstest.MapFile{Data: []byte("embedded-tar")},
		"files/hello.yaml":   &fstest.MapFile{Data: []byte("embedded-yaml")},
		"files/.gitkeep":     &fstest.MapFile{Data: []byte("")},
	}

	tmpDir, err := resolveTemplateFiles(embedded, "/nonexistent/output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should have the two files (not .gitkeep)
	assertFileContent(t, filepath.Join(tmpDir, "hello.tar.gz"), "embedded-tar")
	assertFileContent(t, filepath.Join(tmpDir, "hello.yaml"), "embedded-yaml")
	assertFileNotExists(t, filepath.Join(tmpDir, ".gitkeep"))
}

func TestResolveTemplateFiles_ExternalOnly(t *testing.T) {
	// Empty embedded FS
	embedded := fstest.MapFS{
		"files/.gitkeep": &fstest.MapFile{Data: []byte("")},
	}

	// Create external files/ directory next to "output"
	baseDir := t.TempDir()
	outputDir := filepath.Join(baseDir, "generated")
	os.MkdirAll(outputDir, 0755)
	filesDir := filepath.Join(baseDir, "files")
	os.MkdirAll(filesDir, 0755)
	os.WriteFile(filepath.Join(filesDir, "ext.tar.gz"), []byte("external-tar"), 0644)

	tmpDir, err := resolveTemplateFiles(embedded, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	assertFileContent(t, filepath.Join(tmpDir, "ext.tar.gz"), "external-tar")
}

func TestResolveTemplateFiles_ExternalOverridesEmbedded(t *testing.T) {
	embedded := fstest.MapFS{
		"files/hello.tar.gz": &fstest.MapFile{Data: []byte("embedded-version")},
		"files/hello.yaml":   &fstest.MapFile{Data: []byte("embedded-yaml")},
	}

	baseDir := t.TempDir()
	outputDir := filepath.Join(baseDir, "generated")
	os.MkdirAll(outputDir, 0755)
	filesDir := filepath.Join(baseDir, "files")
	os.MkdirAll(filesDir, 0755)
	// Override hello.tar.gz, leave hello.yaml from embedded
	os.WriteFile(filepath.Join(filesDir, "hello.tar.gz"), []byte("external-version"), 0644)

	tmpDir, err := resolveTemplateFiles(embedded, outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	assertFileContent(t, filepath.Join(tmpDir, "hello.tar.gz"), "external-version")
	assertFileContent(t, filepath.Join(tmpDir, "hello.yaml"), "embedded-yaml")
}

func TestResolveTemplateFiles_BothEmpty(t *testing.T) {
	embedded := fstest.MapFS{
		"files/.gitkeep": &fstest.MapFile{Data: []byte("")},
	}

	tmpDir, err := resolveTemplateFiles(embedded, "/nonexistent/output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should return empty dir (no files)
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir, got %d entries", len(entries))
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
		return
	}
	if string(data) != expected {
		t.Errorf("file %s: got %q, want %q", path, string(data), expected)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected %s to not exist", path)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestResolveTemplateFiles -v`
Expected: FAIL — `resolveTemplateFiles` not defined

- [ ] **Step 3: Implement resolveTemplateFiles**

Create `internal/engine/resolve.go`:

```go
package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// resolveTemplateFiles merges embedded and external template files into a temp
// directory. External files override embedded ones with the same name.
// The caller must clean up the returned directory with os.RemoveAll.
func resolveTemplateFiles(embedded fs.FS, outputDir string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "launchpad-templates-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	// Phase 1: extract embedded files (skip .gitkeep)
	sub, err := fs.Sub(embedded, "files")
	if err == nil {
		fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || path == "." {
				return nil
			}
			if d.Name() == ".gitkeep" {
				return nil
			}
			data, readErr := fs.ReadFile(sub, path)
			if readErr != nil {
				return nil // skip, best effort
			}
			return os.WriteFile(filepath.Join(tmpDir, d.Name()), data, 0644)
		})
	}

	// Phase 2: copy external files (overwrite embedded)
	externalDir := findExternalFilesDir(outputDir)
	if externalDir != "" {
		entries, _ := os.ReadDir(externalDir)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".yaml") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(externalDir, name))
			if err != nil {
				continue // skip, best effort
			}
			os.WriteFile(filepath.Join(tmpDir, name), data, 0644)
		}
	}

	return tmpDir, nil
}

// findExternalFilesDir looks for a "files" directory using the existing
// two-level fallback: first next to output dir, then one level up.
func findExternalFilesDir(outputDir string) string {
	candidates := []string{
		filepath.Join(filepath.Dir(outputDir), "files"),
		filepath.Join(filepath.Dir(filepath.Dir(outputDir)), "files"),
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run TestResolveTemplateFiles -v`
Expected: PASS — all 4 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/engine/resolve.go internal/engine/resolve_test.go
git commit -m "feat: add resolveTemplateFiles to merge embedded and external templates"
```

---

## Task 3: Wire resolveTemplateFiles into importTemplates

**Files:**
- Modify: `internal/engine/import.go:32-42`

- [ ] **Step 1: Add import for importfiles package**

In `internal/engine/import.go`, add to imports:

```go
"github.com/gradient8/launchpad/internal/importfiles"
```

- [ ] **Step 2: Replace files directory lookup with resolveTemplateFiles**

Replace lines 32-42 of `import.go` (the `filesDir` lookup block):

```go
// Find files directory (look in project root)
// The "files" directory contains .tar.gz repos and .yaml templates
filesDir := filepath.Join(filepath.Dir(e.output), "files")
if _, err := os.Stat(filesDir); os.IsNotExist(err) {
    // Try parent directory
    filesDir = filepath.Join(filepath.Dir(filepath.Dir(e.output)), "files")
    if _, err := os.Stat(filesDir); os.IsNotExist(err) {
        e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "No files directory"})
        return nil
    }
}
```

With:

```go
// Resolve template files: merge embedded (built into binary) with external files/ directory
filesDir, err := resolveTemplateFiles(importfiles.Files, e.output)
if err != nil {
    return fmt.Errorf("resolving template files: %w", err)
}
defer os.RemoveAll(filesDir)

// Check if any files were resolved
tarFiles, _ := filepath.Glob(filepath.Join(filesDir, "*.tar.gz"))
yamlFiles, _ := filepath.Glob(filepath.Join(filesDir, "*.yaml"))
if len(tarFiles) == 0 && len(yamlFiles) == 0 {
    e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "No template files found"})
    return nil
}
```

Also remove the duplicate `tarFiles` and `yamlFiles` Glob calls that were below (lines 45 and 117), since they're now done above. Use the variables from above instead.

- [ ] **Step 3: Build and run all tests**

Run: `go build ./... && go test ./...`
Expected: All pass — no behavioral change, just different file source.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/import.go
git commit -m "feat: wire resolveTemplateFiles into importTemplates for embedded template support"
```

---

## Task 4: Build with embedded files and verify

- [ ] **Step 1: Copy template files and build release binary**

```bash
mkdir -p internal/importfiles/files
cp ../../files/*.tar.gz ../../files/*.yaml internal/importfiles/files/
GOOS=linux GOARCH=amd64 go build -o /tmp/launchpad-linux ./cmd/launchpad/
rm -f internal/importfiles/files/*.tar.gz internal/importfiles/files/*.yaml
```

(Paths assume working directory is `new/`, and `files/` is at the repo root `../../files/` relative to `new/`)

- [ ] **Step 2: Verify binary size increased**

```bash
ls -lh /tmp/launchpad-linux
```

Expected: ~18MB (was ~15MB before embedding ~2.6MB of templates)

- [ ] **Step 3: Deploy and test on server**

```bash
scp /tmp/launchpad-linux root@10.233.201.133:/usr/local/bin/launchpad
```

On server — test that templates import without needing external `files/` directory:
```bash
# Remove external files/ if present
rm -rf ~/launchpad/files
# Run install (or just the import step)
launchpad install
```

Templates should import successfully from the embedded files.

- [ ] **Step 4: Commit any final adjustments**

```bash
git add -A
git commit -m "chore: verify embedded templates build and deployment"
```
