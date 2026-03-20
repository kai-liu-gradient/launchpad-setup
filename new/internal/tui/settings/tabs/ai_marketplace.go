package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type AIMarketplaceTab struct {
	Visible          bool
	EnabledProviders string
}

func NewAIMarketplaceTab(s *settingsmod.Settings) *AIMarketplaceTab {
	return &AIMarketplaceTab{
		Visible:          s.AIMarketplace.Visible,
		EnabledProviders: strings.Join(s.AIMarketplace.EnabledProviders, ","),
	}
}

func (t *AIMarketplaceTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Visible").
				Description("Show the AI Marketplace to users").
				Value(&t.Visible),
			huh.NewInput().
				Title("Enabled Providers").
				Description("Comma-separated list of enabled AI provider IDs").
				Value(&t.EnabledProviders),
		),
	)
}

func (t *AIMarketplaceTab) View() string {
	return fmt.Sprintf(
		"  Visible:           %s\n  Enabled Providers: %s",
		boolDisplay(t.Visible),
		displayValue(t.EnabledProviders),
	)
}

func (t *AIMarketplaceTab) Apply(s *settingsmod.Settings) {
	s.AIMarketplace.Visible = t.Visible

	if t.EnabledProviders == "" {
		s.AIMarketplace.EnabledProviders = nil
	} else {
		parts := strings.Split(t.EnabledProviders, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.AIMarketplace.EnabledProviders = result
	}
}
