package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type CredentialsTab struct {
	ClaudeIncludedDisabled bool
	ClaudeIncludedNotice   string
	ZAIIncludedDisabled    bool
	ZAIIncludedNotice      string
	ClaudePaygoDisabled    bool
	ClaudePaygoNotice      string
	ZAIPaygoDisabled       bool
	ZAIPaygoNotice         string
	GeminiPaygoDisabled    bool
	GeminiPaygoNotice      string
}

func NewCredentialsTab(s *settingsmod.Settings) *CredentialsTab {
	return &CredentialsTab{
		ClaudeIncludedDisabled: s.Credentials.ClaudeIncluded.Disabled,
		ClaudeIncludedNotice:   s.Credentials.ClaudeIncluded.Notice,
		ZAIIncludedDisabled:    s.Credentials.ZAIIncluded.Disabled,
		ZAIIncludedNotice:      s.Credentials.ZAIIncluded.Notice,
		ClaudePaygoDisabled:    s.Credentials.ClaudePaygo.Disabled,
		ClaudePaygoNotice:      s.Credentials.ClaudePaygo.Notice,
		ZAIPaygoDisabled:       s.Credentials.ZAIPaygo.Disabled,
		ZAIPaygoNotice:         s.Credentials.ZAIPaygo.Notice,
		GeminiPaygoDisabled:    s.Credentials.GeminiPaygo.Disabled,
		GeminiPaygoNotice:      s.Credentials.GeminiPaygo.Notice,
	}
}

func credStatus(disabled bool) string {
	if disabled {
		return "disabled"
	}
	return "active"
}

func (t *CredentialsTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable Claude Included").
				Value(&t.ClaudeIncludedDisabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable ZAI Included").
				Value(&t.ZAIIncludedDisabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable Claude PayGo").
				Value(&t.ClaudePaygoDisabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable ZAI PayGo").
				Value(&t.ZAIPaygoDisabled),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Disable Gemini PayGo").
				Value(&t.GeminiPaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("Claude Included — Disabled Notice").Value(&t.ClaudeIncludedNotice),
			huh.NewInput().Title("ZAI Included — Disabled Notice").Value(&t.ZAIIncludedNotice),
			huh.NewInput().Title("Claude PayGo — Disabled Notice").Value(&t.ClaudePaygoNotice),
			huh.NewInput().Title("ZAI PayGo — Disabled Notice").Value(&t.ZAIPaygoNotice),
			huh.NewInput().Title("Gemini PayGo — Disabled Notice").Value(&t.GeminiPaygoNotice),
		).WithHideFunc(func() bool {
			return !t.ClaudeIncludedDisabled && !t.ZAIIncludedDisabled &&
				!t.ClaudePaygoDisabled && !t.ZAIPaygoDisabled && !t.GeminiPaygoDisabled
		}),
	)
}

func (t *CredentialsTab) View() string {
	return fmt.Sprintf(
		"  Claude Included: %s\n  ZAI Included:    %s\n  Claude Paygo:    %s\n  ZAI Paygo:       %s\n  Gemini Paygo:    %s",
		credStatus(t.ClaudeIncludedDisabled),
		credStatus(t.ZAIIncludedDisabled),
		credStatus(t.ClaudePaygoDisabled),
		credStatus(t.ZAIPaygoDisabled),
		credStatus(t.GeminiPaygoDisabled),
	)
}

func (t *CredentialsTab) Apply(s *settingsmod.Settings) {
	s.Credentials.ClaudeIncluded = settingsmod.CredentialToggle{
		Disabled: t.ClaudeIncludedDisabled,
		Notice:   t.ClaudeIncludedNotice,
	}
	s.Credentials.ZAIIncluded = settingsmod.CredentialToggle{
		Disabled: t.ZAIIncludedDisabled,
		Notice:   t.ZAIIncludedNotice,
	}
	s.Credentials.ClaudePaygo = settingsmod.CredentialToggle{
		Disabled: t.ClaudePaygoDisabled,
		Notice:   t.ClaudePaygoNotice,
	}
	s.Credentials.ZAIPaygo = settingsmod.CredentialToggle{
		Disabled: t.ZAIPaygoDisabled,
		Notice:   t.ZAIPaygoNotice,
	}
	s.Credentials.GeminiPaygo = settingsmod.CredentialToggle{
		Disabled: t.GeminiPaygoDisabled,
		Notice:   t.GeminiPaygoNotice,
	}
}
