package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/spf13/cobra"
)

type upgradeFlags struct {
	dir            string
	apiVersion     string
	uiVersion      string
	routerVersion  string
	gatewayVersion string
	giteaVersion   string
}

func newUpgradeCmd() *cobra.Command {
	f := &upgradeFlags{}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade AniLaunchpad to a new version",
		Long:  "Pull new images and restart services. Optionally specify per-service versions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(f)
		},
	}

	cmd.Flags().StringVar(&f.dir, "dir", ".", "Installation directory")
	cmd.Flags().StringVar(&f.apiVersion, "api-version", "", "API image version")
	cmd.Flags().StringVar(&f.uiVersion, "ui-version", "", "UI image version")
	cmd.Flags().StringVar(&f.routerVersion, "router-version", "", "Router image version")
	cmd.Flags().StringVar(&f.gatewayVersion, "gateway-version", "", "Gateway image version")
	cmd.Flags().StringVar(&f.giteaVersion, "gitea-version", "", "Gitea image version")

	return cmd
}

func runUpgrade(f *upgradeFlags) error {
	cfgPath := filepath.Join(f.dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	// Apply version flags if specified
	changed := false
	if f.apiVersion != "" {
		cfg.Images.APIVersion = f.apiVersion
		changed = true
	}
	if f.uiVersion != "" {
		cfg.Images.UIVersion = f.uiVersion
		changed = true
	}
	if f.routerVersion != "" {
		cfg.Images.RouterVersion = f.routerVersion
		changed = true
	}
	if f.gatewayVersion != "" {
		cfg.Images.GatewayVersion = f.gatewayVersion
		changed = true
	}
	if f.giteaVersion != "" {
		cfg.Images.GiteaVersion = f.giteaVersion
		changed = true
	}

	// Save updated config if versions were changed
	if changed {
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}

	// Load secrets for template rendering
	secPath := filepath.Join(f.dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return fmt.Errorf("loading secrets from %s: %w", secPath, err)
	}

	// Re-render templates with updated config
	outputDir := filepath.Join(f.dir, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	derived := template.ComputeDerived(cfg, sec)

	runtimePath := filepath.Join(f.dir, ".runtime.yaml")
	runtime, err := template.LoadRuntime(runtimePath)
	if err != nil {
		return fmt.Errorf("loading runtime values: %w", err)
	}

	ctx := &template.RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}
	if err := template.RenderAll(ctx, outputDir); err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	composePath := filepath.Join(outputDir, "docker-compose.yml")

	// Pull new images
	pullCmd := exec.Command("docker", "compose", "-f", composePath, "pull")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("docker compose pull failed: %w", err)
	}

	// Restart services with new images
	upCmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("docker compose up -d failed: %w", err)
	}

	fmt.Println("Upgrade completed successfully.")
	return nil
}
