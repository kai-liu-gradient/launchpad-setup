package engine

import (
	"context"
	"fmt"

	"github.com/gradient8/launchpad/internal/config"
)

// BuildReconfigureSteps compares old and new configs and returns the steps
// needed to apply the changes: a re-render step followed by a restart step
// for each affected service. Returns an empty slice when there are no changes.
func (e *Engine) BuildReconfigureSteps(old, new *config.Config) []Step {
	diff := config.Diff(old, new)
	if len(diff.Changes) == 0 {
		return nil
	}

	var steps []Step

	steps = append(steps, Step{
		Name: "Re-rendering templates",
		Fn:   func(ctx context.Context) error { return nil },
	})

	for _, svc := range diff.AffectedServices() {
		svc := svc // capture loop variable
		steps = append(steps, Step{
			Name: fmt.Sprintf("Restarting %s", svc),
			Fn:   func(ctx context.Context) error { return nil },
		})
	}

	return steps
}
