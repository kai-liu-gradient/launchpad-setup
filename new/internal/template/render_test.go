package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func fixtureSecrets() *secrets.Secrets {
	return &secrets.Secrets{
		JWTSecret:                      "aaaa",
		JWTRefreshSecret:               "bbbb",
		SessionSecret:                  "cccc",
		EncryptionKey:                  "dddd",
		ClaudeCredentialsEncryptionKey: "eeee",
		SSHKeyEncryptionSecret:         "ffff",
		RSAPublicKey:                   "pub-key",
		RSAPrivateKey:                  "priv-key",
		DBPasswordMain:                 "pass-main",
		DBPasswordMonitoring:           "pass-monitoring",
		DBPasswordEvents:               "pass-events",
		DBPasswordBilling:              "pass-billing",
		DBPasswordStats:                "pass-stats",
		DBPasswordGateway:              "pass-gateway",
		DBPasswordGitea:                "pass-gitea",
		PostgresSuperuserPassword:      "pass-super",
		RedisPassword:                  "pass-redis",
		GatewayAPIKey:                  "gw-key",
		InternalSecret:                 "internal",
		AdminPassword:                  "admin-pass",
	}
}

func fixtureRenderContext() *RenderContext {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	sec := fixtureSecrets()
	derived := ComputeDerived(cfg, sec)
	return &RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: &RuntimeValues{},
	}
}

func TestRenderAll(t *testing.T) {
	ctx := fixtureRenderContext()

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Check .env was created in launchpad/ subdirectory
	envPath := filepath.Join(outDir, "launchpad", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DOMAIN=example.com") {
		t.Error(".env should contain DOMAIN=example.com (found in INGRESS_DOMAIN or BASE_DOMAIN line)")
	}
	if !strings.Contains(content, "SSL_MODE=selfsigned") {
		t.Error(".env should contain SSL_MODE=selfsigned")
	}
}

func TestRenderAll_PreservesRuntime(t *testing.T) {
	ctx := fixtureRenderContext()
	ctx.Runtime = &RuntimeValues{
		GiteaAccessToken: "test-token-123",
		GiteaUser:        "admin",
	}

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	envPath := filepath.Join(outDir, "launchpad", ".env")
	data, _ := os.ReadFile(envPath)
	content := string(data)
	if !strings.Contains(content, "test-token-123") {
		t.Error(".env should contain the runtime Gitea token")
	}
}

func TestRenderAll_OutputFiles(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	expectedFiles := []string{
		"launchpad/.env",
		"gateway/.env",
		"nginx/nginx.conf",
		"launchpad/config/settings.yml",
		"docker-compose.yml",
		"coredns-custom.yaml",
		"kyverno-inject-ca.yaml",
		"kyverno-sync-ca.yaml",
		"values-builtin.yml",
		"launchpad/cron/crontab",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected output file %q not found", f)
		}
	}
}

func TestRenderCompose_BuiltinMode(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "docker-compose.yml"))
	content := string(got)

	if !strings.Contains(content, "postgres:") {
		t.Error("builtin mode should include postgres service")
	}
	if !strings.Contains(content, "redis:") {
		t.Error("builtin mode should include redis service")
	}
	if !strings.Contains(content, "extra_hosts") {
		t.Error("builtin K8s should include extra_hosts")
	}
	if !strings.Contains(content, "pass-super") {
		t.Error("builtin mode should include postgres superuser password")
	}
	if !strings.Contains(content, "pass-redis") {
		t.Error("builtin mode should include redis password")
	}
}

func TestRenderCompose_ExternalMode(t *testing.T) {
	ctx := fixtureRenderContext()
	ctx.Config.Database.Mode = "external"
	ctx.Config.Database.URLs = map[string]string{
		"main":       "postgresql://ext:pass@exthost:5432/db?schema=main",
		"monitoring": "postgresql://ext:pass@exthost:5432/db?schema=monitoring",
		"events":     "postgresql://ext:pass@exthost:5432/db?schema=events",
		"billing":    "postgresql://ext:pass@exthost:5432/db?schema=billing",
		"stats":      "postgresql://ext:pass@exthost:5432/db?schema=stats",
		"gateway":    "postgresql://ext:pass@exthost:5432/db?schema=gateway",
	}
	ctx.Config.Kubernetes.Mode = "external"
	ctx.Config.SSL.Mode = "letsencrypt"
	ctx.Derived = ComputeDerived(ctx.Config, ctx.Secrets)

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "docker-compose.yml"))
	content := string(got)

	if strings.Contains(content, "  postgres:") {
		t.Error("external DB should not include postgres service")
	}
	if strings.Contains(content, "extra_hosts") {
		t.Error("external K8s should not include extra_hosts")
	}
	if strings.Contains(content, "NODE_EXTRA_CA_CERTS") {
		t.Error("letsencrypt mode should not include NODE_EXTRA_CA_CERTS")
	}
}

