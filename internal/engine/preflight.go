package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/gradient8/launchpad/internal/config"
)

// ToolCheck describes a single binary that must be present before deployment.
type ToolCheck struct {
	Name      string
	Required  bool
	CheckCmd  string
	CheckArgs []string
}

// RequiredTools returns the list of ToolChecks derived from cfg. Always
// requires docker and curl. Builtin K8s mode additionally requires helm and
// kubectl; external K8s mode requires only kubectl.
func RequiredTools(cfg *config.Config) []ToolCheck {
	checks := []ToolCheck{
		{Name: "docker", Required: true, CheckCmd: "docker", CheckArgs: []string{"--version"}},
		{Name: "docker compose", Required: true, CheckCmd: "docker", CheckArgs: []string{"compose", "version"}},
		{Name: "curl", Required: true, CheckCmd: "curl", CheckArgs: []string{"--version"}},
	}

	// helm is needed in both modes (builtin uses it for ingress-nginx, kyverno).
	checks = append(checks,
		ToolCheck{Name: "helm", Required: true, CheckCmd: "helm", CheckArgs: []string{"version", "--short"}},
	)

	// External K8s mode additionally requires kubectl (builtin gets it from K3s).
	if cfg.Kubernetes.Mode != "builtin" {
		checks = append(checks,
			ToolCheck{Name: "kubectl", Required: true, CheckCmd: "kubectl", CheckArgs: []string{"version", "--client"}},
		)
	}

	return checks
}

// RunPreflightChecks executes each ToolCheck returned by RequiredTools and
// returns the first failure with a helpful diagnostic message.
func RunPreflightChecks(cfg *config.Config) error {
	for _, tc := range RequiredTools(cfg) {
		_, err := RunWithOutput(context.Background(), tc.Name, 10*time.Second, tc.CheckCmd, tc.CheckArgs...)
		if err != nil {
			return fmt.Errorf("pre-flight check failed for %q: %w\nPlease install %q and ensure it is on your PATH", tc.Name, err, tc.Name)
		}
	}
	return nil
}
