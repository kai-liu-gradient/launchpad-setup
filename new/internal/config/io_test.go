package config

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".setup.yaml")

	original := DefaultConfig("example.com")
	original.SSL.Mode = "letsencrypt"
	original.SSL.DNSProvider = "cloudflare"

	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Domain != original.Domain {
		t.Errorf("Domain = %q, want %q", loaded.Domain, original.Domain)
	}
	if loaded.SSL.Mode != "letsencrypt" {
		t.Errorf("SSL.Mode = %q, want %q", loaded.SSL.Mode, "letsencrypt")
	}
	if loaded.SSL.DNSProvider != "cloudflare" {
		t.Errorf("SSL.DNSProvider = %q, want %q", loaded.SSL.DNSProvider, "cloudflare")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}
