package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/gradient8/launchpad/internal/tui/components"
	"github.com/gradient8/launchpad/internal/tui/progress"
	"github.com/gradient8/launchpad/internal/tui/wizard"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		flagCustom bool
		flagConfig string
		flagResume bool
		flagDir    string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install AniLaunchpad",
		Long:  "Run the AniLaunchpad installation wizard and deploy all services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(flagCustom, flagConfig, flagResume, flagDir)
		},
	}

	cmd.Flags().BoolVar(&flagCustom, "custom", false, "Run the full custom setup wizard")
	cmd.Flags().StringVar(&flagConfig, "config", "", "Load configuration from FILE and skip wizard")
	cmd.Flags().BoolVar(&flagResume, "resume", false, "Resume an interrupted installation from DIR")
	cmd.Flags().StringVar(&flagDir, "dir", ".", "Working directory for the installation")

	return cmd
}

func runInstall(custom bool, configFile string, resume bool, dir string) error {
	var cfg *config.Config
	var sec *secrets.Secrets

	outputDir := filepath.Join(dir, "generated")

	// Check for existing installation (skip for --resume which handles this itself)
	if !resume {
		cfgPath := filepath.Join(dir, ".setup.yaml")
		if _, err := os.Stat(cfgPath); err == nil {
			existingCfg, loadErr := config.Load(cfgPath)
			if loadErr == nil {
				fmt.Printf("An existing installation was detected (domain: %s).\n", existingCfg.Domain)
				fmt.Println("Reinstalling will remove all generated files, services, and network configuration,")
				fmt.Println("and create a completely new environment.")
				fmt.Print("Do you want to reinstall? [y/N] ")

				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					fmt.Println("Installation cancelled.")
					return nil
				}

				fmt.Println("Cleaning up existing installation...")
				cleanupExisting(existingCfg, dir, outputDir)
				fmt.Print("Cleanup complete. Proceeding with fresh install.\n\n")
			}
		}
	}

	switch {
	case configFile != "":
		// Load config from file, validate, skip wizard
		loaded, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config from %s: %w", configFile, err)
		}
		if err := config.Validate(loaded); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
		cfg = loaded

		// Generate fresh secrets
		generated, err := secrets.Generate()
		if err != nil {
			return fmt.Errorf("generating secrets: %w", err)
		}
		sec = generated

	case resume:
		// Load existing .setup.yaml from dir
		cfgPath := filepath.Join(dir, ".setup.yaml")
		loaded, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading existing config from %s: %w", cfgPath, err)
		}
		cfg = loaded

		// Load existing secrets
		secPath := filepath.Join(dir, ".secrets.yaml")
		loadedSec, err := secrets.Load(secPath)
		if err != nil {
			return fmt.Errorf("loading secrets from %s: %w", secPath, err)
		}
		sec = loadedSec

		// Jump straight to deploy
		return deployWithProgress(cfg, sec, dir, outputDir)

	default:
		// Run wizard (express or custom)
		mode := wizard.Express
		if custom {
			mode = wizard.Custom
		}

		var wizardResult *config.Config
		for {
			result, err := runWizard(mode)
			if err != nil {
				return err
			}
			if result == nil {
				fmt.Println("Installation cancelled.")
				os.Exit(0)
			}

			// Domain preview: let user confirm or customize generated domains
			preview, err := wizard.RunDomainPreview(result.Domain, result.Subdomain, result.ProjectDomain, result.AdminEmail)
			if err != nil {
				return fmt.Errorf("domain preview: %w", err)
			}
			if preview.GoBack {
				continue // re-run wizard
			}
			result.Subdomain = preview.Subdomain
			result.GiteaSubdomain = preview.GiteaSubdomain
			wizardResult = result
			break
		}

		cfg = wizardResult

		// Generate secrets
		generated, err := secrets.Generate()
		if err != nil {
			return fmt.Errorf("generating secrets: %w", err)
		}
		sec = generated
	}

	// Save config and secrets
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	cfgPath := filepath.Join(dir, ".setup.yaml")
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	secPath := filepath.Join(dir, ".secrets.yaml")
	if err := secrets.Save(sec, secPath); err != nil {
		return fmt.Errorf("saving secrets: %w", err)
	}

	// Compute derived values and render templates
	derived := template.ComputeDerived(cfg, sec)
	runtime := &template.RuntimeValues{}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
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

	return deployWithProgress(cfg, sec, dir, outputDir)
}

