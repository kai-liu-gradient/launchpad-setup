package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type YAMLBuilderTab struct {
	AllowedUsers string
}

func NewYAMLBuilderTab(s *settingsmod.Settings) *YAMLBuilderTab {
	return &YAMLBuilderTab{
		AllowedUsers: s.YAMLBuilder.AllowedUsers,
	}
}

func (t *YAMLBuilderTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Allowed Users").
				Description("Users permitted to access the YAML builder").
				Value(&t.AllowedUsers),
		),
	)
}

func (t *YAMLBuilderTab) Edit() error { return RunFormAsEdit(t.Form()) }

func (t *YAMLBuilderTab) View() string {
	return fmt.Sprintf("  Allowed Users: %s", truncate(displayValue(t.AllowedUsers), 50))
}

func (t *YAMLBuilderTab) Apply(s *settingsmod.Settings) {
	s.YAMLBuilder.AllowedUsers = t.AllowedUsers
}
