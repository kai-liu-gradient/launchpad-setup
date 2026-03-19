package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of AniLaunchpad services",
		Long:  "Display the current status of all running AniLaunchpad services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(flagDir)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runStatus(dir string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	if _, err := config.Load(cfgPath); err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	composePath := filepath.Join(dir, "generated", "docker-compose.yml")
	args := []string{"compose", "-f", composePath, "ps", "--format", "table"}

	c := exec.Command("docker", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("docker compose ps failed: %w", err)
	}

	return nil
}
