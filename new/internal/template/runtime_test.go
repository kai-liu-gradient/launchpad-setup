package template

import (
	"path/filepath"
	"testing"
)

func TestRuntimeValues_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".runtime.yaml")

	rv := &RuntimeValues{GiteaAccessToken: "token-123", GiteaUser: "admin"}
	if err := rv.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadRuntime(path)
	if err != nil {
		t.Fatalf("LoadRuntime failed: %v", err)
	}
	if loaded.GiteaAccessToken != "token-123" {
		t.Error("GiteaAccessToken mismatch")
	}
}

func TestRuntimeValues_LoadMissing(t *testing.T) {
	rv, err := LoadRuntime("/nonexistent")
	if err != nil {
		t.Error("missing file should return empty RuntimeValues, not error")
	}
	if rv.GiteaAccessToken != "" {
		t.Error("missing file should return empty token")
	}
}
