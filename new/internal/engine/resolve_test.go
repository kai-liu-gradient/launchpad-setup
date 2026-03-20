package engine

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestResolveTemplateFiles_EmbeddedOnly(t *testing.T) {
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

	assertFileContent(t, filepath.Join(tmpDir, "hello.tar.gz"), "embedded-tar")
	assertFileContent(t, filepath.Join(tmpDir, "hello.yaml"), "embedded-yaml")
	assertFileNotExists(t, filepath.Join(tmpDir, ".gitkeep"))
}

func TestResolveTemplateFiles_ExternalOnly(t *testing.T) {
	embedded := fstest.MapFS{
		"files/.gitkeep": &fstest.MapFile{Data: []byte("")},
	}

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
