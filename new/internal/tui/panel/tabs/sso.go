package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/config"
)

type SSOTab struct {
	Enabled         bool
	Provider        string
	EntraTenantID   string
	EntraClientID   string
	EntraSecret     string
	RedirectURL     string
	EntraButtonIcon string
	EntraButtonText string
}

func NewSSOTab(cfg *config.Config) *SSOTab {
	provider := cfg.SSO.Provider
	if provider == "" {
		provider = "entra"
	}
	buttonIcon := cfg.SSO.EntraButtonIcon
	if buttonIcon == "" && provider == "entra" {
		buttonIcon = "/svg/svg-microsoft.svg"
	}
	buttonText := cfg.SSO.EntraButtonText
	if buttonText == "" && provider == "entra" {
		buttonText = "Sign in with Microsoft"
	}
	return &SSOTab{
		Enabled:         cfg.SSO.Enabled,
		Provider:        provider,
		EntraTenantID:   cfg.SSO.EntraTenantID,
		EntraClientID:   cfg.SSO.EntraClientID,
		EntraSecret:     cfg.SSO.EntraSecret,
		RedirectURL:     cfg.SSO.RedirectURL,
		EntraButtonIcon: buttonIcon,
		EntraButtonText: buttonText,
	}
}

func (t *SSOTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enable SSO").
				Value(&t.Enabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSO Provider").
				Options(
					huh.NewOption("Microsoft Entra ID", "entra"),
				).
				Value(&t.Provider),
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
			huh.NewInput().
				Title("Redirect URL").
				Description("Override auto-generated callback URL (optional)").
				Value(&t.RedirectURL),
			huh.NewInput().
				Title("Button Icon URL").
				Description("Custom SSO button icon (optional)").
				Value(&t.EntraButtonIcon),
			huh.NewInput().
				Title("Button Text").
				Description("Custom SSO button label (optional, e.g. \"Sign in with Contoso\")").
				Value(&t.EntraButtonText),
		).WithHideFunc(func() bool { return !t.Enabled }),
	)
}

func (t *SSOTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enable SSO").
				Value(&t.Enabled),
		).Title("SSO"),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSO Provider").
				Options(
					huh.NewOption("Microsoft Entra ID", "entra"),
				).
				Value(&t.Provider),
			huh.NewInput().
				Title("Entra Tenant ID").
				Value(&t.EntraTenantID),
			huh.NewInput().
				Title("Entra Client ID").
				Value(&t.EntraClientID),
			huh.NewInput().
				Title("Entra Client Secret").
				EchoMode(huh.EchoModePassword).
				Value(&t.EntraSecret),
			huh.NewInput().
				Title("Redirect URL").
				Value(&t.RedirectURL),
			huh.NewInput().
				Title("Button Icon URL").
				Value(&t.EntraButtonIcon),
			huh.NewInput().
				Title("Button Text").
				Value(&t.EntraButtonText),
		).Title("SSO — Entra ID").WithHideFunc(func() bool { return !t.Enabled }),
	}
}

func (t *SSOTab) View() string {
	if !t.Enabled {
		return "  SSO: disabled"
	}
	s := fmt.Sprintf("  SSO:           enabled\n  Provider:      %s", t.Provider)
	s += fmt.Sprintf("\n  Tenant ID:     %s\n  Client ID:     %s\n  Client Secret: %s",
		displayValue(t.EntraTenantID), displayValue(t.EntraClientID), maskValue(t.EntraSecret))
	if t.RedirectURL != "" {
		s += fmt.Sprintf("\n  Redirect URL:  %s", t.RedirectURL)
	}
	if t.EntraButtonIcon != "" {
		s += fmt.Sprintf("\n  Button Icon:   %s", t.EntraButtonIcon)
	}
	if t.EntraButtonText != "" {
		s += fmt.Sprintf("\n  Button Text:   %s", t.EntraButtonText)
	}
	return s
}

func (t *SSOTab) Apply(cfg *config.Config) {
	cfg.SSO.Enabled = t.Enabled
	cfg.SSO.Provider = t.Provider
	cfg.SSO.EntraTenantID = t.EntraTenantID
	cfg.SSO.EntraClientID = t.EntraClientID
	cfg.SSO.EntraSecret = t.EntraSecret
	cfg.SSO.RedirectURL = t.RedirectURL
	cfg.SSO.EntraButtonIcon = t.EntraButtonIcon
	cfg.SSO.EntraButtonText = t.EntraButtonText
}
