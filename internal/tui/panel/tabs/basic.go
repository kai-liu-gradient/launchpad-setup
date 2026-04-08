package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type BasicTab struct {
	Domain        string
	ProjectDomain string
	Registry      string
}

func NewBasicTab(cfg *config.Config) *BasicTab {
	registry := cfg.Images.Registry
	if registry == "" {
		registry = config.DefaultImageRegistry
	}
	return &BasicTab{
		Domain:        cfg.Domain,
		ProjectDomain: cfg.ProjectDomain,
		Registry:      registry,
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
			huh.NewInput().
				Title("Image registry").
				Placeholder("swr.ap-southeast-1.myhuaweicloud.com/ghisha").
				Value(&t.Registry),
		),
	)
}

func (t *BasicTab) View() string {
	pd := displayValue(t.ProjectDomain)
	if t.ProjectDomain == "" {
		pd = "(same as domain)"
	}
	return fmt.Sprintf("  Domain:          %s\n  Project Domain:  %s\n  Registry:        %s",
		displayValue(t.Domain), pd, displayValue(t.Registry))
}

func (t *BasicTab) Apply(cfg *config.Config) {
	cfg.Domain = t.Domain
	cfg.ProjectDomain = t.ProjectDomain
	cfg.AdminEmail = "admin@" + t.Domain
	cfg.Images.Registry = t.Registry
}

func (t *BasicTab) Edit() error {
	return RunFormAsEdit(t.Form())
}
