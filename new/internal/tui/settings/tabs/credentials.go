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
				Title("Claude Included — Disable").
				Description("Disable the Claude Included credential").
				Value(&t.ClaudeIncludedDisabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Claude Included — Notice").
				Description("Message shown when Claude Included is disabled").
				Value(&t.ClaudeIncludedNotice),
		).WithHideFunc(func() bool { return !t.ClaudeIncludedDisabled }),

		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("ZAI Included — Disable").
				Description("Disable the ZAI Included credential").
				Value(&t.ZAIIncludedDisabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("ZAI Included — Notice").
				Description("Message shown when ZAI Included is disabled").
				Value(&t.ZAIIncludedNotice),
		).WithHideFunc(func() bool { return !t.ZAIIncludedDisabled }),

		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Claude Paygo — Disable").
				Description("Disable the Claude Paygo credential").
				Value(&t.ClaudePaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Claude Paygo — Notice").
				Description("Message shown when Claude Paygo is disabled").
				Value(&t.ClaudePaygoNotice),
		).WithHideFunc(func() bool { return !t.ClaudePaygoDisabled }),

		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("ZAI Paygo — Disable").
				Description("Disable the ZAI Paygo credential").
				Value(&t.ZAIPaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("ZAI Paygo — Notice").
				Description("Message shown when ZAI Paygo is disabled").
				Value(&t.ZAIPaygoNotice),
		).WithHideFunc(func() bool { return !t.ZAIPaygoDisabled }),

		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Gemini Paygo — Disable").
				Description("Disable the Gemini Paygo credential").
				Value(&t.GeminiPaygoDisabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Gemini Paygo — Notice").
				Description("Message shown when Gemini Paygo is disabled").
				Value(&t.GeminiPaygoNotice),
		).WithHideFunc(func() bool { return !t.GeminiPaygoDisabled }),
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
