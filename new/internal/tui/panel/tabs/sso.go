package tabs

import (
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type SSOTab struct {
	EntraTenantID string
	EntraClientID string
	EntraSecret   string
}

func NewSSOTab(cfg *config.Config) *SSOTab {
	return &SSOTab{
		EntraTenantID: cfg.SSO.EntraTenantID,
		EntraClientID: cfg.SSO.EntraClientID,
		EntraSecret:   cfg.SSO.EntraSecret,
	}
}

func (t *SSOTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Entra Tenant ID").
				Description("Microsoft Entra ID (Azure AD) tenant identifier").
				Value(&t.EntraTenantID),
			huh.NewInput().
				Title("Entra Client ID").
				Description("Application (client) ID registered in Entra").
				Value(&t.EntraClientID),
			huh.NewInput().
				Title("Entra Client Secret").
				EchoMode(huh.EchoModePassword).
				Value(&t.EntraSecret),
		),
	)
}

func (t *SSOTab) Apply(cfg *config.Config) {
	cfg.SSO.EntraTenantID = t.EntraTenantID
	cfg.SSO.EntraClientID = t.EntraClientID
	cfg.SSO.EntraSecret = t.EntraSecret
}
