package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/config"
)

type AITab struct {
	CRS2Enabled  bool
	CRS2Endpoint string
	CRS2Token    string
	PayGOClaude  bool
	PayGOGemini  bool
	PayGOCodex   bool
}

func NewAITab(cfg *config.Config) *AITab {
	return &AITab{
		CRS2Enabled:  cfg.AI.CRS2Enabled,
		CRS2Endpoint: cfg.AI.CRS2Endpoint,
		CRS2Token:    cfg.AI.CRS2Token,
		PayGOClaude:  cfg.AI.PayGOClaude,
		PayGOGemini:  cfg.AI.PayGOGemini,
		PayGOCodex:   cfg.AI.PayGOCodex,
	}
}

func (t *AITab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enable CRS2").
				Description("Enable AI service integration").
				Value(&t.CRS2Enabled),
		),
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
		).WithHideFunc(func() bool { return !t.CRS2Enabled }),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Claude").
				Description("Enable Claude PayGO billing").
				Value(&t.PayGOClaude),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Gemini").
				Description("Enable Gemini PayGO billing").
				Value(&t.PayGOGemini),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Codex").
				Description("Enable Codex PayGO billing").
				Value(&t.PayGOCodex),
		),
	)
}

func (t *AITab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enable CRS2").
				Value(&t.CRS2Enabled),
		).Title("AI"),
		huh.NewGroup(
			huh.NewInput().
				Title("CRS2 Endpoint").
				Value(&t.CRS2Endpoint),
			huh.NewInput().
				Title("CRS2 Token").
				EchoMode(huh.EchoModePassword).
				Value(&t.CRS2Token),
		).Title("AI — CRS2").WithHideFunc(func() bool { return !t.CRS2Enabled }),
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Claude").
				Value(&t.PayGOClaude),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Gemini").
				Value(&t.PayGOGemini),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("PayGO Codex").
				Value(&t.PayGOCodex),
		).Title("AI — PayGO"),
	}
}

func (t *AITab) View() string {
	s := fmt.Sprintf("  CRS2: %s", boolDisplay(t.CRS2Enabled))
	if t.CRS2Enabled {
		s += fmt.Sprintf("\n  CRS2 Endpoint: %s\n  CRS2 Token:    %s",
			displayValue(t.CRS2Endpoint), maskValue(t.CRS2Token))
	}
	s += fmt.Sprintf("\n\n  PayGO Claude:  %s\n  PayGO Gemini:  %s\n  PayGO Codex:   %s",
		boolDisplay(t.PayGOClaude), boolDisplay(t.PayGOGemini), boolDisplay(t.PayGOCodex))
	return s
}

func (t *AITab) Apply(cfg *config.Config) {
	cfg.AI.CRS2Enabled = t.CRS2Enabled
	cfg.AI.CRS2Endpoint = t.CRS2Endpoint
	cfg.AI.CRS2Token = t.CRS2Token
	cfg.AI.PayGOClaude = t.PayGOClaude
	cfg.AI.PayGOGemini = t.PayGOGemini
	cfg.AI.PayGOCodex = t.PayGOCodex
}

func (t *AITab) Edit() error {
	return RunFormAsEdit(t.Form())
}
