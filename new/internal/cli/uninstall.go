package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gradient8/launchpad/internal/config"
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
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	// Confirmation prompt
	scope := "stop and remove all services"
	if all {
		scope = "COMPLETELY remove AniLaunchpad including K3s, host configuration, and install directory"
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
	downCmd := exec.Command("docker", "compose", "-f", composePath, "down")
	downCmd.Stdout = os.Stdout
	downCmd.Stderr = os.Stderr
	if err := downCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: docker compose down failed: %v\n", err)
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

	// Remove /etc/hosts entries for this domain
	if cfg.Domain != "" {
		fmt.Printf("Removing /etc/hosts entries for %s...\n", cfg.Domain)
		if err := removeHostsEntries(cfg.Domain, cfg.Subdomain); err != nil {
			fmt.Fprintf(os.Stderr, "warning: removing /etc/hosts entries failed: %v\n", err)
		}
	}

	// Remove dnsmasq config
	dnsmasqConf := fmt.Sprintf("/etc/dnsmasq.d/launchpad-%s.conf", cfg.Domain)
	if _, err := os.Stat(dnsmasqConf); err == nil {
		fmt.Printf("Removing dnsmasq config %s...\n", dnsmasqConf)
		if err := os.Remove(dnsmasqConf); err != nil {
			fmt.Fprintf(os.Stderr, "warning: removing dnsmasq config failed: %v\n", err)
		}
	}

	// Remove install directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	fmt.Printf("Removing install directory %s...\n", absDir)
	if err := os.RemoveAll(absDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: removing install directory failed: %v\n", err)
	}

	fmt.Println("Uninstall complete.")
	return nil
}

// removeHostsEntries removes lines matching the domain from /etc/hosts.
func removeHostsEntries(domain, subdomain string) error {
	hostsPath := "/etc/hosts"
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", hostsPath, err)
	}

	lines := strings.Split(string(data), "\n")
	var kept []string
	for _, line := range lines {
		if strings.Contains(line, domain) || strings.Contains(line, subdomain+"."+domain) {
			continue
		}
		kept = append(kept, line)
	}

	output := strings.Join(kept, "\n")
	if err := os.WriteFile(hostsPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", hostsPath, err)
	}

	return nil
}
