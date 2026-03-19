package engine

import (
	"context"
	"fmt"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

// StepStatus represents the state of a deploy step.
type StepStatus int

const (
	Pending StepStatus = iota
	Running
	Done
	Failed
)

// StepEvent is sent over the notify channel to update the TUI on progress.
type StepEvent struct {
	Step   string
	Status StepStatus
	Detail string
	Err    error
}

// Step is a named unit of work in the deployment pipeline.
type Step struct {
	Name string
	Fn   func(context.Context) error
}

// Engine orchestrates the deployment pipeline.
type Engine struct {
	cfg    *config.Config
	sec    *secrets.Secrets
	output string
	notify chan<- StepEvent
}

// New creates a new Engine. notify may be nil if no progress updates needed.
func New(cfg *config.Config, sec *secrets.Secrets, output string, notify chan<- StepEvent) *Engine {
	return &Engine{
		cfg:    cfg,
		sec:    sec,
		output: output,
		notify: notify,
	}
}

// Deploy runs all steps sequentially, sending events through the notify channel.
func (e *Engine) Deploy(ctx context.Context) error {
	steps := e.BuildStepList()
	for _, step := range steps {
		e.send(StepEvent{Step: step.Name, Status: Running})
		if err := step.Fn(ctx); err != nil {
			e.send(StepEvent{Step: step.Name, Status: Failed, Err: err})
			return fmt.Errorf("step %q failed: %w", step.Name, err)
		}
		e.send(StepEvent{Step: step.Name, Status: Done})
	}
	return nil
}

func (e *Engine) send(ev StepEvent) {
	if e.notify != nil {
		e.notify <- ev
	}
}
