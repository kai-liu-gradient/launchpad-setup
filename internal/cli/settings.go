package cli

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/settings"
	settingsui "github.com/gradient8/launchpad/internal/tui/settings"
	"github.com/spf13/cobra"
)

func newSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "settings",
		Short: "Edit runtime settings (hot-reloadable)",
		Long:  "Open the interactive settings editor. Changes take effect immediately without restart.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettings(".")
		},
	}
}

func runSettings(dir string) error {
	settingsPath := filepath.Join(dir, "generated", "launchpad", "config", "settings.yml")

	s, err := settings.Load(settingsPath)
	if err != nil {
		return fmt.Errorf("cannot load settings.yml: %w\nRun initial deployment first if the file does not exist.", err)
	}

	activeTab := 0
	for {
		panelModel := settingsui.New(s, activeTab)
		p := tea.NewProgram(panelModel, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return fmt.Errorf("settings panel failed: %w", err)
		}

		m := result.(settingsui.Model)
		activeTab = m.ActiveTab()

		switch m.ExitReason() {
		case settingsui.ExitQuit:
			return nil

		case settingsui.ExitEdit:
			tabIdx := m.ActiveTab()
			if tabIdx >= len(m.Tabs) {
				continue
			}
			tab := m.Tabs[tabIdx]
			if err := tab.Edit(); err == nil {
				tab.Apply(s)
				if saveErr := settings.Save(s, settingsPath); saveErr != nil {
					return fmt.Errorf("saving settings: %w", saveErr)
				}
			}
			continue
		}
		break
	}
	return nil
}
