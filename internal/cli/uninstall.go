package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var (
		flagAll bool
		flagDir string
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AniLaunchpad",
		Long:  "Stop and remove AniLaunchpad services. Use --all to also remove K3s and host configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(flagDir, flagAll)
		},
	}

	cmd.Flags().BoolVar(&flagAll, "all", false, "Fully remove K3s, dnsmasq config, /etc/hosts entries, and install directory")
	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runUninstall(dir string, all bool) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, _ := config.Load(cfgPath) // nil if not found — that's ok

	// Confirmation prompt
	scope := "stop and remove all services"
	if all {
		scope = "COMPLETELY remove AniLaunchpad including K3s, host configuration, and all generated files"
	}

	if cfg != nil && cfg.Domain != "" {
		fmt.Printf("Installation found: %s\n", cfg.Domain)
	} else {
		fmt.Println("No .setup.yaml found — will clean up any remaining artifacts.")
	}
	fmt.Printf("This will %s.\n", scope)
	fmt.Print("Are you sure? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Uninstall cancelled.")
		return nil
	}

	composePath := filepath.Join(dir, "generated", "docker-compose.yml")

	// Basic: docker compose down
	if _, err := os.Stat(composePath); err == nil {
		fmt.Println("Stopping services...")
		downCmd := exec.Command("docker", "compose", "-f", composePath, "down", "-v")
		downCmd.Stdout = os.Stdout
		downCmd.Stderr = os.Stderr
		if err := downCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: docker compose down failed: %v\n", err)
		}
	}

	if !all {
		fmt.Println("Services stopped and removed.")
		return nil
	}

	// Full uninstall: remove K3s
	k3sUninstall := "/usr/local/bin/k3s-uninstall.sh"
	if _, err := os.Stat(k3sUninstall); err == nil {
		fmt.Println("Removing K3s...")
		k3sCmd := exec.Command(k3sUninstall)
		k3sCmd.Stdout = os.Stdout
		k3sCmd.Stderr = os.Stderr
		if err := k3sCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: k3s uninstall failed: %v\n", err)
		}
	}

	// Remove /etc/hosts entries (marker-based, works without config)
	fmt.Println("Removing /etc/hosts entries...")
	if err := engine.RemoveHostEntries(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: removing /etc/hosts entries: %v\n", err)
	}

	// Remove dnsmasq config
	fmt.Println("Removing dnsmasq config...")
	if err := engine.RemoveDnsmasq(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: removing dnsmasq config: %v\n", err)
	}

	// Remove CoreDNS custom config
	fmt.Println("Removing CoreDNS custom config...")
	exec.Command("kubectl", "delete", "configmap", "coredns-custom", "-n", "kube-system", "--ignore-not-found").Run() //nolint:errcheck
	exec.Command("kubectl", "rollout", "restart", "deploy/coredns", "-n", "kube-system").Run()                        //nolint:errcheck

	// Remove launchpad files from install directory
	absDir, _ := filepath.Abs(dir)
	fmt.Printf("Removing launchpad files from %s...\n", absDir)
	for _, name := range []string{".setup.yaml", ".secrets.yaml", "generated"} {
		target := filepath.Join(absDir, name)
		if err := os.RemoveAll(target); err != nil {
			fmt.Fprintf(os.Stderr, "warning: removing %s failed: %v\n", target, err)
		}
	}

	fmt.Println("Uninstall complete.")
	return nil
}
