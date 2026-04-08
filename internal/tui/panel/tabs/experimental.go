package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/config"
)

type ExperimentalTab struct {
	UseBunRuntime bool
	DebugMode     bool
}

func NewExperimentalTab(cfg *config.Config) *ExperimentalTab {
	return &ExperimentalTab{
		UseBunRuntime: cfg.Experimental.UseBunRuntime,
		DebugMode:     cfg.Experimental.DebugMode,
	}
}

func (t *ExperimentalTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Use Bun Runtime").
				Description("Use Bun instead of Node.js for bootstrap (default: false)").
				Value(&t.UseBunRuntime),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Debug Mode").
				Description("Enable bootstrap debug logging (default: false)").
				Value(&t.DebugMode),
		),
	)
}

func (t *ExperimentalTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Use Bun Runtime").
				Description("Use Bun instead of Node.js for bootstrap").
				Value(&t.UseBunRuntime),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Debug Mode").
				Description("Enable bootstrap debug logging").
				Value(&t.DebugMode),
		).Title("Experimental"),
	}
}

func (t *ExperimentalTab) View() string {
	return fmt.Sprintf(
		"  Bun Runtime: %s\n  Debug Mode:  %s",
		boolDisplay(t.UseBunRuntime), boolDisplay(t.DebugMode))
}

func (t *ExperimentalTab) Apply(cfg *config.Config) {
	cfg.Experimental.UseBunRuntime = t.UseBunRuntime
	cfg.Experimental.DebugMode = t.DebugMode
}

func (t *ExperimentalTab) Edit() error {
	return RunFormAsEdit(t.Form())
}

func boolDisplay(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
