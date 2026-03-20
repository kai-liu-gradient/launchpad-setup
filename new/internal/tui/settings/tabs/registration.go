package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type RegistrationTab struct {
	RestrictDomain           bool
	AllowedDomains           string
	RequireEmailVerification bool
	Notice                   string
	BlockedNotice            string
}

func NewRegistrationTab(s *settingsmod.Settings) *RegistrationTab {
	return &RegistrationTab{
		RestrictDomain:           s.Registration.RestrictDomain,
		AllowedDomains:           strings.Join(s.Registration.AllowedDomains, ","),
		RequireEmailVerification: s.Registration.RequireEmailVerification,
		Notice:                   s.Registration.Notice,
		BlockedNotice:            s.Registration.BlockedNotice,
	}
}

func (t *RegistrationTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Restrict Domain").
				Description("Only allow sign-ups from specific email domains").
				Value(&t.RestrictDomain),
			huh.NewConfirm().
				Title("Require Email Verification").
				Description("Users must verify their email before accessing the platform").
				Value(&t.RequireEmailVerification),
			huh.NewInput().
				Title("Notice").
				Description("Message shown on the registration page").
				Value(&t.Notice),
			huh.NewInput().
				Title("Blocked Notice").
				Description("Message shown to blocked registrations").
				Value(&t.BlockedNotice),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Allowed Domains").
				Description("Comma-separated list of allowed email domains (e.g. example.com,corp.io)").
				Value(&t.AllowedDomains),
		).WithHideFunc(func() bool { return !t.RestrictDomain }),
	)
}

func (t *RegistrationTab) View() string {
	return fmt.Sprintf(
		"  Restrict Domain:            %s\n  Allowed Domains:            %s\n  Require Email Verification: %s\n  Notice:                     %s\n  Blocked Notice:             %s",
		boolDisplay(t.RestrictDomain),
		displayValue(t.AllowedDomains),
		boolDisplay(t.RequireEmailVerification),
		truncate(displayValue(t.Notice), 40),
		truncate(displayValue(t.BlockedNotice), 40),
	)
}

func (t *RegistrationTab) Apply(s *settingsmod.Settings) {
	s.Registration.RestrictDomain = t.RestrictDomain
	s.Registration.RequireEmailVerification = t.RequireEmailVerification
	s.Registration.Notice = t.Notice
	s.Registration.BlockedNotice = t.BlockedNotice

	if t.AllowedDomains == "" {
		s.Registration.AllowedDomains = nil
	} else {
		parts := strings.Split(t.AllowedDomains, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.Registration.AllowedDomains = result
	}
}
