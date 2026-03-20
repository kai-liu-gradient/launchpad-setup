package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/gradient8/launchpad/internal/tui/components"
	"github.com/gradient8/launchpad/internal/tui/panel"
	"github.com/gradient8/launchpad/internal/tui/progress"
	"github.com/spf13/cobra"
)

func newConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure a running AniLaunchpad installation",
		Long:  "Open the interactive configuration panel to modify a running installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(".")
		},
	}

	cmd.AddCommand(newConfigureExportCmd())

	return cmd
}

func runConfigure(dir string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	secPath := filepath.Join(dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return fmt.Errorf("loading secrets: %w", err)
	}

	runtimePath := filepath.Join(dir, "generated", ".runtime.yaml")
	runtime, _ := template.LoadRuntime(runtimePath)

	activeTab := 0
	for {
		panelModel := panel.New(cfg, activeTab)
		p := tea.NewProgram(panelModel, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return fmt.Errorf("configure panel failed: %w", err)
		}

		m := result.(panel.Model)
		activeTab = m.ActiveTab()

		switch m.ExitReason() {
		case panel.ExitQuit:
			fmt.Println("Configuration cancelled.")
			return nil

		case panel.ExitEdit:
			tabIdx := m.EditTabIndex()
			tab := m.Tabs[tabIdx]
			form := tab.Form()
			form.WithKeyMap(components.FormKeyMap()).WithProgramOptions(tea.WithAltScreen())
			if err := form.Run(); err == nil {
				tab.Apply(cfg)
			}
			// Loop back to panel with same active tab
			continue

		case panel.ExitApply:
			// Fall through to save/render/restart
		}
		break
	}

	// Run apply steps with progress UI
	stepNames := []string{
		"Saving configuration",
		"Rendering templates",
		"Restarting services",
	}

	eventCh := make(chan engine.StepEvent, 16)
	outputDir := filepath.Join(dir, "generated")

	go func() {
		defer close(eventCh)

		// Step 1: Save config
		eventCh <- engine.StepEvent{Step: stepNames[0], Status: engine.Running}
		if err := config.Save(cfg, cfgPath); err != nil {
			eventCh <- engine.StepEvent{Step: stepNames[0], Status: engine.Failed, Err: err}
			return
		}
		eventCh <- engine.StepEvent{Step: stepNames[0], Status: engine.Done}

		// Step 2: Render templates
		eventCh <- engine.StepEvent{Step: stepNames[1], Status: engine.Running}
		derived := template.ComputeDerived(cfg, sec)
		renderCtx := &template.RenderContext{
			Config:  cfg,
			Secrets: sec,
			Derived: derived,
			Runtime: runtime,
		}
		if err := template.RenderAll(renderCtx, outputDir); err != nil {
			eventCh <- engine.StepEvent{Step: stepNames[1], Status: engine.Failed, Err: err}
			return
		}
		eventCh <- engine.StepEvent{Step: stepNames[1], Status: engine.Done}

		// Step 3: Restart services
		eventCh <- engine.StepEvent{Step: stepNames[2], Status: engine.Running}
		composePath := filepath.Join(outputDir, "docker-compose.yml")
		c := exec.Command("docker", "compose", "-f", composePath, "up", "-d", "--force-recreate")
		var stderr bytes.Buffer
		c.Stderr = &stderr
		if err := c.Run(); err != nil {
			eventCh <- engine.StepEvent{Step: stepNames[2], Status: engine.Failed,
				Err: fmt.Errorf("%w: %s", err, stderr.String())}
			return
		}
		eventCh <- engine.StepEvent{Step: stepNames[2], Status: engine.Done}
	}()

	progressModel := progress.New(stepNames, eventCh)
	p := tea.NewProgram(progressModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("progress display failed: %w", err)
	}

	return nil
}

func newConfigureExportCmd() *cobra.Command {
	var flagOutput string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export current configuration to a file",
		Long:  "Load the current .setup.yaml and write it to a specified output file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigureExport(".", flagOutput)
		},
	}

	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output file path (required)")
	cmd.MarkFlagRequired("output") //nolint:errcheck

	return cmd
}

func runConfigureExport(dir, outputFile string) error {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	if err := config.Save(cfg, outputFile); err != nil {
		return fmt.Errorf("saving config to %s: %w", outputFile, err)
	}

	fmt.Printf("Configuration exported to %s\n", outputFile)
	return nil
}
