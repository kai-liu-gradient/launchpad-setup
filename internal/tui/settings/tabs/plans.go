package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type PlansTab struct {
	EnforceFreePlan bool
	Disabled        string
	DisabledNotice  string
}

func NewPlansTab(s *settingsmod.Settings) *PlansTab {
	return &PlansTab{
		EnforceFreePlan: s.Plans.EnforceFreePlan,
		Disabled:        strings.Join(s.Plans.Disabled, ","),
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
			huh.NewInput().
				Title("Disabled Plans").
				Description("Comma-separated list of plan IDs to disable").
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
		displayValue(t.Disabled),
		truncate(displayValue(t.DisabledNotice), 40),
	)
}

func (t *PlansTab) Apply(s *settingsmod.Settings) {
	s.Plans.EnforceFreePlan = t.EnforceFreePlan
	s.Plans.DisabledNotice = t.DisabledNotice

	if t.Disabled == "" {
		s.Plans.Disabled = nil
	} else {
		parts := strings.Split(t.Disabled, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.Plans.Disabled = result
	}
}
