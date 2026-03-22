package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "restart [service]",
		Short: "Restart AniLaunchpad services",
		Long:  "Restart all services, or a specific service if provided.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var service string
			if len(args) > 0 {
				service = args[0]
			}
			return runRestart(flagDir, service)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runRestart(dir, service string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	if _, err := config.Load(cfgPath); err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	composePath := filepath.Join(dir, "generated", "docker-compose.yml")

	args := []string{"compose", "-f", composePath, "restart"}
	if service != "" {
		args = append(args, service)
		fmt.Printf("Restarting service: %s\n", service)
	} else {
		fmt.Println("Restarting all services...")
	}

	c := exec.Command("docker", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("docker compose restart failed: %w", err)
	}

	fmt.Println("Restart completed.")
	return nil
}
