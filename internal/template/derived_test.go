package template

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func testConfig() *config.Config {
	return config.DefaultConfig("example.com")
}

func testSecrets() *secrets.Secrets {
	s, _ := secrets.Generate()
	return s
}

func TestComputeDerived_URLs(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()
	d := ComputeDerived(cfg, sec)

	if d.DashboardURL != "https://launchpad.example.com" {
		t.Errorf("DashboardURL = %q, want %q", d.DashboardURL, "https://launchpad.example.com")
	}
	if d.GiteaURL != "https://launchpad-gitea.example.com" {
		t.Errorf("GiteaURL = %q, want %q", d.GiteaURL, "https://launchpad-gitea.example.com")
	}
	if d.APIURL != "https://launchpad.example.com:6804" {
		t.Errorf("APIURL = %q, want %q", d.APIURL, "https://launchpad.example.com:6804")
	}
	if d.ExternalDomain != "example.com" {
		t.Errorf("ExternalDomain = %q, want %q", d.ExternalDomain, "example.com")
	}
}

func TestComputeDerived_SelfSignedTLS(t *testing.T) {
	cfg := testConfig()
	cfg.SSL.Mode = "selfsigned"
	d := ComputeDerived(cfg, testSecrets())
	if d.NodeTLSReject != "0" {
		t.Errorf("NodeTLSReject = %q, want %q for selfsigned", d.NodeTLSReject, "0")
	}
}

func TestComputeDerived_LetsEncryptTLS(t *testing.T) {
	cfg := testConfig()
	cfg.SSL.Mode = "letsencrypt"
	d := ComputeDerived(cfg, testSecrets())
	if d.NodeTLSReject != "" {
		t.Errorf("NodeTLSReject = %q, want empty for letsencrypt", d.NodeTLSReject)
	}
}

func TestComputeDerived_BuiltinDBConnStrings(t *testing.T) {
	cfg := testConfig()
	cfg.Database.Mode = "builtin"
	sec := testSecrets()
	d := ComputeDerived(cfg, sec)

	schemas := []string{"main", "monitoring", "events", "billing", "stats", "gateway"}
	for _, schema := range schemas {
		if _, ok := d.DBConnStrings[schema]; !ok {
			t.Errorf("missing DBConnString for schema %q", schema)
		}
	}
}

func TestComputeDerived_ImageRefs(t *testing.T) {
	cfg := testConfig()
	d := ComputeDerived(cfg, testSecrets())

	services := []string{"api", "ui", "router", "gateway", "gitea"}
	for _, svc := range services {
		if _, ok := d.ImageRefs[svc]; !ok {
			t.Errorf("missing ImageRef for %q", svc)
		}
	}
}
