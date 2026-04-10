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
	if !names["docker"] {
		t.Error("builtin K8s should require docker")
	}
	if !names["curl"] {
		t.Error("builtin K8s should require curl")
	}
	if !names["helm"] {
		t.Error("builtin K8s should require helm")
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
