package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type BasicTab struct {
	Domain        string
	ProjectDomain string
}

func NewBasicTab(cfg *config.Config) *BasicTab {
	return &BasicTab{
		Domain:        cfg.Domain,
		ProjectDomain: cfg.ProjectDomain,
	}
}

func (t *BasicTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain name").
				Placeholder("example.com").
				Value(&t.Domain),
			huh.NewInput().
				Title("Project Domain (optional)").
				Description("Wildcard domain for project pods. Leave empty to use Domain.").
				Placeholder("example.com").
				Value(&t.ProjectDomain),
		),
	)
}

func (t *BasicTab) View() string {
	pd := displayValue(t.ProjectDomain)
	if t.ProjectDomain == "" {
		pd = "(same as domain)"
	}
	return fmt.Sprintf("  Domain:          %s\n  Project Domain:  %s",
		displayValue(t.Domain), pd)
}

func (t *BasicTab) Apply(cfg *config.Config) {
	cfg.Domain = t.Domain
	cfg.ProjectDomain = t.ProjectDomain
	cfg.AdminEmail = "admin@" + t.Domain
}

func (t *BasicTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
