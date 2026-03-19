package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAll(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	derived := ComputeDerived(cfg, sec)
	runtime := &RuntimeValues{}

	ctx := &RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Check .env was created
	envPath := filepath.Join(outDir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DOMAIN=example.com") {
		t.Error(".env should contain DOMAIN=example.com")
	}
	if !strings.Contains(content, "SSL_MODE=selfsigned") {
		t.Error(".env should contain SSL_MODE=selfsigned")
	}
}

func TestRenderAll_PreservesRuntime(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	derived := ComputeDerived(cfg, sec)
	runtime := &RuntimeValues{
		GiteaAccessToken: "test-token-123",
		GiteaUser:        "admin",
	}

	ctx := &RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	envPath := filepath.Join(outDir, ".env")
	data, _ := os.ReadFile(envPath)
	content := string(data)
	if !strings.Contains(content, "test-token-123") {
		t.Error(".env should contain the runtime Gitea token")
	}
}
