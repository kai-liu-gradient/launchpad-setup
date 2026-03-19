package panel

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type Model struct {
	tabBar   TabBar
	cfg      *config.Config
	original *config.Config
	dirty    bool
	quitting bool
}

func New(cfg *config.Config) Model {
	// Deep copy for comparison
	original := *cfg

	tabs := []Tab{
		{Name: "Overview"},
		{Name: "SSL"},
		{Name: "Database"},
		{Name: "SMTP"},
		{Name: "SSO"},
		{Name: "Storage"},
		{Name: "Telegram"},
		{Name: "AI"},
	}

	return Model{
		tabBar:   TabBar{Tabs: tabs, Active: 0},
		cfg:      cfg,
		original: &original,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "shift+tab":
			m.tabBar.Prev()
		case "right", "tab":
			m.tabBar.Next()
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "a":
			// Apply changes — placeholder
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Configure")
	tabs := m.tabBar.Render()

	// Tab content placeholder
	content := fmt.Sprintf("\n  [%s tab content]\n", m.tabBar.Tabs[m.tabBar.Active].Name)

	statusBar := components.MutedStyle.Render("←/→ switch tabs · a apply · q quit")

	return fmt.Sprintf("%s\n%s\n%s\n\n%s\n", header, tabs, content, statusBar)
}
