package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type AuthenticationTab struct {
	Mode string

	PasswordEnabled        bool
	PasswordEnableSignup   bool
	PasswordDisabledNotice string

	GoogleEnabled    bool
	GoogleAllowSignup bool

	GitHubEnabled    bool
	GitHubAllowSignup bool

	MicrosoftEnabled     bool
	MicrosoftAllowSignup bool
	MicrosoftEnforceOnly bool

	ProviderSignup bool
	SSOOnlyNotice  string
}

func NewAuthenticationTab(s *settingsmod.Settings) *AuthenticationTab {
	return &AuthenticationTab{
		Mode:                   s.Authentication.Mode,
		PasswordEnabled:        s.Authentication.PasswordLogin.Enabled,
		PasswordEnableSignup:   s.Authentication.PasswordLogin.EnableSignup,
		PasswordDisabledNotice: s.Authentication.PasswordLogin.DisabledNotice,
		GoogleEnabled:          s.Authentication.OAuth.Google.Enabled,
		GoogleAllowSignup:      s.Authentication.OAuth.Google.AllowSignup,
		GitHubEnabled:          s.Authentication.OAuth.GitHub.Enabled,
		GitHubAllowSignup:      s.Authentication.OAuth.GitHub.AllowSignup,
		MicrosoftEnabled:       s.Authentication.OAuth.Microsoft.Enabled,
		MicrosoftAllowSignup:   s.Authentication.OAuth.Microsoft.AllowSignup,
		MicrosoftEnforceOnly:   s.Authentication.OAuth.Microsoft.EnforceOnly,
		ProviderSignup:         s.Authentication.ProviderSignup,
		SSOOnlyNotice:          s.Authentication.SSOOnlyNotice,
	}
}

func (t *AuthenticationTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Authentication Mode").
				Options(
					huh.NewOption("Standard", "standard"),
					huh.NewOption("SSO Only", "sso"),
				).
				Value(&t.Mode),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Provider Signup").
				Description("Allow OAuth providers to create new accounts").
				Value(&t.ProviderSignup),
			huh.NewInput().
				Title("SSO Only Notice").
				Description("Message shown when SSO-only mode is active").
				Value(&t.SSOOnlyNotice),
		),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Password Login — Enabled").
				Value(&t.PasswordEnabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Password Login — Enable Signup").
				Value(&t.PasswordEnableSignup),
			huh.NewInput().
				Title("Password Login — Disabled Notice").
				Value(&t.PasswordDisabledNotice),
		),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Google — Enabled").
				Value(&t.GoogleEnabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Google — Allow Signup").
				Value(&t.GoogleAllowSignup),
		),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("GitHub — Enabled").
				Value(&t.GitHubEnabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("GitHub — Allow Signup").
				Value(&t.GitHubAllowSignup),
		),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Microsoft — Enabled").
				Value(&t.MicrosoftEnabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Microsoft — Allow Signup").
				Value(&t.MicrosoftAllowSignup),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Microsoft — Enforce Only").
				Value(&t.MicrosoftEnforceOnly),
		),
	)
}

func (t *AuthenticationTab) View() string {
	providers := ""
	if t.GoogleEnabled {
		providers += " Google"
	}
	if t.GitHubEnabled {
		providers += " GitHub"
	}
	if t.MicrosoftEnabled {
		providers += " Microsoft"
	}
	if providers == "" {
		providers = " (none)"
	}

	return fmt.Sprintf(
		"  Mode:           %s\n  Password Login: %s\n  Providers:     %s\n  Provider Signup: %s",
		displayValue(t.Mode),
		boolDisplay(t.PasswordEnabled),
		providers,
		boolDisplay(t.ProviderSignup),
	)
}

func (t *AuthenticationTab) Apply(s *settingsmod.Settings) {
	s.Authentication.Mode = t.Mode
	s.Authentication.PasswordLogin = settingsmod.PasswordLogin{
		Enabled:        t.PasswordEnabled,
		EnableSignup:   t.PasswordEnableSignup,
		DisabledNotice: t.PasswordDisabledNotice,
	}
	s.Authentication.OAuth.Google = settingsmod.OAuthProvider{
		Enabled:     t.GoogleEnabled,
		AllowSignup: t.GoogleAllowSignup,
	}
	s.Authentication.OAuth.GitHub = settingsmod.OAuthProvider{
		Enabled:     t.GitHubEnabled,
		AllowSignup: t.GitHubAllowSignup,
	}
	s.Authentication.OAuth.Microsoft = settingsmod.MicrosoftOAuth{
		Enabled:     t.MicrosoftEnabled,
		AllowSignup: t.MicrosoftAllowSignup,
		EnforceOnly: t.MicrosoftEnforceOnly,
	}
	s.Authentication.ProviderSignup = t.ProviderSignup
	s.Authentication.SSOOnlyNotice = t.SSOOnlyNotice
}
