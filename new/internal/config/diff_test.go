package config

import "testing"

func TestDiff_NoChanges(t *testing.T) {
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
	d := Diff(a, b)
	if len(d.Changes) != 0 {
		t.Errorf("expected no changes, got %d", len(d.Changes))
	}
}

func TestDiff_SSLModeChanged(t *testing.T) {
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
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
	a := DefaultConfig("example.com", "admin@example.com")
	b := DefaultConfig("example.com", "admin@example.com")
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