func deployWithProgress(cfg *config.Config, sec *secrets.Secrets, dir, outputDir string) error {
	events := make(chan engine.StepEvent, 100)
	eng := engine.New(cfg, sec, outputDir, events)
	steps := eng.BuildStepList()

	// Collect step names for the progress TUI
	stepNames := make([]string, len(steps))
	for i, s := range steps {
		stepNames[i] = s.Name
	}

	progressModel := progress.New(stepNames, events)
	p := tea.NewProgram(progressModel)

	// Run deploy in goroutine, sending events
	var deployErr error
	go func() {
		deployErr = eng.Deploy(context.Background())
		close(events)
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("progress TUI failed: %w", err)
	}

	if deployErr != nil {
		return nil // error already shown in TUI
	}

	// Print access information
	subdomain := cfg.Subdomain
	if subdomain == "" {
		subdomain = "launchpad"
	}
	giteaSub := cfg.GiteaSubdomain
	if giteaSub == "" {
		giteaSub = subdomain + "-gitea"
	}
	dashboardURL := fmt.Sprintf("https://%s.%s", subdomain, cfg.Domain)
	giteaURL := fmt.Sprintf("https://%s.%s", giteaSub, cfg.Domain)
	adminUser := cfg.AdminEmail

	fmt.Println()
	fmt.Println("  AniLaunchpad is ready!")
	fmt.Println()
	fmt.Printf("  Dashboard : %s\n", dashboardURL)
	fmt.Printf("  Gitea     : %s\n", giteaURL)
	fmt.Println()
	fmt.Println("  ⚠️  Admin Credentials (SAVE THESE!):")
	fmt.Printf("  Admin     : %s\n", adminUser)
	fmt.Printf("  Password  : %s\n", sec.AdminPassword)
	fmt.Println()
	fmt.Println("  This password is auto-generated and will NOT be shown again.")
	fmt.Println("  To retrieve later: cat .secrets.yaml | grep admin_password")
	fmt.Println()

	return nil
}

// runWizard runs the interactive install wizard and returns the final config.
// Returns nil if the user aborted.
func runWizard(mode wizard.Mode) (*config.Config, error) {
	wm := wizard.New(mode)

	for {
		p := tea.NewProgram(wm, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return nil, fmt.Errorf("wizard failed: %w", err)
		}

		wm = finalModel.(wizard.Model)

		if wm.Aborted() {
			return nil, nil
		}
		if wm.Done() {
			return wm.Result(), nil
		}

		// ExitEdit: user pressed enter on a tab — launch its form
		if wm.ExitReason() == wizard.ExitEdit {
			tabIdx := wm.ActiveTab()
			tabList := wm.Tabs()
			if tabIdx < len(tabList) {
				tab := tabList[tabIdx]
				form := tab.Form()
				form.WithKeyMap(components.FormKeyMap()).WithProgramOptions(tea.WithAltScreen())
				if err := form.Run(); err == nil {
					// Tab values updated in place
				}
			}
			// Loop back to panel with same active tab
			continue
		}

		break
	}
	return nil, nil
}

// cleanupExisting removes all artifacts from a previous installation so the
// next install starts from a clean slate.
func cleanupExisting(cfg *config.Config, dir, outputDir string) {
	ctx := context.Background()
	composePath := filepath.Join(outputDir, "docker-compose.yml")

	// 1. Stop all Docker services
	if _, err := os.Stat(composePath); err == nil {
		fmt.Println("  Stopping services...")
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "down", "-v")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: docker compose down failed: %v\n", err)
		}
	}

	// 2. Remove /etc/hosts entries (using marker-based cleanup)
	fmt.Println("  Removing /etc/hosts entries...")
	if err := engine.RemoveHostEntries(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: removing /etc/hosts entries: %v\n", err)
	}

	// 3. Remove dnsmasq config
	fmt.Println("  Removing dnsmasq config...")
	if err := engine.RemoveDnsmasq(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: removing dnsmasq config: %v\n", err)
	}

	// 4. Remove CoreDNS custom config
	fmt.Println("  Removing CoreDNS custom config...")
	_ = exec.CommandContext(ctx, "kubectl", "delete", "configmap", "coredns-custom", "-n", "kube-system", "--ignore-not-found").Run()

	// 5. Restart CoreDNS to pick up removal
	_ = exec.CommandContext(ctx, "kubectl", "rollout", "restart", "deploy/coredns", "-n", "kube-system").Run()

	// 6. Remove generated directory
	fmt.Println("  Removing generated files...")
	if err := os.RemoveAll(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: removing generated directory: %v\n", err)
	}

	// 7. Remove .setup.yaml and .secrets.yaml
	os.Remove(filepath.Join(dir, ".setup.yaml"))
	os.Remove(filepath.Join(dir, ".secrets.yaml"))
}
