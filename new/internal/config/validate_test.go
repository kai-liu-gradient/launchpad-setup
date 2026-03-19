package config

import "testing"

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	if err := Validate(cfg); err != nil {
		t.Errorf("valid config should not error: %v", err)
	}
}

func TestValidate_MissingDomain(t *testing.T) {
	cfg := DefaultConfig("", "admin@example.com")
	if err := Validate(cfg); err == nil {
		t.Error("empty domain should fail validation")
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	cfg := DefaultConfig("example.com", "not-an-email")
	if err := Validate(cfg); err == nil {
		t.Error("invalid email should fail validation")
	}
}

func TestValidate_InvalidSSLMode(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.SSL.Mode = "invalid"
	if err := Validate(cfg); err == nil {
		t.Error("invalid SSL mode should fail validation")
	}
}

func TestValidate_ExternalDBRequiresURLs(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.Database.Mode = "external"
	cfg.Database.URLs = nil
	if err := Validate(cfg); err == nil {
		t.Error("external DB without URLs should fail validation")
	}
}

func TestValidate_ExternalK8sRequiresKubeconfig(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "external"
	cfg.Kubernetes.Kubeconfig = ""
	if err := Validate(cfg); err == nil {
		t.Error("external K8s without kubeconfig should fail validation")
	}
}

func TestValidate_LetsencryptRequiresDNSProvider(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.SSL.Mode = "letsencrypt"
	cfg.SSL.DNSProvider = ""
	if err := Validate(cfg); err == nil {
		t.Error("letsencrypt without dns_provider should fail validation")
	}
}

func TestValidate_InvalidDBMode(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.Database.Mode = "cloud"
	if err := Validate(cfg); err == nil {
		t.Error("invalid DB mode should fail validation")
	}
}

func TestValidate_InvalidK8sMode(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "managed"
	if err := Validate(cfg); err == nil {
		t.Error("invalid K8s mode should fail validation")
	}
}
