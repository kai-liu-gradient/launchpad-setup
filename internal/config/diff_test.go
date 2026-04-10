package config

import "testing"

func TestDiff_NoChanges(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	d := Diff(a, b)
	if len(d.Changes) != 0 {
		t.Errorf("expected no changes, got %d", len(d.Changes))
	}
}

func TestDiff_SSLModeChanged(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	b.SSL.Mode = "letsencrypt"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Error("expected changes for SSL mode")
	}
	services := d.AffectedServices()
	// SSL change affects nginx and possibly all services
	found := false
	for _, s := range services {
		if s == "nginx" {
			found = true
		}
	}
	if !found {
		t.Error("SSL change should affect nginx")
	}
}

func TestDiff_SMTPAdded(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	b.SMTP.Host = "smtp.example.com"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Error("expected changes for SMTP")
	}
	services := d.AffectedServices()
	found := false
	for _, s := range services {
		if s == "api" {
			found = true
		}
	}
	if !found {
		t.Error("SMTP change should affect api")
	}
}

func TestDiff_ImageVersionChanged(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	b.Images.APIVersion = "3.0.0"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Fatal("expected changes for image version")
	}
	services := d.AffectedServices()
	found := false
	for _, s := range services {
		if s == "api" {
			found = true
		}
	}
	if !found {
		t.Error("APIVersion change should affect api service")
	}
}

func TestDiff_GiteaVersionChanged(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	b.Images.GiteaVersion = "1.23-rootless"
	d := Diff(a, b)
	if len(d.Changes) == 0 {
		t.Fatal("expected changes for gitea version")
	}
	services := d.AffectedServices()
	found := false
	for _, s := range services {
		if s == "gitea" {
			found = true
		}
	}
	if !found {
		t.Error("GiteaVersion change should affect gitea service")
	}
}

func TestDiff_ImageVersionOnlyAffectsTargetService(t *testing.T) {
	a := DefaultConfig("example.com")
	b := DefaultConfig("example.com")
	b.Images.UIVersion = "9.9.9"
	d := Diff(a, b)
	services := d.AffectedServices()
	if len(services) != 1 || services[0] != "ui" {
		t.Errorf("UIVersion change should only affect ui, got %v", services)
	}
}
