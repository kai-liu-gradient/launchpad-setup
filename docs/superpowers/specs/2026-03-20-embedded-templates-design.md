# Embedded Templates Design

## Goal

Embed template files (`.tar.gz` repos and `.yaml` definitions) into the `launchpad` binary so users get a single self-contained file with all dependencies. External `files/` directory can override or extend the built-in templates.

## Architecture

Go's `embed` package compiles the contents of `internal/importfiles/files/` into the binary at build time. At deploy time, a resolution step merges embedded and external files into a temporary directory, which the existing import logic consumes.

**Note:** The project already has an embed system at `internal/template/embed.go` (`//go:embed all:files`) for deployment config templates (`.tmpl` files like `docker-compose.yml.tmpl`, `nginx.conf.template`, etc.) used by `template.RenderAll()`. The new embed is for a completely different purpose: import templates (git repo tarballs and YAML definitions) used by `engine.importTemplates()`. They are placed in separate packages to avoid confusion.

### File Layout

```
internal/
  importfiles/
    importfiles.go     # //go:embed all:files
    files/             # populated at build time from top-level files/
      .gitkeep         # keeps directory in git when empty
      *.tar.gz         # template git repos (copied before release build)
      *.yaml           # template definitions (copied before release build)
```

### Resolution Priority

When `importTemplates` runs:

1. Extract all embedded files to a temp directory.
2. Check for external `files/` directory using the existing two-level fallback:
   - First: `{outputDir}/../files/` (install dir)
   - Then: `{outputDir}/../../files/` (parent of install dir)
3. If external `files/` found, copy its contents into the temp directory — overwriting any same-name embedded files.
4. Import from the merged temp directory.
5. Clean up temp directory after import.

This means:
- **No external `files/`**: built-in templates are used (standard case).
- **External `files/` with extra files**: extra templates are added alongside built-in ones.
- **External `files/` with same-name files**: external versions override built-in ones.

## Components

### `internal/importfiles/importfiles.go`

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

Uses `all:files` (consistent with existing `internal/template/embed.go`) to include any hidden files.

If `files/` contains only `.gitkeep` at compile time, `Files` is valid but effectively empty — the binary still builds, and the system falls back to external `files/` only.

### `resolveTemplateFiles(outputDir string) (string, error)`

New function in the engine package:

1. Create temp directory.
2. Use `fs.Sub(importfiles.Files, "files")` to get a clean FS without the `files/` prefix.
3. Walk the sub-FS, copy each file (skip `.gitkeep`) to temp dir.
4. Check for external `files/` using the existing two-level fallback:
   - `filepath.Join(filepath.Dir(outputDir), "files")`
   - `filepath.Join(filepath.Dir(filepath.Dir(outputDir)), "files")`
5. If found, copy all `.tar.gz` and `.yaml` files from it into temp dir (overwriting duplicates).
6. Return temp dir path.

**Error handling:** If temp dir creation fails, return error (aborts import). If individual file copies fail, log a warning and continue — partial templates are better than none. If both embedded and external are empty, return empty dir (import step will log "No files directory" equivalent and skip gracefully).

### `importTemplates` changes

Minimal change: replace the existing `filesDir` lookup (lines 32-42 of `import.go`) with a call to `resolveTemplateFiles(e.output)`. The rest of the import logic (tar.gz extraction, git push, yaml import) is unchanged. Add `defer os.RemoveAll(tmpDir)` for cleanup.

The standalone `import-templates` CLI command also uses the engine's `importTemplates`, so it automatically benefits from embedded files too.

## Build Process

### Development

Developers don't need templates in `internal/importfiles/files/` — the directory has only `.gitkeep`. Templates come from the external `files/` directory during testing.

### Release Build

```makefile
build-release:
	@mkdir -p internal/importfiles/files
	@cp -f files/*.tar.gz files/*.yaml internal/importfiles/files/ 2>/dev/null || true
	GOOS=linux GOARCH=amd64 go build -o dist/launchpad ./cmd/launchpad/
	@rm -f internal/importfiles/files/*.tar.gz internal/importfiles/files/*.yaml
```

The final `rm` step cleans up copied binaries to prevent accidental git commits.

### .gitignore

Add to `.gitignore`:
```
internal/importfiles/files/*.tar.gz
internal/importfiles/files/*.yaml
```

### Binary Size Impact

Current `files/` directory is ~2.6MB. Binary grows from ~15MB to ~18MB. Acceptable.

## What Does NOT Change

- Template import logic (tar.gz extraction, git init/push, yaml validation/creation) — unchanged.
- Gitea bootstrap flow — unchanged.
- Config, secrets, derived values — unchanged.
- Docker image handling — unchanged (images still pulled from registry; offline image support is a future enhancement).

## Future: Offline Image Bundle

Not in scope for this spec. A future `launchpad bundle` command could:
- `docker save` all images referenced in `versions.conf`
- Package them alongside the binary
- `docker load` during install if local images are detected

This is independent work that builds on top of this spec's pattern.

## Testing

- **Unit test**: `resolveTemplateFiles` returns correct merged file list when:
  - Only embedded files exist
  - Only external files exist
  - Both exist with overlapping names (external wins)
  - Both are empty (returns empty dir, no error)
- **Integration test**: Build with embedded files, run import step, verify templates appear in Gitea.
- **Edge case**: Empty `internal/importfiles/files/` (only `.gitkeep`) compiles and runs (falls back to external only).
