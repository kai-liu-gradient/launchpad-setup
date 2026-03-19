package engine

import (
	"testing"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

func TestApplyChanges_BuildsRestartList(t *testing.T) {
	old := config.DefaultConfig("example.com", "admin@example.com")
	new_ := config.DefaultConfig("example.com", "admin@example.com")
	new_.SMTP.Host = "smtp.example.com"

	sec, _ := secrets.Generate()
	events := make(chan StepEvent, 100)
	e := New(old, sec, t.TempDir(), events)

	steps := e.BuildReconfigureSteps(old, new_)
	if len(steps) == 0 {
		t.Error("SMTP change should produce reconfigure steps")
	}

	// Should include re-render and api restart
	hasRender := false
	hasRestart := false
	for _, s := range steps {
		if s.Name == "Re-rendering templates" {
			hasRender = true
		}
		if s.Name == "Restarting api" {
			hasRestart = true
		}
	}
	if !hasRender {
		t.Error("should include re-render step")
	}
	if !hasRestart {
		t.Error("should include api restart step")
	}
}

func TestApplyChanges_NoChanges(t *testing.T) {
	old := config.DefaultConfig("example.com", "admin@example.com")
	new_ := config.DefaultConfig("example.com", "admin@example.com")

	sec, _ := secrets.Generate()
	e := New(old, sec, t.TempDir(), nil)

	steps := e.BuildReconfigureSteps(old, new_)
	if len(steps) != 0 {
		t.Errorf("no changes should produce 0 steps, got %d", len(steps))
	}
}

func TestApplyChanges_SSLChange(t *testing.T) {
	old := config.DefaultConfig("example.com", "admin@example.com")
	new_ := config.DefaultConfig("example.com", "admin@example.com")
	new_.SSL.Mode = "letsencrypt"

	sec, _ := secrets.Generate()
	e := New(old, sec, t.TempDir(), nil)

	steps := e.BuildReconfigureSteps(old, new_)

	// SSL changes should restart nginx + other services
	hasNginx := false
	for _, s := range steps {
		if s.Name == "Restarting nginx" {
			hasNginx = true
		}
	}
	if !hasNginx {
		t.Error("SSL change should restart nginx")
	}
}
