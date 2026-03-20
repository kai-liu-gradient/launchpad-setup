package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/tui/components"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type ExitReason int

const (
	ExitQuit ExitReason = iota
	ExitEdit
)

// SettingsTab is the interface each settings group implements.
type SettingsTab interface {
	View() string
	Form() *huh.Form
	Apply(s *settingsmod.Settings)
}

type Model struct {
	menu       components.SideMenu
	settings   *settingsmod.Settings
	Tabs       []SettingsTab
	exitReason ExitReason
}

func New(s *settingsmod.Settings, activeItem int) Model {
	items := []components.MenuItem{
		{Name: "Registration"},
		{Name: "Plans"},
		{Name: "Credentials"},
		{Name: "Projects"},
		{Name: "Authentication"},
		{Name: "Welcome Credits"},
		{Name: "Menu"},
		{Name: "YAML Builder"},
		{Name: "AI Marketplace"},
		{Name: "Ops Report"},
		{Name: "Email Suppress"},
		{Name: "Notices"},
	}

	tabs := []SettingsTab{
		// Will be populated when all tabs are implemented
	}

	if activeItem < 0 || activeItem >= len(items) {
		activeItem = 0
	}

	return Model{
		menu:     components.SideMenu{Items: items, Active: activeItem},
		settings: s,
		Tabs:     tabs,
	}
}

func (m Model) ExitReason() ExitReason { return m.exitReason }
func (m Model) ActiveTab() int         { return m.menu.Active }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.menu.Prev()
		case "down", "j":
			m.menu.Next()
		case "q", "ctrl+c":
			m.exitReason = ExitQuit
			return m, tea.Quit
		case "enter":
			if m.menu.Active < len(m.Tabs) {
				m.exitReason = ExitEdit
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Settings")

	menuStr := m.menu.Render()

	var content string
	if m.menu.Active < len(m.Tabs) {
		content = m.Tabs[m.menu.Active].View()
	} else {
		content = components.MutedStyle.Render("  (not yet implemented)")
	}

	menuBox := lipgloss.NewStyle().
		Width(18).
		MarginRight(2).
		Render(menuStr)

	contentLines := strings.Split(content, "\n")
	contentStr := strings.Join(contentLines, "\n")
	contentBox := lipgloss.NewStyle().Render(contentStr)

	body := lipgloss.JoinHorizontal(lipgloss.Top, menuBox, contentBox)

	hint := "↑/↓ navigate · enter edit · q quit"
	statusBar := components.MutedStyle.Render(hint)

	return fmt.Sprintf("\n%s\n\n%s\n\n%s\n", header, body, statusBar)
}
