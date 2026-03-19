package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("example.com", "admin@example.com")
	if cfg.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "example.com")
	}
	if cfg.Subdomain != "launchpad" {
		t.Errorf("Subdomain = %q, want %q", cfg.Subdomain, "launchpad")
	}
	if cfg.AdminEmail != "admin@example.com" {
		t.Errorf("AdminEmail = %q, want %q", cfg.AdminEmail, "admin@example.com")
	}
	if cfg.SSL.Mode != "selfsigned" {
		t.Errorf("SSL.Mode = %q, want %q", cfg.SSL.Mode, "selfsigned")
	}
	if cfg.Database.Mode != "builtin" {
		t.Errorf("Database.Mode = %q, want %q", cfg.Database.Mode, "builtin")
	}
	if cfg.Kubernetes.Mode != "builtin" {
		t.Errorf("Kubernetes.Mode = %q, want %q", cfg.Kubernetes.Mode, "builtin")
	}
	if cfg.Storage.Mode != "local" {
		t.Errorf("Storage.Mode = %q, want %q", cfg.Storage.Mode, "local")
	}
	if cfg.Performance.APIReplicas != 1 {
		t.Errorf("Performance.APIReplicas = %d, want 1", cfg.Performance.APIReplicas)
	}
	if cfg.Performance.DBConnLimit != 100 {
		t.Errorf("Performance.DBConnLimit = %d, want 100", cfg.Performance.DBConnLimit)
	}
	if cfg.Images.Registry == "" {
		t.Error("Images.Registry should not be empty")
	}
}
