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

func newUpgradeCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade AniLaunchpad to a new version",
		Long:  "Pull new images and restart services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(flagDir)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runUpgrade(dir string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	// Load secrets for template rendering
	secPath := filepath.Join(dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return fmt.Errorf("loading secrets from %s: %w", secPath, err)
	}

	// Re-render templates with updated config
	outputDir := filepath.Join(dir, "generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	derived := template.ComputeDerived(cfg, sec)

	runtimePath := filepath.Join(dir, ".runtime.yaml")
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
