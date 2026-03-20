package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/gradient8/launchpad/internal/tui/panel/tabs"
	"github.com/spf13/cobra"
)

func newSetupTelegramCmd() *cobra.Command {
	var flagDir string
	var flagToken string
	var flagUsername string

	cmd := &cobra.Command{
		Use:   "setup-telegram",
		Short: "Configure Telegram bot settings",
		Long:  "Set Telegram bot token and username, re-render templates, and restart affected services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupTelegram(flagDir, flagToken, flagUsername)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")
	cmd.Flags().StringVar(&flagToken, "token", "", "Telegram bot token")
	cmd.Flags().StringVar(&flagUsername, "username", "", "Telegram bot username")

	return cmd
}

func runSetupTelegram(dir, token, username string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	secPath := filepath.Join(dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return fmt.Errorf("loading secrets: %w", err)
	}

	runtimePath := filepath.Join(dir, "generated", ".runtime.yaml")
	runtime, _ := template.LoadRuntime(runtimePath)

	// If flags provided, skip interactive UI
	if token != "" || username != "" {
		if token != "" {
			cfg.Telegram.BotToken = token
		}
		if username != "" {
			cfg.Telegram.BotUsername = username
		}
	} else {
		// Interactive: use the same TUI form as the configure panel
		tab := tabs.NewTelegramTab(cfg)
		form := tab.Form()
		if err := form.Run(); err != nil {
			return fmt.Errorf("telegram form cancelled: %w", err)
		}
		tab.Apply(cfg)
	}

	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}

	// Save config
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Println("✓ Configuration saved.")

	// Re-render templates
	outputDir := filepath.Join(dir, "generated")
	derived := template.ComputeDerived(cfg, sec)
	renderCtx := &template.RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}
	if err := template.RenderAll(renderCtx, outputDir); err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}
	fmt.Println("✓ Templates re-rendered.")

	// Restart api and gateway
	composePath := filepath.Join(outputDir, "docker-compose.yml")
	fmt.Println("Restarting api and gateway...")
	c := exec.Command("docker", "compose", "-f", composePath, "up", "-d", "--force-recreate", "api", "gateway")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("restarting services: %w", err)
	}

	fmt.Println("✓ Telegram configuration complete.")
	return nil
}
