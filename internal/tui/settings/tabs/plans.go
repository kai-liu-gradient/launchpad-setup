package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type PlansTab struct {
	EnforceFreePlan bool
	Disabled        []string
	DisabledNotice  string
}

func NewPlansTab(s *settingsmod.Settings) *PlansTab {
	disabled := make([]string, len(s.Plans.Disabled))
	copy(disabled, s.Plans.Disabled)
	return &PlansTab{
		EnforceFreePlan: s.Plans.EnforceFreePlan,
		Disabled:        disabled,
		DisabledNotice:  s.Plans.DisabledNotice,
	}
}

func (t *PlansTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enforce Free Plan").
				Description("Force all users onto the free plan").
				Value(&t.EnforceFreePlan),
			huh.NewMultiSelect[string]().
				Title("Disabled Plans").
				Description("Select plans to disable").
				Options(
					huh.NewOption("Free", "free"),
					huh.NewOption("Pro", "pro"),
					huh.NewOption("Team", "team"),
					huh.NewOption("Enterprise", "enterprise"),
				).
				Filterable(true).
				Value(&t.Disabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when a plan is disabled").
				Value(&t.DisabledNotice),
		),
	)
}

func (t *PlansTab) Edit() error { return RunFormAsEdit(t.Form()) }

func (t *PlansTab) View() string {
	return fmt.Sprintf(
		"  Enforce Free Plan: %s\n  Disabled Plans:    %s\n  Disabled Notice:   %s",
		boolDisplay(t.EnforceFreePlan),
		listDisplay(t.Disabled),
		truncate(displayValue(t.DisabledNotice), 40),
	)
}

func (t *PlansTab) Apply(s *settingsmod.Settings) {
	s.Plans.EnforceFreePlan = t.EnforceFreePlan
	s.Plans.DisabledNotice = t.DisabledNotice
	s.Plans.Disabled = t.Disabled
}
