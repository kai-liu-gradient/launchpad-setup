package config

import "testing"

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig("example.com")
	if err := Validate(cfg); err != nil {
		t.Errorf("valid config should not error: %v", err)
	}
}

func TestValidate_MissingDomain(t *testing.T) {
	cfg := DefaultConfig("")
	if err := Validate(cfg); err == nil {
		t.Error("empty domain should fail validation")
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.AdminEmail = "not-an-email"
	if err := Validate(cfg); err == nil {
		t.Error("invalid email should fail validation")
	}
}

func TestValidate_InvalidSSLMode(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.SSL.Mode = "invalid"
	if err := Validate(cfg); err == nil {
		t.Error("invalid SSL mode should fail validation")
	}
}

func TestValidate_ExternalDBRequiresURLs(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.Database.Mode = "external"
	cfg.Database.URLs = nil
	if err := Validate(cfg); err == nil {
		t.Error("external DB without URLs should fail validation")
	}
}

func TestValidate_ExternalK8sRequiresKubeconfig(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.Kubernetes.Mode = "external"
	cfg.Kubernetes.Kubeconfig = ""
	if err := Validate(cfg); err == nil {
		t.Error("external K8s without kubeconfig should fail validation")
	}
}

func TestValidate_LetsencryptRequiresDNSProvider(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.SSL.Mode = "letsencrypt"
	cfg.SSL.DNSProvider = ""
	if err := Validate(cfg); err == nil {
		t.Error("letsencrypt without dns_provider should fail validation")
	}
}

func TestValidate_InvalidDBMode(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.Database.Mode = "cloud"
	if err := Validate(cfg); err == nil {
		t.Error("invalid DB mode should fail validation")
	}
}

func TestValidate_InvalidK8sMode(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.Kubernetes.Mode = "managed"
	if err := Validate(cfg); err == nil {
		t.Error("invalid K8s mode should fail validation")
	}
}

func TestValidate_GiteaSubdomainSameSuffix(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.GiteaSubdomain = "git"
	if err := Validate(cfg); err != nil {
		t.Errorf("gitea subdomain with same domain suffix should pass: %v", err)
	}
}

func TestValidate_GiteaSubdomainEmpty(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.GiteaSubdomain = ""
	if err := Validate(cfg); err != nil {
		t.Errorf("empty gitea subdomain (default) should pass: %v", err)
	}
}

func TestValidate_ProjectDomainParent(t *testing.T) {
	cfg := DefaultConfig("corp.example.com")
	cfg.ProjectDomain = "example.com"
	if err := Validate(cfg); err != nil {
		t.Errorf("parent domain should pass: %v", err)
	}
}

func TestValidate_ProjectDomainSame(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.ProjectDomain = "example.com"
	if err := Validate(cfg); err != nil {
		t.Errorf("same domain should pass: %v", err)
	}
}

func TestValidate_ProjectDomainEmpty(t *testing.T) {
	cfg := DefaultConfig("example.com")
	cfg.ProjectDomain = ""
	if err := Validate(cfg); err != nil {
		t.Errorf("empty project domain should pass: %v", err)
	}
}

func TestValidate_ProjectDomainUnrelated(t *testing.T) {
	cfg := DefaultConfig("corp.example.com")
	cfg.ProjectDomain = "other.com"
	if err := Validate(cfg); err == nil {
		t.Error("unrelated project domain should fail validation")
	}
}

func TestDomainSuffix(t *testing.T) {
	tests := []struct {
		fqdn string
		want string
	}{
		{"launchpad.example.com", "example.com"},
		{"example.com", "example.com"},
		{"app.dev.example.com", "example.com"},
		{"a.b.c.d.com", "d.com"},
	}
	for _, tt := range tests {
		got := domainSuffix(tt.fqdn)
		if got != tt.want {
			t.Errorf("domainSuffix(%q) = %q, want %q", tt.fqdn, got, tt.want)
		}
	}
}
