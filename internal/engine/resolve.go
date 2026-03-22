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
		fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
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
			os.WriteFile(filepath.Join(tmpDir, name), data, 0644) //nolint:errcheck
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
