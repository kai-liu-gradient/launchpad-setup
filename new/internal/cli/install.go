package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
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

		wizardModel := wizard.New(mode)
		p := tea.NewProgram(wizardModel)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("wizard failed: %w", err)
		}

		wm, ok := finalModel.(wizard.Model)
		if !ok {
			return fmt.Errorf("unexpected wizard model type")
		}
		if wm.Aborted() {
			fmt.Println("Installation cancelled.")
			os.Exit(0)
		}
		if !wm.Done() {
			return fmt.Errorf("wizard did not complete")
		}
		cfg = wm.Result()

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
	go func() {
		eng.Deploy(context.Background()) //nolint:errcheck
		close(events)
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("progress TUI failed: %w", err)
	}

	return nil
}
