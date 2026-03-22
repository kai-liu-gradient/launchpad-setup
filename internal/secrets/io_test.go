package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".secrets")
	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file perms = %o, want 0600", info.Mode().Perm())
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.JWTSecret != original.JWTSecret {
		t.Error("JWTSecret mismatch after save/load")
	}
	if loaded.RSAPublicKey != original.RSAPublicKey {
		t.Error("RSAPublicKey mismatch after save/load")
	}
}