func TestRenderCompose_SelfsignedGateway(t *testing.T) {
	ctx := fixtureRenderContext()
	ctx.Config.SSL.Mode = "selfsigned"
	ctx.Derived = ComputeDerived(ctx.Config, ctx.Secrets)

	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "docker-compose.yml"))
	content := string(got)

	if !strings.Contains(content, "NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem") {
		t.Error("selfsigned mode should include NODE_EXTRA_CA_CERTS for gateway")
	}
	if !strings.Contains(content, "ca.pem:/etc/ssl/certs/launchpad-ca.pem:ro") {
		t.Error("selfsigned mode should mount CA cert into gateway")
	}
}

func TestRenderNginx(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "nginx", "nginx.conf"))
	content := string(got)

	if !strings.Contains(content, "server_name launchpad.example.com;") {
		t.Error("nginx.conf should contain the launchpad server_name")
	}
	if !strings.Contains(content, "server_name launchpad-gitea.example.com;") {
		t.Error("nginx.conf should contain the gitea server_name")
	}
	if !strings.Contains(content, "server_name *.example.com;") {
		t.Error("nginx.conf should contain the wildcard server_name")
	}
}

func TestRenderSettings(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "launchpad", "config", "settings.yml"))
	content := string(got)

	if !strings.Contains(content, `"example.com"`) {
		t.Error("settings.yml should contain the domain in allowedDomains")
	}
	if !strings.Contains(content, "admin@example.com") {
		t.Error("settings.yml should contain the admin email")
	}
}

func TestRenderGatewayEnv(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "gateway", ".env"))
	content := string(got)

	if !strings.Contains(content, "ALLOWED_API_KEYS=gw-key") {
		t.Error(".env.gateway should contain the gateway API key")
	}
	if !strings.Contains(content, "LAUNCHPAD_INTERNAL_SECRET=internal") {
		t.Error(".env.gateway should contain the internal secret")
	}
	if !strings.Contains(content, "PORT_API=6555") {
		t.Error(".env.gateway should contain PORT_API=6555")
	}
}

func TestRenderStaticFiles(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}
	for _, f := range []string{"kyverno-inject-ca.yaml", "kyverno-sync-ca.yaml", "launchpad/cron/crontab", "values-builtin.yml"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Errorf("static file %s not copied: %v", f, err)
		}
	}
}

func TestRenderDerivedDomainFields(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	d := ComputeDerived(cfg, sec)

	if d.LaunchpadDomain != "launchpad.example.com" {
		t.Errorf("LaunchpadDomain = %q, want %q", d.LaunchpadDomain, "launchpad.example.com")
	}
	if d.GiteaDomain != "launchpad-gitea.example.com" {
		t.Errorf("GiteaDomain = %q, want %q", d.GiteaDomain, "launchpad-gitea.example.com")
	}
	if d.IngressDomain != "example.com" {
		t.Errorf("IngressDomain = %q, want %q", d.IngressDomain, "example.com")
	}
	if d.GatewayPublicURL != "https://launchpad.example.com/gatewayproxy" {
		t.Errorf("GatewayPublicURL = %q, want %q", d.GatewayPublicURL, "https://launchpad.example.com/gatewayproxy")
	}
	if d.GatewayURL != d.GatewayPublicURL {
		t.Errorf("GatewayURL = %q, want it to equal GatewayPublicURL", d.GatewayURL)
	}
	if d.StorageClass != "local-path" {
		t.Errorf("StorageClass = %q, want %q for default", d.StorageClass, "local-path")
	}
	if d.IngressClusterIP != "10.43.0.0" {
		t.Errorf("IngressClusterIP = %q, want %q", d.IngressClusterIP, "10.43.0.0")
	}
}

func TestRenderCoreDNS(t *testing.T) {
	ctx := fixtureRenderContext()
	ctx.Runtime.IngressClusterIP = "10.43.0.100"
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(outDir, "coredns-custom.yaml"))
	content := string(got)

	if !strings.Contains(content, "launchpad.example.com:53") {
		t.Error("coredns should contain the launchpad domain")
	}
	if !strings.Contains(content, "10.43.0.100") {
		t.Error("coredns should contain the ingress cluster IP")
	}
	// Verify CoreDNS template syntax is preserved (escaped {{ .Name }})
	if !strings.Contains(content, "{{ .Name }}") {
		t.Error("coredns should preserve the CoreDNS {{ .Name }} template syntax")
	}
	if !strings.Contains(content, `example\.com`) {
		t.Error("coredns should contain the escaped domain for regex")
	}
}

func TestRenderKyvernoStaticCopy(t *testing.T) {
	ctx := fixtureRenderContext()
	outDir := t.TempDir()
	if err := RenderAll(ctx, outDir); err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Kyverno files should be copied as-is with their {{ }} syntax preserved
	got, _ := os.ReadFile(filepath.Join(outDir, "kyverno-sync-ca.yaml"))
	content := string(got)

	if !strings.Contains(content, "{{request.object.metadata.name}}") {
		t.Error("kyverno-sync-ca.yaml should preserve Kyverno template syntax")
	}
}
