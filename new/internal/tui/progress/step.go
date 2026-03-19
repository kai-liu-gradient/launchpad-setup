package progress

import (
	"fmt"
	"time"

	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type StepView struct {
	Name     string
	Status   engine.StepStatus
	Duration time.Duration
	Detail   string
}

func (s StepView) Render() string {
	icon := components.StepPending
	switch s.Status {
	case engine.Done:
		icon = components.StepDone
	case engine.Running:
		icon = components.StepRunning
	case engine.Failed:
		icon = components.StepFailed
	}

	line := fmt.Sprintf("%s %s", icon, s.Name)
	if s.Status == engine.Done {
		line += components.MutedStyle.Render(fmt.Sprintf(" %s", s.Duration.Round(time.Second)))
	}
	if s.Status == engine.Running && s.Detail != "" {
		line += components.MutedStyle.Render(fmt.Sprintf(" %s", s.Detail))
	}
	if s.Status == engine.Failed {
		line += components.ErrorStyle.Render(fmt.Sprintf(" %s", s.Detail))
	}
	return line
}
