package panel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/components"
	"github.com/gradient8/launchpad/internal/tui/panel/tabs"
)

// ExitReason indicates why the panel exited.
type ExitReason int

const (
	ExitQuit  ExitReason = iota // user pressed q
	ExitApply                   // user pressed a
	ExitEdit                    // user pressed enter to edit a tab
)

// EditableTab is a tab that can show a summary View and launch a Form for editing.
type EditableTab interface {
	View() string
	Edit() error
	Apply(cfg *config.Config)
}

type Model struct {
	menu       SideMenu
	cfg        *config.Config
	Tabs       []EditableTab
	overview   *tabs.OverviewTab
	exitReason ExitReason
	message    string
}

func New(cfg *config.Config, activeItem int) Model {
	items := []MenuItem{
		{Name: "Overview"},
		{Name: "Images"},
		{Name: "SSL"},
		{Name: "Database"},
		{Name: "SMTP"},
		{Name: "SSO"},
		{Name: "Storage"},
		{Name: "Telegram"},
		{Name: "AI"},
		{Name: "Experimental"},
	}

	editTabs := []EditableTab{
		tabs.NewImagesTab(cfg),
		tabs.NewSSLTab(cfg),
		tabs.NewDatabaseTab(cfg),
		tabs.NewSMTPTab(cfg),
		tabs.NewSSOTab(cfg),
		tabs.NewStorageTab(cfg),
		tabs.NewTelegramTab(cfg),
		tabs.NewAITab(cfg),
		tabs.NewExperimentalTab(cfg),
	}

	if activeItem < 0 || activeItem >= len(items) {
		activeItem = 0
	}

	return Model{
		menu:     SideMenu{Items: items, Active: activeItem},
		cfg:      cfg,
		Tabs:     editTabs,
		overview: tabs.NewOverviewTab(cfg),
	}
}

func (m Model) ExitReason() ExitReason { return m.exitReason }
func (m Model) ActiveTab() int         { return m.menu.Active }
func (m Model) EditTabIndex() int      { return m.menu.Active - 1 }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.menu.Prev()
			m.message = ""
		case "down", "j":
			m.menu.Next()
			m.message = ""
		case "q", "ctrl+c":
			m.exitReason = ExitQuit
			return m, tea.Quit
		case "enter":
			if m.menu.Active == 0 {
				m.message = "Overview is read-only"
				return m, nil
			}
			m.exitReason = ExitEdit
			return m, tea.Quit
		case "a":
			m.exitReason = ExitApply
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Configure")

	// Left: side menu
	menuStr := m.menu.Render()

	// Right: content
	var content string
	if m.menu.Active == 0 {
		content = m.overview.View()
	} else {
		tabIdx := m.menu.Active - 1
		content = m.Tabs[tabIdx].View()
	}

	if m.message != "" {
		content += "\n\n  " + components.SuccessStyle.Render(m.message)
	}

	// Layout: menu on left, content on right
	menuBox := lipgloss.NewStyle().
		Width(18).
		MarginRight(2).
		Render(menuStr)

	// Pad content lines for alignment
	contentLines := strings.Split(content, "\n")
	contentStr := strings.Join(contentLines, "\n")
	contentBox := lipgloss.NewStyle().Render(contentStr)

	body := lipgloss.JoinHorizontal(lipgloss.Top, menuBox, contentBox)

	hint := "↑/↓ navigate · enter edit · a apply · q quit"
	statusBar := components.MutedStyle.Render(hint)

	return fmt.Sprintf("\n%s\n\n%s\n\n%s\n", header, body, statusBar)
}
