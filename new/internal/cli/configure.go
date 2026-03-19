package cli

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/panel"
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

	panelModel := panel.New(cfg)
	p := tea.NewProgram(panelModel)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("configure panel failed: %w", err)
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
