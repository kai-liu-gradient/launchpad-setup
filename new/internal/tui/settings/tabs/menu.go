package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type MenuTab struct {
	DocsVisible bool
}

func NewMenuTab(s *settingsmod.Settings) *MenuTab {
	return &MenuTab{
		DocsVisible: s.Menu.Docs.Visible,
	}
}

func (t *MenuTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Docs Visible").
				Description("Show the Docs link in the navigation menu").
				Value(&t.DocsVisible),
		),
	)
}

func (t *MenuTab) View() string {
	return fmt.Sprintf("  Docs Visible: %s", boolDisplay(t.DocsVisible))
}

func (t *MenuTab) Apply(s *settingsmod.Settings) {
	s.Menu.Docs.Visible = t.DocsVisible
}
