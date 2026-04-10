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

func TestComputeDerived_GiteaSubdomainOverride(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()

	// Default: gitea domain uses subdomain-gitea pattern
	d := ComputeDerived(cfg, sec)
	if d.GiteaDomain != "launchpad-gitea.example.com" {
		t.Errorf("default GiteaDomain = %q, want %q", d.GiteaDomain, "launchpad-gitea.example.com")
	}

	// Override: custom gitea subdomain
	cfg.GiteaSubdomain = "git"
	d = ComputeDerived(cfg, sec)
	if d.GiteaDomain != "git.example.com" {
		t.Errorf("overridden GiteaDomain = %q, want %q", d.GiteaDomain, "git.example.com")
	}
	if d.GiteaURL != "https://git.example.com" {
		t.Errorf("overridden GiteaURL = %q, want %q", d.GiteaURL, "https://git.example.com")
	}
}

func TestComputeDerived_ProjectDomain(t *testing.T) {
	cfg := testConfig()
	sec := testSecrets()

	// Default: ProjectDomain empty → IngressDomain == Domain
	d := ComputeDerived(cfg, sec)
	if d.IngressDomain != "example.com" {
		t.Errorf("default IngressDomain = %q, want %q", d.IngressDomain, "example.com")
	}
	if d.ExternalDomain != "example.com" {
		t.Errorf("default ExternalDomain = %q, want %q", d.ExternalDomain, "example.com")
	}
	if d.DomainEscaped != `example\.com` {
		t.Errorf("default DomainEscaped = %q, want %q", d.DomainEscaped, `example\.com`)
	}

	// Override: ProjectDomain set → IngressDomain uses it
	cfg.Domain = "corp.example.com"
	cfg.ProjectDomain = "example.com"
	d = ComputeDerived(cfg, sec)
	if d.IngressDomain != "example.com" {
		t.Errorf("overridden IngressDomain = %q, want %q", d.IngressDomain, "example.com")
	}
	if d.ExternalDomain != "example.com" {
		t.Errorf("overridden ExternalDomain = %q, want %q", d.ExternalDomain, "example.com")
	}
	if d.DomainEscaped != `example\.com` {
		t.Errorf("overridden DomainEscaped = %q, want %q", d.DomainEscaped, `example\.com`)
	}
	// Platform domains still use cfg.Domain
	if d.LaunchpadDomain != "launchpad.corp.example.com" {
		t.Errorf("LaunchpadDomain = %q, want %q", d.LaunchpadDomain, "launchpad.corp.example.com")
	}
}

func TestComputeDerived_ImageRefs_VersionOverride(t *testing.T) {
	cfg := testConfig()
	cfg.Images.APIVersion = "3.0.0"
	cfg.Images.GiteaVersion = "1.23-rootless"
	d := ComputeDerived(cfg, testSecrets())

	wantAPI := config.DefaultImageRegistry + "/launchpad-api:3.0.0"
	if d.ImageRefs["api"] != wantAPI {
		t.Errorf("ImageRefs[api] = %q, want %q", d.ImageRefs["api"], wantAPI)
	}

	wantGitea := "gitea/gitea:1.23-rootless"
	if d.ImageRefs["gitea"] != wantGitea {
		t.Errorf("ImageRefs[gitea] = %q, want %q", d.ImageRefs["gitea"], wantGitea)
	}

	// UI should still use compiled default
	wantUI := config.DefaultImageRegistry + "/launchpad-ui:" + config.DefaultUIVersion
	if d.ImageRefs["ui"] != wantUI {
		t.Errorf("ImageRefs[ui] = %q, want %q", d.ImageRefs["ui"], wantUI)
	}
}

func TestComputeDerived_ImageRefs_FullOverrideTakesPriority(t *testing.T) {
	cfg := testConfig()
	cfg.Images.APIVersion = "3.0.0"
	cfg.Images.API = "custom-registry.io/my-api:latest"
	d := ComputeDerived(cfg, testSecrets())

	// Full override wins over version field
	if d.ImageRefs["api"] != "custom-registry.io/my-api:latest" {
		t.Errorf("ImageRefs[api] = %q, want full override", d.ImageRefs["api"])
	}
}
