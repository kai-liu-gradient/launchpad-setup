package tabs

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type WelcomeCreditsTab struct {
	Enabled       bool
	Amount        string
	CampaignStart string
	CampaignEnd   string
	Message       string
}

func NewWelcomeCreditsTab(s *settingsmod.Settings) *WelcomeCreditsTab {
	amount := ""
	if s.WelcomeCredits.Amount != 0 {
		amount = strconv.Itoa(s.WelcomeCredits.Amount)
	}
	return &WelcomeCreditsTab{
		Enabled:       s.WelcomeCredits.Enabled,
		Amount:        amount,
		CampaignStart: s.WelcomeCredits.CampaignStart,
		CampaignEnd:   s.WelcomeCredits.CampaignEnd,
		Message:       s.WelcomeCredits.Message,
	}
}

func (t *WelcomeCreditsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Welcome Credits").
				Description("Grant credits to new users upon registration").
				Value(&t.Enabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Amount").
				Description("Number of credits to grant (integer)").
				Value(&t.Amount),
			huh.NewInput().
				Title("Campaign Start").
				Description("Optional campaign start date (e.g. 2026-01-01)").
				Value(&t.CampaignStart),
			huh.NewInput().
				Title("Campaign End").
				Description("Optional campaign end date (e.g. 2026-12-31)").
				Value(&t.CampaignEnd),
			huh.NewInput().
				Title("Message").
				Description("Message shown to users when credits are granted").
				Value(&t.Message),
		).WithHideFunc(func() bool { return !t.Enabled }),
	)
}

func (t *WelcomeCreditsTab) View() string {
	if !t.Enabled {
		return "  Status: disabled"
	}
	return fmt.Sprintf(
		"  Status:         enabled\n  Amount:         %s\n  Campaign Start: %s\n  Campaign End:   %s\n  Message:        %s",
		displayValue(t.Amount),
		displayValue(t.CampaignStart),
		displayValue(t.CampaignEnd),
		truncate(displayValue(t.Message), 40),
	)
}

func (t *WelcomeCreditsTab) Apply(s *settingsmod.Settings) {
	s.WelcomeCredits.Enabled = t.Enabled
	s.WelcomeCredits.CampaignStart = t.CampaignStart
	s.WelcomeCredits.CampaignEnd = t.CampaignEnd
	s.WelcomeCredits.Message = t.Message

	if n, err := strconv.Atoi(t.Amount); err == nil {
		s.WelcomeCredits.Amount = n
	} else {
		s.WelcomeCredits.Amount = 0
	}
}
