package tabs

import (
	"fmt"

	settingsmod "github.com/gradient8/launchpad/internal/settings"
	"github.com/gradient8/launchpad/internal/tui/components"
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
	ZAIGhishaHidden        bool
	ZAIGhishaDisabled      bool
	ZAIGhishaNotice        string
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
		ZAIGhishaHidden:        s.Credentials.ZAIGhisha.Hidden,
		ZAIGhishaDisabled:      s.Credentials.ZAIGhisha.Disabled,
		ZAIGhishaNotice:        s.Credentials.ZAIGhisha.Notice,
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

func (t *CredentialsTab) Edit() error {
	nodes := []components.TreeNode{
		{
			Label: "Claude Included",
			OnToggle: func() {
				t.ClaudeIncludedDisabled = !t.ClaudeIncludedDisabled
			},
			Status: func() string {
				return credStatus(t.ClaudeIncludedDisabled)
			},
			Expanded: t.ClaudeIncludedDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ClaudeIncludedNotice},
			},
		},
		{
			Label: "ZAI Included",
			OnToggle: func() {
				t.ZAIIncludedDisabled = !t.ZAIIncludedDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIIncludedDisabled)
			},
			Expanded: t.ZAIIncludedDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIIncludedNotice},
			},
		},
		{
			Label: "Claude PayGo",
			OnToggle: func() {
				t.ClaudePaygoDisabled = !t.ClaudePaygoDisabled
			},
			Status: func() string {
				return credStatus(t.ClaudePaygoDisabled)
			},
			Expanded: t.ClaudePaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ClaudePaygoNotice},
			},
		},
		{
			Label: "ZAI PayGo",
			OnToggle: func() {
				t.ZAIPaygoDisabled = !t.ZAIPaygoDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIPaygoDisabled)
			},
			Expanded: t.ZAIPaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIPaygoNotice},
			},
		},
		{
			Label: "ZAI Ghisha",
			OnToggle: func() {
				t.ZAIGhishaDisabled = !t.ZAIGhishaDisabled
			},
			Status: func() string {
				return credStatus(t.ZAIGhishaDisabled)
			},
			Expanded: t.ZAIGhishaDisabled,
			Fields: []components.TreeField{
				{Label: "Hidden", Type: components.FieldToggle, BoolValue: &t.ZAIGhishaHidden},
				{Label: "Notice", Type: components.FieldText, TextValue: &t.ZAIGhishaNotice},
			},
		},
		{
			Label: "Gemini PayGo",
			OnToggle: func() {
				t.GeminiPaygoDisabled = !t.GeminiPaygoDisabled
			},
			Status: func() string {
				return credStatus(t.GeminiPaygoDisabled)
			},
			Expanded: t.GeminiPaygoDisabled,
			Fields: []components.TreeField{
				{Label: "Notice", Type: components.FieldText, TextValue: &t.GeminiPaygoNotice},
			},
		},
	}

	return components.NewTreeForm("Credentials", nodes).Run()
}

func (t *CredentialsTab) View() string {
	return fmt.Sprintf(
		"  Claude Included: %s\n  ZAI Included:    %s\n  Claude Paygo:    %s\n  ZAI Paygo:       %s\n  ZAI Ghisha:      %s\n  Gemini Paygo:    %s",
		credStatus(t.ClaudeIncludedDisabled),
		credStatus(t.ZAIIncludedDisabled),
		credStatus(t.ClaudePaygoDisabled),
		credStatus(t.ZAIPaygoDisabled),
		credStatus(t.ZAIGhishaDisabled),
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
	s.Credentials.ZAIGhisha = settingsmod.CredentialToggle{
		Hidden:   t.ZAIGhishaHidden,
		Disabled: t.ZAIGhishaDisabled,
		Notice:   t.ZAIGhishaNotice,
	}
	s.Credentials.GeminiPaygo = settingsmod.CredentialToggle{
		Disabled: t.GeminiPaygoDisabled,
		Notice:   t.GeminiPaygoNotice,
	}
}
