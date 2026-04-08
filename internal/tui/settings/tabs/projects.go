package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type ProjectsTab struct {
	CreateDisabled       bool
	CreateDisabledNotice string
}

func NewProjectsTab(s *settingsmod.Settings) *ProjectsTab {
	return &ProjectsTab{
		CreateDisabled:       s.Projects.CreateDisabled,
		CreateDisabledNotice: s.Projects.CreateDisabledNotice,
	}
}

func (t *ProjectsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable Project Creation").
				Description("Prevent users from creating new projects").
				Value(&t.CreateDisabled),
			huh.NewInput().
				Title("Disabled Notice").
				Description("Message shown when project creation is disabled").
				Value(&t.CreateDisabledNotice),
		),
	)
}

func (t *ProjectsTab) Edit() error { return RunFormAsEdit(t.Form()) }

func (t *ProjectsTab) View() string {
	return fmt.Sprintf(
		"  Create Disabled: %s\n  Notice:          %s",
		boolDisplay(t.CreateDisabled),
		truncate(displayValue(t.CreateDisabledNotice), 40),
	)
}

func (t *ProjectsTab) Apply(s *settingsmod.Settings) {
	s.Projects.CreateDisabled = t.CreateDisabled
	s.Projects.CreateDisabledNotice = t.CreateDisabledNotice
}
