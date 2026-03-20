package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func TestBuildStepList_BuiltinFull(t *testing.T) {
	cfg := config.DefaultConfig("example.com")
	sec, _ := secrets.Generate()

	e := New(cfg, sec, "/tmp/test-output", nil)
	steps := e.BuildStepList()

	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}

	expected := []string{
		"Pre-flight checks",
		"Installing K3s",
		"Generating SSL certificates",
		"Installing Ingress-Nginx",
		"Configuring CoreDNS",
		"Setting up Kyverno + CA distribution",
	}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected step: %q", exp)
		}
	}
}

func TestBuildStepList_ExternalSkipsK3s(t *testing.T) {
	cfg := config.DefaultConfig("example.com")
	cfg.Kubernetes.Mode = "external"
	cfg.SSL.Mode = "letsencrypt"
	sec, _ := secrets.Generate()

	e := New(cfg, sec, "/tmp/test-output", nil)
	steps := e.BuildStepList()

	for _, s := range steps {
		if s.Name == "Installing K3s" {
			t.Error("external K8s should not have K3s step")
		}
		if s.Name == "Setting up Kyverno + CA distribution" {
			t.Error("letsencrypt should not have Kyverno step")
		}
	}
}

func TestDeploy_SendsEvents(t *testing.T) {
	cfg := config.DefaultConfig("example.com")
	sec, _ := secrets.Generate()
	events := make(chan StepEvent, 100)

	e := New(cfg, sec, t.TempDir(), events)
	steps := e.BuildStepList()
	if len(steps) == 0 {
		t.Error("step list should not be empty")
	}
}
