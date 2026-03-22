package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type BasicTab struct {
	Domain   string
	Registry string
}

func NewBasicTab(cfg *config.Config) *BasicTab {
	registry := cfg.Images.Registry
	if registry == "" {
		registry = config.DefaultImageRegistry
	}
	return &BasicTab{
		Domain:   cfg.Domain,
		Registry: registry,
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
				Title("Image registry").
				Placeholder("swr.ap-southeast-1.myhuaweicloud.com/ghisha").
				Value(&t.Registry),
		),
	)
}

func (t *BasicTab) View() string {
	return fmt.Sprintf("  Domain:   %s\n  Registry: %s",
		displayValue(t.Domain), displayValue(t.Registry))
}

func (t *BasicTab) Apply(cfg *config.Config) {
	cfg.Domain = t.Domain
	cfg.AdminEmail = "admin@" + t.Domain
	cfg.Images.Registry = t.Registry
}
