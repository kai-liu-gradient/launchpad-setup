package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type EmailSuppressionTab struct {
	SuppressedRecipients string
}

func NewEmailSuppressionTab(s *settingsmod.Settings) *EmailSuppressionTab {
	return &EmailSuppressionTab{
		SuppressedRecipients: strings.Join(s.EmailSuppression.SuppressedRecipients, ","),
	}
}

func (t *EmailSuppressionTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Suppressed Recipients").
				Description("Comma-separated list of email addresses to suppress").
				Value(&t.SuppressedRecipients),
		),
	)
}

func (t *EmailSuppressionTab) View() string {
	return fmt.Sprintf("  Suppressed Recipients: %s", truncate(displayValue(t.SuppressedRecipients), 50))
}

func (t *EmailSuppressionTab) Apply(s *settingsmod.Settings) {
	if t.SuppressedRecipients == "" {
		s.EmailSuppression.SuppressedRecipients = nil
	} else {
		parts := strings.Split(t.SuppressedRecipients, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.EmailSuppression.SuppressedRecipients = result
	}
}
