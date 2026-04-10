package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
)

func TestPreflightChecks_BuiltinRequiresHelmSkipsKubectl(t *testing.T) {
	cfg := config.DefaultConfig("example.com")
	cfg.Kubernetes.Mode = "builtin"
	checks := RequiredTools(cfg)

	names := map[string]bool{}
	for _, c := range checks {
		names[c.Name] = true
	}
	required := []string{"docker", "curl", "bash", "git", "tar", "sed", "bc", "openssl", "crontab", "helm"}
	for _, name := range required {
		if !names[name] {
			t.Errorf("builtin K8s should require %s", name)
		}
	}
}

func TestPreflightChecks_ExternalRequiresKubectlHelm(t *testing.T) {
	cfg := config.DefaultConfig("example.com")
	cfg.Kubernetes.Mode = "external"
	checks := RequiredTools(cfg)

	names := map[string]bool{}
	for _, c := range checks {
		names[c.Name] = true
	}
	if !names["kubectl"] {
		t.Error("external K8s should require kubectl")
	}
	if !names["helm"] {
		t.Error("external K8s should require helm")
	}
}
