package tabs

import (
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type AITab struct {
	CRS2Endpoint string
	CRS2Token    string
}

func NewAITab(cfg *config.Config) *AITab {
	return &AITab{
		CRS2Endpoint: cfg.AI.CRS2Endpoint,
		CRS2Token:    cfg.AI.CRS2Token,
	}
}

func (t *AITab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("CRS2 Endpoint").
				Description("AI service endpoint URL").
				Value(&t.CRS2Endpoint),
			huh.NewInput().
				Title("CRS2 Token").
				Description("Authentication token for the AI service").
				EchoMode(huh.EchoModePassword).
				Value(&t.CRS2Token),
		),
	)
}

func (t *AITab) Apply(cfg *config.Config) {
	cfg.AI.CRS2Endpoint = t.CRS2Endpoint
	cfg.AI.CRS2Token = t.CRS2Token
}
