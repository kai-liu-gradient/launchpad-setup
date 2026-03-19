package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
)

func TestPreflightChecks_BuiltinRequiresHelm(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "builtin"
	checks := RequiredTools(cfg)

	found := false
	for _, c := range checks {
		if c.Name == "helm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("builtin K8s should require helm")
	}
}

func TestPreflightChecks_ExternalSkipsHelm(t *testing.T) {
	cfg := config.DefaultConfig("example.com", "admin@example.com")
	cfg.Kubernetes.Mode = "external"
	checks := RequiredTools(cfg)

	for _, c := range checks {
		if c.Name == "helm" {
			t.Error("external K8s should not require helm")
		}
	}
}
